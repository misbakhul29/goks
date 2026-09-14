package cli

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtractTarGzSecurely_TarSlip(t *testing.T) {
	tempDest := t.TempDir()

	// Construct an in-memory tar.gz containing a malicious path traversal entry
	var buf bytes.Buffer
	gzWriter := gzip.NewWriter(&buf)
	tarWriter := tar.NewWriter(gzWriter)

	maliciousEntry := "../../evil.txt"
	header := &tar.Header{
		Name: maliciousEntry,
		Mode: 0644,
		Size: int64(len("malicious content")),
	}

	if err := tarWriter.WriteHeader(header); err != nil {
		t.Fatalf("failed to write header: %v", err)
	}
	if _, err := tarWriter.Write([]byte("malicious content")); err != nil {
		t.Fatalf("failed to write content: %v", err)
	}
	_ = tarWriter.Close()
	_ = gzWriter.Close()

	// Extraction must detect Tar Slip and return an error
	err := extractTarGzSecurely(&buf, tempDest)
	if err == nil {
		t.Fatal("expected Tar Slip error for path traversal, got nil")
	}
	if !strings.Contains(err.Error(), "security violation") {
		t.Errorf("expected security violation error message, got: %v", err)
	}
}

func TestExtractZipSecurely_ZipSlip(t *testing.T) {
	tempDir := t.TempDir()
	zipFilePath := filepath.Join(tempDir, "malicious.zip")

	// Construct an in-memory zip file with path traversal
	f, err := os.Create(zipFilePath)
	if err != nil {
		t.Fatalf("failed to create zip file: %v", err)
	}

	zipWriter := zip.NewWriter(f)
	w, err := zipWriter.Create("../../../evil.txt")
	if err != nil {
		t.Fatalf("failed to create zip entry: %v", err)
	}
	_, _ = w.Write([]byte("evil payload"))
	_ = zipWriter.Close()
	_ = f.Close()

	destDir := filepath.Join(tempDir, "extract")
	_ = os.MkdirAll(destDir, 0755)

	err = extractZipSecurely(zipFilePath, destDir)
	if err == nil {
		t.Fatal("expected Zip Slip error for path traversal, got nil")
	}
	if !strings.Contains(err.Error(), "security violation") {
		t.Errorf("expected security violation error message, got: %v", err)
	}
}

func TestLocateWasmExec_TinyGo(t *testing.T) {
	// If tinygo is installed, locateWasmExec("tinygo") must return an existing file
	path, err := locateWasmExec("tinygo")
	if err == nil {
		if _, statErr := os.Stat(path); statErr != nil {
			t.Errorf("wasm_exec.js path returned by locateWasmExec does not exist: %s", path)
		}
		if !strings.HasSuffix(path, "wasm_exec.js") {
			t.Errorf("expected path to end in wasm_exec.js, got: %s", path)
		}
	}
}
