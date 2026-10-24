package cli

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/fatih/color"
)

const tailwindVersion = "v4.3.3"

func ensureTailwindCLI() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	goksDir := filepath.Join(home, ".goks", "bin")
	if err := os.MkdirAll(goksDir, 0755); err != nil {
		return "", err
	}

	execName := fmt.Sprintf("tailwindcss-%s", tailwindVersion)
	if runtime.GOOS == "windows" {
		execName += ".exe"
	}
	execPath := filepath.Join(goksDir, execName)

	if _, err := os.Stat(execPath); err == nil {
		return execPath, nil // already downloaded
	}

	fmt.Print(color.HiBlackString("  Downloading Tailwind CSS Standalone CLI (" + tailwindVersion + ")... "))

	baseURL := fmt.Sprintf("https://github.com/tailwindlabs/tailwindcss/releases/download/%s/tailwindcss-", tailwindVersion)
	var url string
	switch runtime.GOOS {
	case "darwin":
		if runtime.GOARCH == "arm64" {
			url = baseURL + "macos-arm64"
		} else {
			url = baseURL + "macos-x64"
		}
	case "linux":
		if runtime.GOARCH == "arm64" {
			url = baseURL + "linux-arm64"
		} else {
			url = baseURL + "linux-x64"
		}
	case "windows":
		url = baseURL + "windows-x64.exe"
	}

	if url == "" {
		return "", fmt.Errorf("unsupported os/arch for tailwind standalone: %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Get(url)
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
