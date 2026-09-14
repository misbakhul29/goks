package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

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
	component.ClientBase
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
			html.Button().Attr("onClick", "handleClick"),
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

type staticOnlyPage struct {
	component.ComponentBase
}

func (p *staticOnlyPage) Render() *component.Node {
	return html.Div(html.H1(component.Text("Pure Static Page")))
}

func TestRenderDocumentHTML_ZeroWASMStatic(t *testing.T) {
	page := &staticOnlyPage{}
	layout := &testLayout{Children: page}

	rootComponentNode := component.C(layout)
	meta := metadata.ExtractFromTree(rootComponentNode)
	renderedNode := component.Expand(rootComponentNode, func() {}, nil)

	docHTML := renderDocumentHTML(renderedNode, meta, "", "")

	// Static pages should NOT contain wasm_exec.js or app.wasm scripts (0-WASM)
	if strings.Contains(docHTML, "/wasm_exec.js") {
		t.Errorf("expected 0-WASM pure HTML output without /wasm_exec.js, but got:\n%s", docHTML)
	}
	if strings.Contains(docHTML, "app.wasm") {
		t.Errorf("expected 0-WASM pure HTML output without app.wasm, but got:\n%s", docHTML)
	}
	if !strings.Contains(docHTML, "Pure Static Page") {
		t.Errorf("expected rendered content in static output")
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

	// Test GET /public/ does not list directory
	reqDir := httptest.NewRequest("GET", "/public/", nil)
	wDir := httptest.NewRecorder()
	srv.router.ServeHTTP(wDir, reqDir)

	if wDir.Code != 404 {
		t.Fatalf("expected 404 for directory listing on /public/, got %d", wDir.Code)
	}
}

func TestServer_HealthAndReady(t *testing.T) {
	srv := New(Config{AppDir: t.TempDir()})
	srv.setupRoutes()

	// 1. Healthz probe
	reqHealth := httptest.NewRequest("GET", "/_goks/healthz", nil)
	recHealth := httptest.NewRecorder()
	srv.Router().ServeHTTP(recHealth, reqHealth)

	if recHealth.Code != 200 {
		t.Fatalf("expected 200 OK for healthz, got %d", recHealth.Code)
	}
	if !strings.Contains(recHealth.Body.String(), `"status":"ok"`) {
		t.Fatalf("expected ok status in healthz, got %s", recHealth.Body.String())
	}

	// 2. Ready probe
	reqReady := httptest.NewRequest("GET", "/_goks/ready", nil)
	recReady := httptest.NewRecorder()
	srv.Router().ServeHTTP(recReady, reqReady)

	if recReady.Code != 200 {
		t.Fatalf("expected 200 OK for ready probe, got %d", recReady.Code)
	}
	if !strings.Contains(recReady.Body.String(), `"status":"ready"`) {
		t.Fatalf("expected ready status in ready probe, got %s", recReady.Body.String())
	}
}

func TestServer_GracefulShutdown(t *testing.T) {
	srv := New(Config{
		Host:   "127.0.0.1",
		Port:   39482,
		AppDir: t.TempDir(),
	})

	errChan := make(chan error, 1)
	go func() {
		errChan <- srv.Start()
	}()

	// Wait briefly for server to bind and start listening
	var resp *http.Response
	var err error
	for i := 0; i < 20; i++ {
		time.Sleep(25 * time.Millisecond)
		resp, err = http.Get("http://127.0.0.1:39482/_goks/healthz")
		if err == nil && resp.StatusCode == 200 {
			_ = resp.Body.Close()
			break
		}
	}
	if err != nil {
		t.Fatalf("failed to reach server during startup: %v", err)
	}

	// Trigger graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		t.Fatalf("unexpected shutdown error: %v", err)
	}

	// Ensure Start() returned cleanly without ErrServerClosed
	startErr := <-errChan
	if startErr != nil {
		t.Fatalf("expected clean exit from Start(), got %v", startErr)
	}
}

func TestServer_ConcurrentRequests(t *testing.T) {
	srv := New(Config{AppDir: t.TempDir()})
	srv.setupRoutes()

	const concurrency = 50
	var wg sync.WaitGroup
	wg.Add(concurrency)

	for i := 0; i < concurrency; i++ {
		go func(id int) {
			defer wg.Done()
			req := httptest.NewRequest("GET", "/_goks/healthz", nil)
			rec := httptest.NewRecorder()
			srv.Router().ServeHTTP(rec, req)

			if rec.Code != 200 {
				t.Errorf("request %d failed with status %d", id, rec.Code)
			}
		}(i)
	}
	wg.Wait()
}
