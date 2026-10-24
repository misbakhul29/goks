package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGeneratePage_PathTraversal(t *testing.T) {
	tempDir := t.TempDir()
	appDir := filepath.Join(tempDir, "app")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatalf("failed to create app dir: %v", err)
	}

	// Change current working directory to tempDir during test
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current wd: %v", err)
	}
	defer func() {
		_ = os.Chdir(oldWd)
	}()

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to chdir to tempDir: %v", err)
	}

	traversalRoutes := []string{
		"../../escaped",
		"../outside",
		"sub/../../escaping",
	}

	for _, route := range traversalRoutes {
		err := generatePage(route)
		if err == nil {
			t.Errorf("expected error for traversal route %q, but got nil", route)
		} else if !strings.Contains(err.Error(), "escapes app/") {
			t.Errorf("expected 'escapes app/' error for %q, got: %v", route, err)
		}
	}
}
