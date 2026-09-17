package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAuthScaffold_GeneratesFiles(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "goks-auth-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	if err := runAuthScaffold(tempDir); err != nil {
		t.Fatalf("runAuthScaffold failed: %v", err)
	}

	expectedFiles := []string{
		filepath.Join(tempDir, "models", "user.go"),
		filepath.Join(tempDir, "app", "login", "page.gox"),
		filepath.Join(tempDir, "app", "register", "page.gox"),
		filepath.Join(tempDir, "app", "login", "actions.go"),
	}

	for _, f := range expectedFiles {
		if _, err := os.Stat(f); os.IsNotExist(err) {
			t.Errorf("expected file was not generated: %s", f)
		}
	}

	// Check that a migration file was created in migrations/
	migDir := filepath.Join(tempDir, "migrations")
	entries, err := os.ReadDir(migDir)
	if err != nil {
		t.Fatalf("failed to read migrations dir: %v", err)
	}
	if len(entries) == 0 {
		t.Errorf("expected at least one migration file in %s", migDir)
	}

	// Idempotency check: running scaffold again does not fail or error
	if err := runAuthScaffold(tempDir); err != nil {
		t.Fatalf("second runAuthScaffold failed: %v", err)
	}
}
