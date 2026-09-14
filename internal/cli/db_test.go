package cli_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/misbakhul29/goks/internal/cli"
)

func TestDBCmd_MakeMigration_Validation(t *testing.T) {
	tempDir := t.TempDir()

	// 1. Valid migration name
	cmd := cli.DBCmd()
	cmd.SetArgs([]string{"make:migration", "create_products_table", "--dir", tempDir})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected successful migration creation, got: %v", err)
	}

	migrationsDir := filepath.Join(tempDir, "migrations")
	files, err := os.ReadDir(migrationsDir)
	if err != nil || len(files) != 1 {
		t.Fatalf("expected 1 migration file, got: %v", files)
	}

	// 2. Security validation: Path traversal or invalid characters in migration name
	badNames := []string{
		"../malicious",
		"hack/payload",
		"inject;rm -rf",
		"spaces not allowed",
	}

	for _, bad := range badNames {
		badCmd := cli.DBCmd()
		badCmd.SetArgs([]string{"make:migration", bad, "--dir", tempDir})
		if err := badCmd.Execute(); err == nil {
			t.Errorf("security violation: expected rejection of invalid migration name %q, but succeeded", bad)
		}
	}
}
