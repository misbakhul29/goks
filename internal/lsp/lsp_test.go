package lsp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func TestValidateDocument_Valid(t *testing.T) {
	validGOX := `package app

import "github.com/misbakhul29/goks/pkg/component"

type Layout struct {
	component.ComponentBase
}

func (l *Layout) Render() *component.Node {
	return (
		<html lang="en">
			<body class="antialiased">
				<div id="app">
					<h1>Hello GoKS</h1>
				</div>
			</body>
		</html>
	)
}
`
	diags := ValidateDocument("file:///app/layout.gox", validGOX)
	if len(diags) > 0 {
		t.Errorf("expected 0 diagnostics for valid GOX, got %d: %v", len(diags), diags)
	}
}

func TestValidateDocument_AttrExpr(t *testing.T) {
	gox := `package app

import (
	"github.com/misbakhul29/goks/pkg/component"
	"github.com/misbakhul29/goks/pkg/font/google"
)

var (
	geistSans = google.Geist(google.Options{Variable: "--font-geist-sans"})
	fontGlitch = google.RubikGlitch(google.Options{Variable: "--font-rubik-glitch"})
)

type Layout struct {
	component.ComponentBase
}

func (l *Layout) Render() *component.Node {
	return (
		<html lang="en">
			<body class={geistSans.Variable() + " " + fontGlitch.Variable() + " antialiased"}>
				<div id="app">
					<h1>Hello GoKS</h1>
				</div>
			</body>
		</html>
	)
}
`
	diags := ValidateDocument("file:///app/layout.gox", gox)
	if len(diags) > 0 {
		t.Errorf("expected 0 diagnostics for GOX with attr expr, got %d: %v", len(diags), diags)
	}
}

func TestValidateDocument_UnclosedTag(t *testing.T) {
	unclosedGOX := `package app

import "github.com/misbakhul29/goks/pkg/component"

type Layout struct {
	component.ComponentBase
}

func (l *Layout) Render() *component.Node {
	return (
		<div>
			<span>Unclosed span
		</div>
	)
}
`
	diags := ValidateDocument("file:///app/layout.gox", unclosedGOX)
	if len(diags) == 0 {
		t.Fatalf("expected diagnostics for unclosed tag, got 0")
	}

	found := false
	for _, d := range diags {
		if strings.Contains(d.Message, "Unclosed tag <span") || strings.Contains(d.Message, "span") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected diagnostic mentioning unclosed tag span, got: %v", diags)
	}
}

func TestValidateDocument_MismatchedTag(t *testing.T) {
	mismatchedGOX := `package app

import "github.com/misbakhul29/goks/pkg/component"

type Layout struct {
	component.ComponentBase
}

func (l *Layout) Render() *component.Node {
	return (
		<div>
			</span>
		</div>
	)
}
`
	diags := ValidateDocument("file:///app/layout.gox", mismatchedGOX)
	if len(diags) == 0 {
		t.Fatalf("expected diagnostics for mismatched tag, got 0")
	}
}

func TestGetCompletions(t *testing.T) {
	content := `package app
func (l *Layout) Render() *component.Node {
	return (
		<
	)
}
`
	items := GetCompletions(content, Position{Line: 3, Character: 3}, "file:///app/layout.gox")
	if len(items) == 0 {
		t.Fatalf("expected tag completions, got 0")
	}

	foundDiv := false
	for _, item := range items {
		if item.Label == "div" {
			foundDiv = true
			break
		}
	}
	if !foundDiv {
		t.Errorf("expected 'div' in completions, got: %v", items)
	}

	// Test attribute completion
	attrContent := `package app
func (l *Layout) Render() *component.Node {
	return (
		<button 
	)
}
`
	attrItems := GetCompletions(attrContent, Position{Line: 3, Character: 10}, "file:///app/layout.gox")
	if len(attrItems) == 0 {
		t.Fatalf("expected attribute completions, got 0")
	}

	foundOnClick := false
	for _, item := range attrItems {
		if item.Label == "onClick" {
			foundOnClick = true
			break
		}
	}
	if !foundOnClick {
		t.Errorf("expected 'onClick' in attribute completions, got: %v", attrItems)
	}

	// Test Go code completion: user typed "r := router"
	goContent := `package app
func (p *Page) handleClick() {
	r := router
}
`
	goItems := GetCompletions(goContent, Position{Line: 2, Character: 12}, "file:///app/page.gox")
	if len(goItems) == 0 {
		t.Fatalf("expected Go code completions, got 0")
	}

	foundRouter := false
	for _, item := range goItems {
		if item.Label == "router" {
			foundRouter = true
			if !strings.Contains(item.Detail, "github.com/misbakhul29/goks/pkg/router") {
				t.Errorf("expected router detail to contain import path, got %q", item.Detail)
			}
			if len(item.AdditionalTextEdits) == 0 {
				t.Errorf("expected AdditionalTextEdits for auto-importing router, got 0")
			}
		}
	}
	if !foundRouter {
		t.Errorf("expected 'router' in Go code completions, got: %v", goItems)
	}

	// Test Dot completion: user typed "router."
	dotContent := `package app
func (p *Page) handleClick() {
	router.
}
`
	dotItems := GetCompletions(dotContent, Position{Line: 2, Character: 8}, "file:///app/page.gox")
	foundUseRouter := false
	for _, item := range dotItems {
		if item.Label == "UseRouter()" {
			foundUseRouter = true
			break
		}
	}
	if !foundUseRouter {
		t.Errorf("expected 'UseRouter()' in dot completions, got: %v", dotItems)
	}
}

func TestGetHover(t *testing.T) {
	content := `package app
func (l *Layout) Render() *component.Node {
	return (
		<button class="btn">Click</button>
	)
}
`
	hoverTag := GetHover(content, Position{Line: 3, Character: 4}) // hovering over "button"
	if hoverTag == nil {
		t.Fatalf("expected hover info for button, got nil")
	}
	if !strings.Contains(hoverTag.Contents.Value, "button") {
		t.Errorf("expected hover value to contain 'button', got %s", hoverTag.Contents.Value)
	}

	hoverAttr := GetHover(content, Position{Line: 3, Character: 12}) // hovering over "class"
	if hoverAttr == nil {
		t.Fatalf("expected hover info for class, got nil")
	}
	if !strings.Contains(hoverAttr.Contents.Value, "class") {
		t.Errorf("expected hover value to contain 'class', got %s", hoverAttr.Contents.Value)
	}

	// Hover over package name in Go code
	goHoverContent := `package app
func (p *Page) handleClick() {
	r := router.UseRouter()
}
`
	hoverPkg := GetHover(goHoverContent, Position{Line: 2, Character: 8}) // hovering over "router"
	if hoverPkg == nil {
		t.Fatalf("expected hover info for router, got nil")
	}
	if !strings.Contains(hoverPkg.Contents.Value, "github.com/misbakhul29/goks/pkg/router") {
		t.Errorf("expected hover value to contain router import path, got %s", hoverPkg.Contents.Value)
	}
}

func TestServer_Initialize(t *testing.T) {
	server := NewServer()

	reqJSON := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"processId":123}}`
	msg := fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(reqJSON), reqJSON)

	in := strings.NewReader(msg)
	var out bytes.Buffer

	// Run serve in goroutine and close input
	done := make(chan struct{})
	go func() {
		_ = server.Serve(in, &out)
		close(done)
	}()

	<-done

	outStr := out.String()
	if !strings.Contains(outStr, `"capabilities"`) {
		t.Errorf("expected initialize response to contain capabilities, got: %s", outStr)
	}

	var resp Response
	// Strip header
	headerEnd := strings.Index(outStr, "\r\n\r\n")
	if headerEnd != -1 {
		body := outStr[headerEnd+4:]
		if err := json.Unmarshal([]byte(body), &resp); err != nil {
			t.Fatalf("failed to parse response JSON: %v", err)
		}
		if resp.Error != nil {
			t.Fatalf("expected no error, got: %v", resp.Error)
		}
	}
}
