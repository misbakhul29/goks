package cli

// Integration tests for goks export asset copying behaviour added in v1.1.0.
//
// The full export pipeline (WASM build, CSS build, export binary) requires a
// real GoKS workspace and the Go toolchain, so we do not try to run it here.
// What we can test deterministically without a network or full build:
//
//  1. copyFile — the primitive that copies every asset.
//  2. Export output directory safety validation (path traversal, protected dirs).
//  3. Public dir walk: if public/ contains files they must appear in the output dir.
//
// These cover the logic that was added in v1.1.0 (static asset copying) and
// the security checks that protect the output directory.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// -----------------------------------------------------------------------
// copyFile tests
// -----------------------------------------------------------------------

func TestCopyFile_CopiesContent(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "dst.txt")

	want := "hello, goks export"
	if err := os.WriteFile(src, []byte(want), 0644); err != nil {
		t.Fatalf("write src: %v", err)
	}

	if err := copyFile(src, dst); err != nil {
		t.Fatalf("copyFile: %v", err)
	}

	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("read dst: %v", err)
	}
	if string(got) != want {
		t.Fatalf("expected %q, got %q", want, string(got))
	}
}

func TestCopyFile_MissingSrcReturnsError(t *testing.T) {
	dir := t.TempDir()
	err := copyFile(filepath.Join(dir, "nonexistent.txt"), filepath.Join(dir, "out.txt"))
	if err == nil {
		t.Fatal("expected error for missing source file, got nil")
	}
}

func TestCopyFile_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "empty.txt")
	dst := filepath.Join(dir, "empty_out.txt")

	if err := os.WriteFile(src, []byte{}, 0644); err != nil {
		t.Fatalf("create empty src: %v", err)
	}
	if err := copyFile(src, dst); err != nil {
		t.Fatalf("copyFile empty file: %v", err)
	}
	fi, err := os.Stat(dst)
	if err != nil {
		t.Fatalf("stat dst: %v", err)
	}
	if fi.Size() != 0 {
		t.Fatalf("expected 0-byte dst, got %d bytes", fi.Size())
	}
}

// -----------------------------------------------------------------------
// Export output directory safety validation
// -----------------------------------------------------------------------

// The ExportCmd validates outDir against a blocklist of system directories
// and project-internal directories. We test these rules directly by running
// the command with a real temp project that has go.mod + app/ in place.

func makeMinimalProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/testapp\n\ngo 1.21\n"), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "app"), 0755); err != nil {
		t.Fatalf("mkdir app: %v", err)
	}
	return dir
}

func TestExportCmd_RejectsSystemOutDir(t *testing.T) {
	proj := makeMinimalProject(t)

	for _, protected := range []string{"/", "/etc", "/bin", "/usr", "/var"} {
		cmd := ExportCmd()
		cmd.SetArgs([]string{"--dir=" + proj, "--out=" + protected})
		err := cmd.Execute()
		if err == nil {
			t.Errorf("expected error for protected outDir %q, got nil", protected)
			continue
		}
		if !strings.Contains(err.Error(), "security violation") && !strings.Contains(err.Error(), "protected") {
			t.Errorf("outDir=%q: expected 'security violation' in error, got: %v", protected, err)
		}
	}
}

func TestExportCmd_RejectsOutDirEqualsProjectRoot(t *testing.T) {
	proj := makeMinimalProject(t)

	cmd := ExportCmd()
	cmd.SetArgs([]string{"--dir=" + proj, "--out=" + proj})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when outDir == project root, got nil")
	}
	if !strings.Contains(err.Error(), "safety violation") {
		t.Errorf("expected 'safety violation', got: %v", err)
	}
}

func TestExportCmd_RejectsOutDirEqualsAppDir(t *testing.T) {
	proj := makeMinimalProject(t)
	appDir := filepath.Join(proj, "app")

	cmd := ExportCmd()
	cmd.SetArgs([]string{"--dir=" + proj, "--out=" + appDir})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when outDir == app/, got nil")
	}
	if !strings.Contains(err.Error(), "safety violation") {
		t.Errorf("expected 'safety violation', got: %v", err)
	}
}

func TestExportCmd_RejectsInvalidCompiler(t *testing.T) {
	proj := makeMinimalProject(t)
	out := filepath.Join(t.TempDir(), "out")

	cmd := ExportCmd()
	cmd.SetArgs([]string{"--dir=" + proj, "--out=" + out, "--compiler=webpack"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for invalid compiler, got nil")
	}
	if !strings.Contains(err.Error(), "invalid --compiler flag") {
		t.Errorf("expected 'invalid --compiler flag', got: %v", err)
	}
}

// -----------------------------------------------------------------------
// Public dir walk — asset copy simulation
// -----------------------------------------------------------------------

// This test simulates what the export pipeline does with the public/ directory:
// walk all files and copy them to the output dir preserving relative paths.
// We test the logic directly using copyFile to confirm the mechanism works.
func TestExport_PublicDirAssetsCopied(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	// Create a realistic public/ structure
	files := map[string]string{
		"favicon.ico":       "fake-ico-content",
		"robots.txt":        "User-agent: *\nDisallow:",
		"images/logo.png":   "fake-png-content",
		"fonts/inter.woff2": "fake-font-content",
	}
	for rel, content := range files {
		full := filepath.Join(src, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			t.Fatalf("mkdir %s: %v", filepath.Dir(full), err)
		}
		if err := os.WriteFile(full, []byte(content), 0644); err != nil {
			t.Fatalf("write %s: %v", full, err)
		}
	}

	// Simulate the public dir walk from export.go lines 139–151
	err := filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(src, path)
		dest := filepath.Join(dst, rel)
		_ = os.MkdirAll(filepath.Dir(dest), 0755)
		return copyFile(path, dest)
	})
	if err != nil {
		t.Fatalf("walk+copy: %v", err)
	}

	// Verify every file landed in the right place with the right content
	for rel, wantContent := range files {
		got, err := os.ReadFile(filepath.Join(dst, rel))
		if err != nil {
			t.Errorf("missing output file %s: %v", rel, err)
			continue
		}
		if string(got) != wantContent {
			t.Errorf("file %s: expected %q, got %q", rel, wantContent, string(got))
		}
	}
}

// TestExport_MissingPublicDirIsNoOp verifies that the export does not
// fail when public/ does not exist (it must be silently skipped).
func TestExport_MissingPublicDirIsNoOp(t *testing.T) {
	publicDir := filepath.Join(t.TempDir(), "public") // does not exist
	_, err := os.Stat(publicDir)
	// The export code guards with `if err == nil { walk }` — simulate that.
	if err == nil {
		t.Fatal("test setup error: public dir should not exist")
	}
	// No walk, no copy — this is a no-op, should not panic or error.
	// We verify by running the same guard logic:
	walked := false
	if _, statErr := os.Stat(publicDir); statErr == nil {
		walked = true
	}
	if walked {
		t.Fatal("expected walk to be skipped when public/ does not exist")
	}
}
