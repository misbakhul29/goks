package image_test

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	goksimage "github.com/misbakhul29/goks/pkg/image"
)

// createDummyPNG creates a valid PNG image of size w x h.
func createDummyPNG(w, h int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, color.RGBA{R: 255, G: 100, B: 50, A: 255})
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

func TestOptimizer_ServeLocalImage(t *testing.T) {
	tempDir := t.TempDir()
	publicDir := filepath.Join(tempDir, "public")
	_ = os.MkdirAll(publicDir, 0755)

	// Write a 200x200 dummy PNG to public/sample.png
	samplePNG := createDummyPNG(200, 200)
	if err := os.WriteFile(filepath.Join(publicDir, "sample.png"), samplePNG, 0644); err != nil {
		t.Fatalf("failed to write sample image: %v", err)
	}

	opt := goksimage.NewOptimizer(goksimage.Config{
		AppDir: tempDir,
	})

	// 1. Request image scaled to width 96 (snapped to 96)
	req := httptest.NewRequest(http.MethodGet, "/__goks_image?url=/sample.png&w=96&q=80", nil)
	w := httptest.NewRecorder()
	opt.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200, got %d: %s", w.Code, w.Body.String())
	}

	etag := w.Header().Get("ETag")
	if etag == "" {
		t.Fatal("expected non-empty ETag header")
	}

	cc := w.Header().Get("Cache-Control")
	if !strings.Contains(cc, "max-age=") {
		t.Fatalf("expected Cache-Control header, got %q", cc)
	}

	// Verify decoded image dimensions
	decoded, _, err := image.Decode(bytes.NewReader(w.Body.Bytes()))
	if err != nil {
		t.Fatalf("failed to decode response image: %v", err)
	}
	if decoded.Bounds().Dx() != 96 {
		t.Fatalf("expected width 96, got %d", decoded.Bounds().Dx())
	}

	// 2. Test If-None-Match (304 Not Modified)
	req304 := httptest.NewRequest(http.MethodGet, "/__goks_image?url=/sample.png&w=96&q=80", nil)
	req304.Header.Set("If-None-Match", etag)
	w304 := httptest.NewRecorder()
	opt.ServeHTTP(w304, req304)

	if w304.Code != http.StatusNotModified {
		t.Fatalf("expected 304 Not Modified, got %d", w304.Code)
	}
}

func TestOptimizer_PathTraversal_Rejected(t *testing.T) {
	tempDir := t.TempDir()
	publicDir := filepath.Join(tempDir, "public")
	_ = os.MkdirAll(publicDir, 0755)

	opt := goksimage.NewOptimizer(goksimage.Config{
		AppDir: tempDir,
	})

	traversalURLs := []string{
		"/../../etc/passwd",
		"../secret.key",
		"public/../../../boot",
	}

	for _, u := range traversalURLs {
		req := httptest.NewRequest(http.MethodGet, "/__goks_image?url="+u+"&w=128", nil)
		w := httptest.NewRecorder()
		opt.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400 Bad Request for traversal %q, got %d", u, w.Code)
		}
	}
}

func TestOptimizer_SnapWidth(t *testing.T) {
	opt := goksimage.NewOptimizer(goksimage.Config{})

	tests := []struct {
		input    int
		expected int
	}{
		{10, 16},
		{50, 48},
		{120, 128},
		{700, 750},
		{3900, 3840},
	}

	for _, tc := range tests {
		actual := opt.SnapWidth(tc.input)
		if actual != tc.expected {
			t.Errorf("SnapWidth(%d): expected %d, got %d", tc.input, tc.expected, actual)
		}
	}
}

func TestOptimizer_DecompressionBomb_Protection(t *testing.T) {
	tempDir := t.TempDir()
	publicDir := filepath.Join(tempDir, "public")
	_ = os.MkdirAll(publicDir, 0755)

	// Create oversized image (exceeding MaxWidth = 100)
	hugePNG := createDummyPNG(200, 200)
	_ = os.WriteFile(filepath.Join(publicDir, "huge.png"), hugePNG, 0644)

	opt := goksimage.NewOptimizer(goksimage.Config{
		AppDir:    tempDir,
		MaxWidth:  100, // strictly reject > 100
		MaxHeight: 100,
	})

	req := httptest.NewRequest(http.MethodGet, "/__goks_image?url=/huge.png&w=64", nil)
	w := httptest.NewRecorder()
	opt.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request for decompression bomb, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "exceed maximum allowed") {
		t.Fatalf("unexpected error message: %s", w.Body.String())
	}
}

func TestOptimizer_SSRF_Protection(t *testing.T) {
	opt := goksimage.NewOptimizer(goksimage.Config{
		AllowedDomains: []string{"cdn.example.com"},
	})

	ssrfTargets := []string{
		"http://127.0.0.1:8080/secret.png",
		"http://169.254.169.254/latest/meta-data",
		"http://192.168.1.1/router.png",
		"http://evil.com/exploit.png",
	}

	for _, target := range ssrfTargets {
		req := httptest.NewRequest(http.MethodGet, "/__goks_image?url="+target+"&w=128", nil)
		w := httptest.NewRecorder()
		opt.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400 Bad Request for SSRF target %q, got %d", target, w.Code)
		}
	}
}

func TestImage_Component_Render(t *testing.T) {
	imgComp := goksimage.New(goksimage.Props{
		Src:      "/photos/nature.jpg",
		Alt:      "Beautiful Nature",
		Width:    800,
		Height:   600,
		Quality:  80,
		Priority: false,
		Class:    "rounded-xl shadow-lg",
	})

	node := imgComp.Render()
	if node.Tag != "img" {
		t.Fatalf("expected img tag, got %s", node.Tag)
	}

	// Verify attributes
	if node.Props["alt"] != "Beautiful Nature" {
		t.Errorf("expected alt attribute, got %v", node.Props["alt"])
	}
	if node.Props["loading"] != "lazy" {
		t.Errorf("expected loading=lazy, got %v", node.Props["loading"])
	}
	if node.Props["decoding"] != "async" {
		t.Errorf("expected decoding=async, got %v", node.Props["decoding"])
	}

	src, ok := node.Props["src"].(string)
	if !ok || !strings.Contains(src, "/__goks_image?url=") {
		t.Errorf("expected src to call /__goks_image, got %v", node.Props["src"])
	}

	srcset, ok := node.Props["srcset"].(string)
	if !ok || !strings.Contains(srcset, "640w") || !strings.Contains(srcset, "1080w") {
		t.Errorf("expected responsive srcset, got %v", node.Props["srcset"])
	}

	style, ok := node.Props["style"].(string)
	if !ok || !strings.Contains(style, "aspect-ratio: 800 / 600") {
		t.Errorf("expected aspect-ratio style to prevent layout shift, got %v", node.Props["style"])
	}
}
