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
