package server

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/misbakhul29/goks/pkg/component"
	"github.com/misbakhul29/goks/pkg/html"
)

type exportTestRouter struct {
	component.ComponentBase
}

func (r *exportTestRouter) PageRoutes() []string {
	return []string{"/", "/about"}
}

func (r *exportTestRouter) HasMatchedPage(path string) bool {
	return path == "/" || path == "/about"
}

func (r *exportTestRouter) Render() *component.Node {
	return html.Html(
		html.Body(
			html.H1(component.Text("Export Test Page")),
		),
	)
}

func TestExportStatic_MultiRoutes(t *testing.T) {
	tempAppDir := t.TempDir()
	tempExportDir := filepath.Join(tempAppDir, "out")

	// Create fake public file
	publicDir := filepath.Join(tempAppDir, "public")
	_ = os.MkdirAll(publicDir, 0755)
	_ = os.WriteFile(filepath.Join(publicDir, "robots.txt"), []byte("User-agent: *\nDisallow:"), 0644)

	// Create fake build assets
	buildDir := filepath.Join(tempAppDir, ".goks", "build")
	_ = os.MkdirAll(buildDir, 0755)
	_ = os.WriteFile(filepath.Join(buildDir, "app.css"), []byte("body{margin:0;}"), 0644)

	srv := NewDev(Config{
		AppDir: tempAppDir,
		Root:   &exportTestRouter{},
	})

	err := srv.ExportStatic(tempExportDir)
	if err != nil {
		t.Fatalf("ExportStatic failed: %v", err)
	}

	// 1. Verify index.html exists
	indexPath := filepath.Join(tempExportDir, "index.html")
	if b, err := os.ReadFile(indexPath); err != nil {
		t.Fatalf("expected index.html to exist: %v", err)
	} else if !strings.Contains(string(b), "Export Test Page") {
		t.Errorf("expected index.html to contain rendered page, got: %s", string(b))
	}

	// 2. Verify /about route (about/index.html and about.html)
	aboutIndexPath := filepath.Join(tempExportDir, "about", "index.html")
	if _, err := os.Stat(aboutIndexPath); err != nil {
		t.Errorf("expected about/index.html to exist: %v", err)
	}

	aboutHTMLPath := filepath.Join(tempExportDir, "about.html")
	if _, err := os.Stat(aboutHTMLPath); err != nil {
		t.Errorf("expected about.html clean-URL to exist: %v", err)
	}

	// 3. Verify 404.html
	notFoundPath := filepath.Join(tempExportDir, "404.html")
	if _, err := os.Stat(notFoundPath); err != nil {
		t.Errorf("expected 404.html to exist: %v", err)
	}

	// 4. Verify assets copied
	cssPath := filepath.Join(tempExportDir, "app.css")
	if _, err := os.Stat(cssPath); err != nil {
		t.Errorf("expected app.css asset in export: %v", err)
	}

	// 5. Verify public/ files copied
	robotsPath := filepath.Join(tempExportDir, "robots.txt")
	if _, err := os.Stat(robotsPath); err != nil {
		t.Errorf("expected robots.txt from public dir in export: %v", err)
	}
}
