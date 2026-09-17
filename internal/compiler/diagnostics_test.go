package compiler_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/misbakhul29/goks/internal/compiler"
)

func TestDiagnostics_UnclosedReturnBlock(t *testing.T) {
	src := `package main

func render() {
	// Line 4
	return (
		<div>hello
`
	_, err := compiler.TranspileWithSource(src, "app/page.gox")
	if err == nil {
		t.Fatal("expected compilation error for unclosed return block")
	}

	ce, ok := err.(*compiler.CompileError)
	if !ok {
		t.Fatalf("expected *compiler.CompileError, got %T: %v", err, err)
	}

	if ce.File != "app/page.gox" {
		t.Errorf("expected file 'app/page.gox', got %q", ce.File)
	}
	if ce.Line != 5 {
		t.Errorf("expected error at line 5, got line %d", ce.Line)
	}
	if !strings.Contains(ce.Error(), "app/page.gox:5:") {
		t.Errorf("expected formatted error to contain 'app/page.gox:5:', got: %s", ce.Error())
	}
}

func TestDiagnostics_InvalidXMLSyntaxLineMapping(t *testing.T) {
	src := `package main

func render() {
	return (
		<div>
			<span className="bold">Text</p>
		</div>
	)
}
`
	_, err := compiler.TranspileWithSource(src, "components/button.gox")
	if err == nil {
		t.Fatal("expected XML syntax error for mismatched closing tag")
	}

	ce, ok := err.(*compiler.CompileError)
	if !ok {
		t.Fatalf("expected *compiler.CompileError, got %T: %v", err, err)
	}

	if ce.File != "components/button.gox" {
		t.Errorf("expected file 'components/button.gox', got %q", ce.File)
	}
	// Line inside return ( ... ) where mismatch happens is around line 6
	if ce.Line < 5 || ce.Line > 8 {
		t.Errorf("expected error line between 5 and 8, got line %d", ce.Line)
	}
}

func TestDiagnostics_PrepareWorkspaceReportsFilePath(t *testing.T) {
	tmpDir := t.TempDir()
	appDir := filepath.Join(tmpDir, "app")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatal(err)
	}

	badGox := filepath.Join(appDir, "broken.gox")
	if err := os.WriteFile(badGox, []byte("package main\nreturn (\n"), 0644); err != nil {
		t.Fatal(err)
	}

	err := compiler.PrepareWorkspace(tmpDir)
	if err == nil {
		t.Fatal("expected PrepareWorkspace to fail on invalid .gox file")
	}

	if !strings.Contains(err.Error(), "app/broken.gox") && !strings.Contains(err.Error(), "broken.gox") {
		t.Fatalf("expected error message to cite broken.gox file path, got: %v", err)
	}
}
