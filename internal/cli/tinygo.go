package cli

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/fatih/color"
)

// ensureTinyGo checks if tinygo is available in PATH or ~/.goks/tinygo/bin/tinygo.
// If not found, it downloads and extracts TinyGo automatically into ~/.goks/tinygo.
func ensureTinyGo() (string, error) {
	// 1. Check system PATH
	if p, err := exec.LookPath("tinygo"); err == nil {
		return p, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	// 2. Check ~/.local/bin/tinygo
	localBin := filepath.Join(home, ".local", "bin", "tinygo")
	if _, err := os.Stat(localBin); err == nil {
		return localBin, nil
	}

	// 3. Check ~/.goks/tinygo
	tinyDir := filepath.Join(home, ".goks", "tinygo")
	execName := "tinygo"
	if runtime.GOOS == "windows" {
		execName += ".exe"
	}
	execPath := filepath.Join(tinyDir, "bin", execName)
	if _, err := os.Stat(execPath); err == nil {
		return execPath, nil
	}

	// 4. Auto-download TinyGo
	fmt.Print(color.HiBlackString("  Downloading TinyGo Standalone (<300KB WASM compiler)... "))

	const version = "0.42.0"
	var archiveURL string

	switch runtime.GOOS {
	case "linux":
		if runtime.GOARCH == "arm64" {
			archiveURL = fmt.Sprintf("https://github.com/tinygo-org/tinygo/releases/download/v%s/tinygo%s.linux-arm64.tar.gz", version, version)
		} else {
			archiveURL = fmt.Sprintf("https://github.com/tinygo-org/tinygo/releases/download/v%s/tinygo%s.linux-amd64.tar.gz", version, version)
		}
	case "darwin":
		if runtime.GOARCH == "arm64" {
			archiveURL = fmt.Sprintf("https://github.com/tinygo-org/tinygo/releases/download/v%s/tinygo%s.darwin-arm64.tar.gz", version, version)
		} else {
			archiveURL = fmt.Sprintf("https://github.com/tinygo-org/tinygo/releases/download/v%s/tinygo%s.darwin-amd64.tar.gz", version, version)
		}
	default:
		return "", fmt.Errorf("automatic TinyGo download is not yet supported for %s/%s. Please install TinyGo manually: https://tinygo.org/getting-started/install/", runtime.GOOS, runtime.GOARCH)
	}

	resp, err := http.Get(archiveURL)
	if err != nil {
		return "", fmt.Errorf("failed to download TinyGo: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("failed to download TinyGo from %s (HTTP %d)", archiveURL, resp.StatusCode)
	}

	gzReader, err := gzip.NewReader(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to decompress gzip: %w", err)
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)
	destParent := filepath.Join(home, ".goks")
	_ = os.MkdirAll(destParent, 0755)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("tar read error: %w", err)
		}

		target := filepath.Join(destParent, header.Name)
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return "", err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return "", err
			}
			outFile, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return "", err
			}
			if _, err := io.Copy(outFile, tarReader); err != nil {
				outFile.Close()
				return "", err
			}
			outFile.Close()
		}
	}

	if _, err := os.Stat(execPath); err == nil {
		_ = os.Chmod(execPath, 0755)
		fmt.Println(color.GreenString("done"))
		return execPath, nil
	}

	return "", fmt.Errorf("tinygo binary not found at %s after extraction", execPath)
}
