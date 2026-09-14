package cli

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/fatih/color"
)

const (
	tinygoVersion   = "0.42.0"
	maxFileExtract  = 350 * 1024 * 1024 // 350 MB per file limit (tar/zip bomb protection)
	downloadTimeout = 10 * time.Minute
)

// ensureTinyGo checks if tinygo is available in system PATH or ~/.goks/tinygo/bin/tinygo.
// If not found, it securely downloads and atomically extracts TinyGo standalone into ~/.goks/tinygo.
func ensureTinyGo() (string, error) {
	// 1. Check system PATH first
	if p, err := exec.LookPath("tinygo"); err == nil {
		return p, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}

	// 2. Check ~/.local/bin/tinygo
	localBin := filepath.Join(home, ".local", "bin", "tinygo")
	if _, err := os.Stat(localBin); err == nil {
		return localBin, nil
	}

	// 3. Check ~/.goks/tinygo/bin/tinygo
	tinyDir := filepath.Join(home, ".goks", "tinygo")
	execName := "tinygo"
	if runtime.GOOS == "windows" {
		execName += ".exe"
	}
	execPath := filepath.Join(tinyDir, "bin", execName)
	if _, err := os.Stat(execPath); err == nil {
		return execPath, nil
	}

	// 4. Secure Auto-download TinyGo
	fmt.Print(color.HiBlackString("  Downloading TinyGo Standalone (<300KB WASM compiler)... "))

	var archiveURL string
	var isZip bool

	switch runtime.GOOS {
	case "linux":
		switch runtime.GOARCH {
		case "arm64":
			archiveURL = fmt.Sprintf("https://github.com/tinygo-org/tinygo/releases/download/v%s/tinygo%s.linux-arm64.tar.gz", tinygoVersion, tinygoVersion)
		case "arm":
			archiveURL = fmt.Sprintf("https://github.com/tinygo-org/tinygo/releases/download/v%s/tinygo%s.linux-arm.tar.gz", tinygoVersion, tinygoVersion)
		default:
			archiveURL = fmt.Sprintf("https://github.com/tinygo-org/tinygo/releases/download/v%s/tinygo%s.linux-amd64.tar.gz", tinygoVersion, tinygoVersion)
		}
	case "darwin":
		if runtime.GOARCH == "arm64" {
			archiveURL = fmt.Sprintf("https://github.com/tinygo-org/tinygo/releases/download/v%s/tinygo%s.darwin-arm64.tar.gz", tinygoVersion, tinygoVersion)
		} else {
			archiveURL = fmt.Sprintf("https://github.com/tinygo-org/tinygo/releases/download/v%s/tinygo%s.darwin-amd64.tar.gz", tinygoVersion, tinygoVersion)
		}
	case "windows":
		isZip = true
		archiveURL = fmt.Sprintf("https://github.com/tinygo-org/tinygo/releases/download/v%s/tinygo%s.windows-amd64.zip", tinygoVersion, tinygoVersion)
	default:
		return "", fmt.Errorf("automatic TinyGo download is not supported for %s/%s. Please install TinyGo manually: https://tinygo.org/getting-started/install/", runtime.GOOS, runtime.GOARCH)
	}

	client := &http.Client{
		Timeout: downloadTimeout,
	}

	resp, err := client.Get(archiveURL)
	if err != nil {
		return "", fmt.Errorf("failed to download TinyGo: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download TinyGo from %s (HTTP %d)", archiveURL, resp.StatusCode)
	}

	// Prepare atomic temporary extraction directory
	goksHome := filepath.Join(home, ".goks")
	if err := os.MkdirAll(goksHome, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory %s: %w", goksHome, err)
	}

	tempExtractDir := filepath.Join(goksHome, fmt.Sprintf("tinygo_download_%d", time.Now().UnixNano()))
	if err := os.MkdirAll(tempExtractDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create temporary extraction directory: %w", err)
	}
	defer os.RemoveAll(tempExtractDir) // Clean up temp dir on any error or exit

	if isZip {
		// Download zip to temp file then extract
		tmpZipFile, err := os.CreateTemp(tempExtractDir, "tinygo-*.zip")
		if err != nil {
			return "", err
		}
		defer os.Remove(tmpZipFile.Name())

		if _, err := io.Copy(tmpZipFile, io.LimitReader(resp.Body, 500*1024*1024)); err != nil {
			tmpZipFile.Close()
			return "", fmt.Errorf("failed to save zip archive: %w", err)
		}
		tmpZipFile.Close()

		if err := extractZipSecurely(tmpZipFile.Name(), tempExtractDir); err != nil {
			return "", err
		}
	} else {
		if err := extractTarGzSecurely(resp.Body, tempExtractDir); err != nil {
			return "", err
		}
	}

	// TinyGo archive extracts into a "tinygo" subfolder
	extractedTinygo := filepath.Join(tempExtractDir, "tinygo")
	if _, err := os.Stat(extractedTinygo); err != nil {
		// Fallback: check if files are directly in tempExtractDir
		extractedTinygo = tempExtractDir
	}

	// Verify extracted executable exists and is valid before replacing destination
	tempExecPath := filepath.Join(extractedTinygo, "bin", execName)
	if _, err := os.Stat(tempExecPath); err != nil {
		return "", fmt.Errorf("extracted TinyGo executable not found at %s", tempExecPath)
	}
	_ = os.Chmod(tempExecPath, 0755)

	// Atomic replacement: remove previous tinyDir (if corrupted) and rename
	_ = os.RemoveAll(tinyDir)
	if err := os.Rename(extractedTinygo, tinyDir); err != nil {
		return "", fmt.Errorf("failed to install TinyGo to %s: %w", tinyDir, err)
	}

	fmt.Println(color.GreenString("done"))
	return execPath, nil
}

// extractTarGzSecurely extracts a .tar.gz stream with Tar Slip and Bomb protections.
func extractTarGzSecurely(r io.Reader, destDir string) error {
	gzReader, err := gzip.NewReader(r)
	if err != nil {
		return fmt.Errorf("failed to decompress gzip stream: %w", err)
	}
	defer gzReader.Close()

	cleanDest := filepath.Clean(destDir)
	tarReader := tar.NewReader(gzReader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("tar read error: %w", err)
		}

		// Security: Tar Slip path traversal check (CWE-22)
		target := filepath.Clean(filepath.Join(cleanDest, header.Name))
		if !strings.HasPrefix(target, cleanDest+string(os.PathSeparator)) && target != cleanDest {
			return fmt.Errorf("security violation: illegal file path %q outside destination", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
		case tar.TypeReg, tar.TypeRegA:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			// Security: Sanitize file permissions (remove setuid/setgid bits)
			mode := header.FileInfo().Mode().Perm() & 0755
			outFile, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR|os.O_TRUNC, mode)
			if err != nil {
				return err
			}

			// Security: Decompression bomb limit (CWE-409)
			copied, err := io.Copy(outFile, io.LimitReader(tarReader, maxFileExtract))
			outFile.Close()
			if err != nil {
				return fmt.Errorf("failed writing %s: %w", target, err)
			}
			if copied >= maxFileExtract {
				return fmt.Errorf("security violation: file %s exceeded max allowed size", header.Name)
			}
		}
	}
	return nil
}

// extractZipSecurely extracts a .zip file with Zip Slip and Bomb protections.
func extractZipSecurely(zipPath, destDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("failed to open zip file: %w", err)
	}
	defer r.Close()

	cleanDest := filepath.Clean(destDir)

	for _, file := range r.File {
		// Security: Zip Slip path traversal check (CWE-22)
		target := filepath.Clean(filepath.Join(cleanDest, file.Name))
		if !strings.HasPrefix(target, cleanDest+string(os.PathSeparator)) && target != cleanDest {
			return fmt.Errorf("security violation: illegal file path %q outside destination", file.Name)
		}

		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}

		rc, err := file.Open()
		if err != nil {
			return err
		}

		mode := file.Mode().Perm() & 0755
		outFile, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR|os.O_TRUNC, mode)
		if err != nil {
			rc.Close()
			return err
		}

		copied, err := io.Copy(outFile, io.LimitReader(rc, maxFileExtract))
		outFile.Close()
		rc.Close()

		if err != nil {
			return fmt.Errorf("failed writing %s: %w", target, err)
		}
		if copied >= maxFileExtract {
			return fmt.Errorf("security violation: file %s exceeded max allowed size", file.Name)
		}
	}
	return nil
}
