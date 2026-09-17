package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/misbakhul29/goks/internal/compiler"
)

func TestScaffoldApp_Validation(t *testing.T) {
	tests := []struct {
		name    string
		appName string
		wantErr string
	}{
		{
			name:    "Empty name",
			appName: "   ",
			wantErr: "app name cannot be empty",
		},
		{
			name:    "Path traversal",
			appName: "../evil",
			wantErr: "must not contain path separators",
		},
		{
			name:    "Slash in name",
			appName: "foo/bar",
			wantErr: "must not contain path separators",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := scaffoldApp(tt.appName, "")
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("expected error containing %q, got: %v", tt.wantErr, err)
			}
		})
	}
}

func TestScaffoldApp_E2E(t *testing.T) {
	tmpDir := t.TempDir()
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working dir: %v", err)
	}
	defer os.Chdir(origWd)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir to tmpDir: %v", err)
	}

	appName := "my-test-app"
	modPath := "example.com/test/my-test-app"

	// 1. First run: should succeed
	if err := scaffoldApp(appName, modPath); err != nil {
		t.Fatalf("scaffoldApp failed: %v", err)
	}

	// 2. Overwrite check: running again in same directory must fail
	if err := scaffoldApp(appName, modPath); err == nil {
		t.Fatal("expected error when directory already exists, got nil")
	}

	// 3. Verify key files exist and contain expected module / version
	expectedFiles := []string{
		"go.mod",
		"app/layout.gox",
		"app/page.gox",
		"app/components/hero.gox",
		"components/button.gox",
		"database/models/user.go",
		"services/user_service.go",
		"database/repositories/user_repo.go",
		"api/routes.go",
		"config/goks.config.go",
		"middleware/logger.go",
		".gitignore",
		"README.md",
		"public/global.css",
	}

	appDir := filepath.Join(tmpDir, appName)
	for _, rel := range expectedFiles {
		p := filepath.Join(appDir, rel)
		if _, err := os.Stat(p); os.IsNotExist(err) {
			t.Errorf("expected generated file %s does not exist", rel)
		}
	}

	// Verify go.mod content
	goModBytes, err := os.ReadFile(filepath.Join(appDir, "go.mod"))
	if err != nil {
		t.Fatalf("failed to read go.mod: %v", err)
	}
	if !strings.Contains(string(goModBytes), "module "+modPath) {
		t.Errorf("go.mod missing expected module path: %s", string(goModBytes))
	}

	// 4. Verify that generated .gox files successfully transpile
	goxFiles := []string{
		"app/layout.gox",
		"app/page.gox",
		"app/components/hero.gox",
		"components/button.gox",
	}

	for _, rel := range goxFiles {
		content, err := os.ReadFile(filepath.Join(appDir, rel))
		if err != nil {
			t.Fatalf("failed to read %s: %v", rel, err)
		}
		transpiled, err := compiler.Transpile(string(content))
		if err != nil {
			t.Fatalf("failed to transpile %s: %v", rel, err)
		}
		if len(transpiled) == 0 {
			t.Errorf("transpiled output for %s was empty", rel)
		}
	}
}
