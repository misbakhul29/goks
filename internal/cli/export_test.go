package cli

import (
	"strings"
	"testing"
)

func TestExportCmd_Validation(t *testing.T) {
	cmd := ExportCmd()

	// Test non-existent project directory
	cmd.SetArgs([]string{"--dir=/nonexistent/path/for/goks/test"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for non-existent project directory, got nil")
	}
	if !strings.Contains(err.Error(), "no go.mod found") {
		t.Errorf("expected 'no go.mod found' error, got: %v", err)
	}

	// Test invalid compiler flag
	tempDir := t.TempDir()
	cmd.SetArgs([]string{"--dir=" + tempDir, "--compiler=invalid_compiler"})
	err = cmd.Execute()
	if err == nil {
		t.Fatal("expected error for invalid compiler, got nil")
	}
}
