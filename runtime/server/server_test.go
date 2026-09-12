package server

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/misbakhul29/goks/pkg/component"
	"github.com/misbakhul29/goks/pkg/html"
	"github.com/misbakhul29/goks/pkg/metadata"
)


type testLayout struct {
	component.ComponentBase
	Children component.Renderable
}

func (l *testLayout) Metadata() metadata.Metadata {
	return metadata.Metadata{
		Title:         "GoKS",
		TitleTemplate: "%s | GoKS App",
		Description:   "Test GoKS App Description",
		OpenGraph: &metadata.OpenGraph{
			SiteName: "GoKS Site",
			Type:     "website",
		},
	}
}

func (l *testLayout) Render() *component.Node {
	return html.Html(
		html.Body(
			component.C(l.Children),
		).Class("antialiased"),
	).Attr("lang", "en")
}

type testPage struct {
	component.ComponentBase
}

func (p *testPage) Metadata() metadata.Metadata {
	return metadata.Metadata{
		Title:       "Dashboard",
		Description: "User Dashboard",
	}
}

func (p *testPage) Render() *component.Node {
	return html.Div(
		html.H1(component.Text("Dashboard Content")),
	)
}

func TestRenderDocumentHTML_AutoHead(t *testing.T) {
	page := &testPage{}
	layout := &testLayout{Children: page}

	rootComponentNode := component.C(layout)
	meta := metadata.ExtractFromTree(rootComponentNode)
	renderedNode := component.Expand(rootComponentNode, func() {}, nil)

	docHTML := renderDocumentHTML(renderedNode, meta, "<script>env</script>", "<script>lr</script>")

	expectedSubstrings := []string{
		"<!DOCTYPE html>",
		`<html lang="en"`,
		`<head>`,
		`<title>Dashboard | GoKS App</title>`,
		`<meta name="description" content="User Dashboard" />`,
		`<meta property="og:site_name" content="GoKS Site" />`,
		`<link rel="stylesheet" href="/app.css" />`,
		`<script>env</script>`,
		`<body class="antialiased">`,
		`<div id="__goks">`,
		`<h1>Dashboard Content</h1>`,
		`/wasm_exec.js`,
		`<script>lr</script>`,
		`</body>`,
		`</html>`,
	}

	for _, sub := range expectedSubstrings {
		if !strings.Contains(docHTML, sub) {
			t.Errorf("expected HTML to contain %q, but got:\n%s", sub, docHTML)
		}
	}
}

func TestRenderDocumentHTML_CustomHead(t *testing.T) {
	customLayoutNode := html.Html(
		html.Head(
			html.Script().Attr("src", "https://example.com/custom.js"),
		),
		html.Body(
			html.Div(component.Text("Hello")),
		).Class("bg-dark"),
	).Attr("lang", "id")

	meta := metadata.Metadata{
		Title:       "Custom App",
		Description: "Testing custom head",
	}

	docHTML := renderDocumentHTML(customLayoutNode, meta, "<script>env</script>", "")

	expectedSubstrings := []string{
		"<!DOCTYPE html>",
		`<html lang="id"`,
		`<head>`,
		`https://example.com/custom.js`,
		`<title>Custom App</title>`,
		`<meta name="description" content="Testing custom head" />`,
		`<body class="bg-dark">`,
		`<div id="__goks">`,
		`Hello`,
		`/wasm_exec.js`,
		`</body>`,
	}

	for _, sub := range expectedSubstrings {
		if !strings.Contains(docHTML, sub) {
			t.Errorf("expected HTML to contain %q, but got:\n%s", sub, docHTML)
		}
	}
}

type testRootWithMatcher struct {
	component.ComponentBase
}

func (r *testRootWithMatcher) HasMatchedPage(path string) bool {
	return path == "/"
}

func (r *testRootWithMatcher) Render() *component.Node {
	return html.Html(
		html.Body(html.Div(component.Text("Test"))),
	)
}

func TestStaticFileServing_PublicRoot(t *testing.T) {
	tempDir := t.TempDir()
	publicDir := filepath.Join(tempDir, "public")
	_ = os.MkdirAll(publicDir, 0755)

	pdfContent := "%PDF-1.4 sample pdf content"
	_ = os.WriteFile(filepath.Join(publicDir, "Resume.pdf"), []byte(pdfContent), 0644)

	srv := NewDev(Config{
		AppDir: tempDir,
	})
	srv.setupRoutes()

	// Test GET /Resume.pdf
	req := httptest.NewRequest("GET", "/Resume.pdf", nil)
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200 OK for /Resume.pdf, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "%PDF-1.4 sample pdf content") {
		t.Fatalf("expected PDF content, got %q", w.Body.String())
	}

	// Test GET /nonexistent (should return 404 when root implements HasMatchedPage)
	srv404 := NewDev(Config{
		AppDir: tempDir,
		Root:   &testRootWithMatcher{},
	})
	srv404.setupRoutes()

	req404 := httptest.NewRequest("GET", "/nonexistent", nil)
	w404 := httptest.NewRecorder()
	srv404.router.ServeHTTP(w404, req404)

	if w404.Code != 404 {
		t.Fatalf("expected 404 Not Found for /nonexistent, got %d", w404.Code)
	}
}

