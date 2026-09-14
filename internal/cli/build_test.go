package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildCmd_Validation(t *testing.T) {
	cmd := BuildCmd()

	// 1. Missing app/ directory
	tempDir := t.TempDir()
	origWd, _ := os.Getwd()
	defer os.Chdir(origWd)

	_ = os.Chdir(tempDir)
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when app/ directory is missing, got nil")
	}
	if !strings.Contains(err.Error(), "folder 'app/' tidak ditemukan") {
		t.Errorf("expected error mentioning missing app/ folder, got: %v", err)
	}

	// 2. Create fake app/ but provide invalid compiler flag
	_ = os.MkdirAll(filepath.Join(tempDir, "app"), 0755)
	cmd = BuildCmd()
	cmd.SetArgs([]string{"--compiler=unsupported_compiler"})
	err = cmd.Execute()
	if err == nil {
		t.Fatal("expected error for unsupported compiler, got nil")
	}
	if !strings.Contains(err.Error(), "invalid --compiler flag") {
		t.Errorf("expected invalid compiler flag error, got: %v", err)
	}
}
