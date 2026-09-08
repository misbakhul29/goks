package cli

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"

	"github.com/fatih/color"
)

func ensureTailwindCLI() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	goksDir := filepath.Join(home, ".goks", "bin")
	if err := os.MkdirAll(goksDir, 0755); err != nil {
		return "", err
	}

	execName := "tailwindcss"
	if runtime.GOOS == "windows" {
		execName += ".exe"
	}
	execPath := filepath.Join(goksDir, execName)

	if _, err := os.Stat(execPath); err == nil {
		return execPath, nil // already downloaded
	}

	fmt.Print(color.HiBlackString("  Downloading Tailwind CSS Standalone CLI... "))

	var url string
	switch runtime.GOOS {
	case "darwin":
		if runtime.GOARCH == "arm64" {
			url = "https://github.com/tailwindlabs/tailwindcss/releases/latest/download/tailwindcss-macos-arm64"
		} else {
			url = "https://github.com/tailwindlabs/tailwindcss/releases/latest/download/tailwindcss-macos-x64"
		}
	case "linux":
		if runtime.GOARCH == "arm64" {
			url = "https://github.com/tailwindlabs/tailwindcss/releases/latest/download/tailwindcss-linux-arm64"
		} else {
			url = "https://github.com/tailwindlabs/tailwindcss/releases/latest/download/tailwindcss-linux-x64"
		}
	case "windows":
		if runtime.GOARCH == "arm64" {
			url = "https://github.com/tailwindlabs/tailwindcss/releases/latest/download/tailwindcss-windows-arm64.exe"
		} else {
			url = "https://github.com/tailwindlabs/tailwindcss/releases/latest/download/tailwindcss-windows-x64.exe"
		}
	}

	if url == "" {
		return "", fmt.Errorf("unsupported os/arch for tailwind standalone: %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("failed to download tailwind: %d", resp.StatusCode)
	}

	out, err := os.Create(execPath)
	if err != nil {
		return "", err
	}
	defer out.Close()

	if _, err := io.Copy(out, resp.Body); err != nil {
		return "", err
	}
	if err := os.Chmod(execPath, 0755); err != nil {
		return "", err
	}

	fmt.Println(color.GreenString("done"))
	return execPath, nil
}
