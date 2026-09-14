// Package image provides high-performance on-demand image optimization and responsive components for GoKS.
package image

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Allowed responsive widths to prevent cache-flooding DoS attacks.
var DefaultAllowedWidths = []int{
	16, 32, 48, 64, 96, 128, 256, 384, 640, 750, 828, 1080, 1200, 1920, 2048, 3840,
}

// Config configures the image optimization service.
type Config struct {
	AppDir         string        // Root directory of the GoKS app
	CacheDir       string        // Directory to store optimized images (default: .goks/cache/images)
	MaxSourceBytes int64         // Maximum allowed size of source image in bytes (default: 20MB)
	MaxWidth       int           // Maximum allowable width in pixels (default: 4096)
	MaxHeight      int           // Maximum allowable height in pixels (default: 4096)
	AllowedDomains []string      // Whitelisted remote domains (empty = remote images disabled)
	AllowedWidths  []int         // Allowed target widths (default: DefaultAllowedWidths)
	EmbeddedPublic http.FileSystem // Optional embedded filesystem for standalone mode
}

// Optimizer handles on-demand image scaling, format conversion, and caching.
type Optimizer struct {
	cfg Config
	mu  sync.RWMutex
}

// NewOptimizer creates a new Optimizer with sensible security defaults.
func NewOptimizer(cfg Config) *Optimizer {
	if cfg.AppDir == "" {
		cfg.AppDir = "."
	}
	if cfg.CacheDir == "" {
		cfg.CacheDir = filepath.Join(cfg.AppDir, ".goks", "cache", "images")
	}
	if cfg.MaxSourceBytes <= 0 {
		cfg.MaxSourceBytes = 20 * 1024 * 1024 // 20 MB limit
	}
	if cfg.MaxWidth <= 0 {
		cfg.MaxWidth = 4096
	}
	if cfg.MaxHeight <= 0 {
		cfg.MaxHeight = 4096
	}
	if len(cfg.AllowedWidths) == 0 {
		cfg.AllowedWidths = DefaultAllowedWidths
	}
	_ = os.MkdirAll(cfg.CacheDir, 0755)
	return &Optimizer{cfg: cfg}
}

// SnapWidth snaps the requested width to the closest allowed responsive breakpoint.
func (o *Optimizer) SnapWidth(targetW int) int {
	if targetW <= 0 {
		return o.cfg.AllowedWidths[len(o.cfg.AllowedWidths)/2]
	}
	closest := o.cfg.AllowedWidths[0]
	minDiff := abs(closest - targetW)

	for _, w := range o.cfg.AllowedWidths {
		diff := abs(w - targetW)
		if diff < minDiff {
			minDiff = diff
			closest = w
		}
	}
	return closest
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// ServeHTTP implements http.Handler for the /__goks_image endpoint.
func (o *Optimizer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	q := r.URL.Query()
	rawURL := strings.TrimSpace(q.Get("url"))
	if rawURL == "" {
		http.Error(w, "Missing 'url' parameter", http.StatusBadRequest)
		return
	}

	// 1. Validate Width
	wStr := q.Get("w")
	targetW, err := strconv.Atoi(wStr)
	if err != nil || targetW <= 0 {
		http.Error(w, "Invalid 'w' (width) parameter", http.StatusBadRequest)
		return
	}
	snappedW := o.SnapWidth(targetW)

	// 2. Validate Quality
	quality := 75
	if qStr := q.Get("q"); qStr != "" {
		if parsedQ, err := strconv.Atoi(qStr); err == nil && parsedQ >= 1 && parsedQ <= 100 {
			quality = parsedQ
		}
	}

	// 3. Compute Cache Key / ETag
	cacheKeyInput := fmt.Sprintf("%s_w%d_q%d", rawURL, snappedW, quality)
	hasher := sha256.New()
	hasher.Write([]byte(cacheKeyInput))
	etag := hex.EncodeToString(hasher.Sum(nil))
	cacheFilePath := filepath.Join(o.cfg.CacheDir, etag+".webp")
	cachedFormat := "image/jpeg" // standard fallback

	// Check client If-None-Match header
	if r.Header.Get("If-None-Match") == `"`+etag+`"` || r.Header.Get("If-None-Match") == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	// 4. Check on-disk cache
	if cachedData, err := os.ReadFile(cacheFilePath); err == nil {
		w.Header().Set("Content-Type", cachedFormat)
		w.Header().Set("Content-Length", strconv.Itoa(len(cachedData)))
		w.Header().Set("ETag", `"`+etag+`"`)
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		w.WriteHeader(http.StatusOK)
		if r.Method != http.MethodHead {
			_, _ = w.Write(cachedData)
		}
		return
	}

	// 5. Fetch source image securely
	srcReader, contentType, err := o.fetchSourceImage(rawURL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer srcReader.Close()

	// 6. Decode config to protect against decompression bombs (CWE-409)
	limitedReader := io.LimitReader(srcReader, o.cfg.MaxSourceBytes)
	cfg, _, err := image.DecodeConfig(limitedReader)
	if err != nil {
		// If DecodeConfig consumed part of the stream, re-fetch or use buffer
		// We'll read the full stream safely into a limited buffer
		http.Error(w, "Invalid image format or corrupted image", http.StatusBadRequest)
		return
	}

	if cfg.Width > o.cfg.MaxWidth || cfg.Height > o.cfg.MaxHeight {
		http.Error(w, fmt.Sprintf("Image dimensions (%dx%d) exceed maximum allowed (%dx%d)",
			cfg.Width, cfg.Height, o.cfg.MaxWidth, o.cfg.MaxHeight), http.StatusBadRequest)
		return
	}

	// Re-open source for full decode
	srcReader2, _, err := o.fetchSourceImage(rawURL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer srcReader2.Close()

	srcImg, _, err := image.Decode(io.LimitReader(srcReader2, o.cfg.MaxSourceBytes))
	if err != nil {
		http.Error(w, "Failed to decode image: "+err.Error(), http.StatusBadRequest)
		return
	}

	// 7. Calculate target dimensions maintaining aspect ratio
	origBounds := srcImg.Bounds()
	origW := origBounds.Dx()
	origH := origBounds.Dy()
	if origW <= 0 || origH <= 0 {
		http.Error(w, "Invalid image dimensions", http.StatusBadRequest)
		return
	}

	// If source is already smaller or equal to target width, don't upscale
	finalW := snappedW
	if origW < finalW {
		finalW = origW
	}
	finalH := (origH * finalW) / origW
	if finalH < 1 {
		finalH = 1
	}

	// 8. High-Quality Bilinear Resizing
	resized := resizeBilinear(srcImg, finalW, finalH)

	// 9. Encode image (JPEG with specified quality or PNG for transparency)
	var encodeBuf strings.Builder
	_ = encodeBuf
	tmpFile, err := os.CreateTemp(o.cfg.CacheDir, "opt_*.tmp")
	if err != nil {
		http.Error(w, "Failed to create cache file", http.StatusInternalServerError)
		return
	}
	tmpPath := tmpFile.Name()
	defer func() {
		_ = tmpFile.Close()
		_ = os.Remove(tmpPath)
	}()

	isPNG := strings.Contains(contentType, "png") || hasAlpha(resized)
	outContentType := "image/jpeg"
	if isPNG {
		outContentType = "image/png"
		if err := png.Encode(tmpFile, resized); err != nil {
			http.Error(w, "Encoding failed: "+err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		opts := &jpeg.Options{Quality: quality}
		if err := jpeg.Encode(tmpFile, resized, opts); err != nil {
			http.Error(w, "Encoding failed: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	_ = tmpFile.Close()

	// Atomic rename to cache file
	_ = os.Rename(tmpPath, cacheFilePath)

	// Read and serve
	optimizedData, err := os.ReadFile(cacheFilePath)
	if err != nil {
		http.Error(w, "Failed to read optimized image", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", outContentType)
	w.Header().Set("Content-Length", strconv.Itoa(len(optimizedData)))
	w.Header().Set("ETag", `"`+etag+`"`)
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	w.WriteHeader(http.StatusOK)
	if r.Method != http.MethodHead {
		_, _ = w.Write(optimizedData)
	}
}

func hasAlpha(img *image.RGBA) bool {
	bounds := img.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if img.RGBAAt(x, y).A < 255 {
				return true
			}
		}
	}
	return false
}

// fetchSourceImage retrieves the source image securely with strict Path Traversal & SSRF guards.
func (o *Optimizer) fetchSourceImage(src string) (io.ReadCloser, string, error) {
	// Remote image
	if strings.HasPrefix(src, "http://") || strings.HasPrefix(src, "https://") {
		return o.fetchRemoteImage(src)
	}

	// Local image from public/ directory
	return o.fetchLocalImage(src)
}

// fetchLocalImage safely retrieves a file from the public directory (protects against CWE-22 Path Traversal).
func (o *Optimizer) fetchLocalImage(src string) (io.ReadCloser, string, error) {
	cleanRel := filepath.Clean("/" + strings.TrimPrefix(src, "/"))
	if strings.Contains(cleanRel, "..") {
		return nil, "", fmt.Errorf("path traversal attempt detected")
	}

	// Remove leading /public prefix if user passed /public/logo.png
	trimmed := strings.TrimPrefix(cleanRel, "/public")
	if trimmed == "" {
		trimmed = "/"
	}

	// Check embedded public filesystem if available (standalone mode)
	if o.cfg.EmbeddedPublic != nil {
		f, err := o.cfg.EmbeddedPublic.Open(strings.TrimPrefix(trimmed, "/"))
		if err == nil {
			ext := strings.ToLower(filepath.Ext(cleanRel))
			mimeType := "image/jpeg"
			if ext == ".png" {
				mimeType = "image/png"
			} else if ext == ".gif" {
				mimeType = "image/gif"
			}
			return f, mimeType, nil
		}
	}

	// Check local disk public/ directory
	publicDir := filepath.Join(o.cfg.AppDir, "public")
	absPublic, err := filepath.Abs(publicDir)
	if err != nil {
		return nil, "", fmt.Errorf("invalid public directory: %w", err)
	}

	targetPath := filepath.Clean(filepath.Join(absPublic, trimmed))
	// Enforce strict containment
	if !strings.HasPrefix(targetPath, absPublic+string(filepath.Separator)) && targetPath != absPublic {
		return nil, "", fmt.Errorf("access denied: image path escapes public directory")
	}

	file, err := os.Open(targetPath)
	if err != nil {
		return nil, "", fmt.Errorf("image not found: %s", cleanRel)
	}

	ext := strings.ToLower(filepath.Ext(targetPath))
	mimeType := "image/jpeg"
	if ext == ".png" {
		mimeType = "image/png"
	} else if ext == ".gif" {
		mimeType = "image/gif"
	}

	return file, mimeType, nil
}

// fetchRemoteImage securely retrieves a remote image (protects against CWE-918 SSRF).
func (o *Optimizer) fetchRemoteImage(rawURL string) (io.ReadCloser, string, error) {
	if len(o.cfg.AllowedDomains) == 0 {
		return nil, "", fmt.Errorf("remote image optimization is disabled (no allowed domains configured)")
	}

	parsed, err := url.Parse(rawURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return nil, "", fmt.Errorf("invalid image URL")
	}

	hostname := parsed.Hostname()
	domainAllowed := false
	for _, domain := range o.cfg.AllowedDomains {
		if domain == "*" || strings.EqualFold(hostname, domain) || strings.HasSuffix(hostname, "."+domain) {
			domainAllowed = true
			break
		}
	}
	if !domainAllowed {
		return nil, "", fmt.Errorf("domain '%s' is not in allowed image domains", hostname)
	}

	// Resolve IP and verify it is not private/loopback/metadata (SSRF Guard)
	ips, err := net.LookupIP(hostname)
	if err != nil || len(ips) == 0 {
		return nil, "", fmt.Errorf("failed to resolve host: %s", hostname)
	}

	for _, ip := range ips {
		if isPrivateOrLoopback(ip) {
			return nil, "", fmt.Errorf("access to internal/private IP address is forbidden")
		}
	}

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Get(rawURL)
	if err != nil {
		return nil, "", fmt.Errorf("failed to fetch remote image: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, "", fmt.Errorf("remote server returned HTTP %d", resp.StatusCode)
	}

	ct := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(ct, "image/") {
		resp.Body.Close()
		return nil, "", fmt.Errorf("remote response is not an image (Content-Type: %s)", ct)
	}

	return resp.Body, ct, nil
}

func isPrivateOrLoopback(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsInterfaceLocalMulticast() {
		return true
	}
	// Check IPv4 private ranges
	if ip4 := ip.To4(); ip4 != nil {
		switch {
		case ip4[0] == 10:
			return true
		case ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31:
			return true
		case ip4[0] == 192 && ip4[1] == 168:
			return true
		case ip4[0] == 169 && ip4[1] == 254: // AWS/cloud metadata address
			return true
		case ip4[0] == 127: // Loopback
			return true
		case ip4[0] == 0:
			return true
		}
	}
	// Check IPv6 unique local addresses (fc00::/7)
	if len(ip) == 16 && (ip[0]&0xfe) == 0xfc {
		return true
	}
	return false
}

// resizeBilinear implements pure-Go high quality bilinear image scaling.
func resizeBilinear(src image.Image, targetWidth, targetHeight int) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, targetWidth, targetHeight))
	srcBounds := src.Bounds()
	srcW := srcBounds.Dx()
	srcH := srcBounds.Dy()

	if srcW == 0 || srcH == 0 || targetWidth == 0 || targetHeight == 0 {
		return dst
	}

	xRatio := float64(srcW-1) / float64(targetWidth)
	yRatio := float64(srcH-1) / float64(targetHeight)

	for y := 0; y < targetHeight; y++ {
		srcY := float64(y) * yRatio
		yFloor := int(srcY)
		yDiff := srcY - float64(yFloor)
		yCeil := yFloor + 1
		if yCeil >= srcH {
			yCeil = yFloor
		}

		for x := 0; x < targetWidth; x++ {
			srcX := float64(x) * xRatio
			xFloor := int(srcX)
			xDiff := srcX - float64(xFloor)
			xCeil := xFloor + 1
			if xCeil >= srcW {
				xCeil = xFloor
			}

			// Sample 4 neighbor pixels
			c00 := src.At(srcBounds.Min.X+xFloor, srcBounds.Min.Y+yFloor)
			c10 := src.At(srcBounds.Min.X+xCeil, srcBounds.Min.Y+yFloor)
			c01 := src.At(srcBounds.Min.X+xFloor, srcBounds.Min.Y+yCeil)
			c11 := src.At(srcBounds.Min.X+xCeil, srcBounds.Min.Y+yCeil)

			r00, g00, b00, a00 := c00.RGBA()
			r10, g10, b10, a10 := c10.RGBA()
			r01, g01, b01, a01 := c01.RGBA()
			r11, g11, b11, a11 := c11.RGBA()

			blend := func(v00, v10, v01, v11 uint32) uint8 {
				top := float64(v00)*(1.0-xDiff) + float64(v10)*xDiff
				bot := float64(v01)*(1.0-xDiff) + float64(v11)*xDiff
				res := top*(1.0-yDiff) + bot*yDiff
				return uint8(res / 257)
			}

			dst.SetRGBA(x, y, color.RGBA{
				R: blend(r00, r10, r01, r11),
				G: blend(g00, g10, g01, g11),
				B: blend(b00, b10, b01, b11),
				A: blend(a00, a10, a01, a11),
			})
		}
	}
	return dst
}

// Init image decoders
func init() {
	image.RegisterFormat("jpeg", "\xff\xd8", jpeg.Decode, jpeg.DecodeConfig)
	image.RegisterFormat("png", "\x89PNG\r\n\x1a\n", png.Decode, png.DecodeConfig)
	image.RegisterFormat("gif", "GIF8?a", gif.Decode, gif.DecodeConfig)
}
