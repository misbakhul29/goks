package lsp

import (
	"testing"
)

func TestValidateDocument_CompilerErrorLocationMapping(t *testing.T) {
	// A .gox file with a syntax error on line 8 (invalid XML closing tag)
	content := `package app

import "github.com/misbakhul29/goks/pkg/component"

func Render() *component.Node {
	return (
		<div>
			<span>valid</span></mismatched>
		</div>
	)
}
`
	diags := ValidateDocument("file:///app/page.gox", content)
	if len(diags) == 0 {
		t.Fatal("expected compiler diagnostics for mismatched closing tag")
	}

	d := diags[0]
	// Line 8 in 1-indexed is Line 7 in 0-indexed LSP Range
	if d.Range.Start.Line == 0 {
		t.Errorf("expected diagnostic line to be mapped to the error line (>0), but got line 0")
	}

	if d.Range.Start.Line < 5 || d.Range.Start.Line > 9 {
		t.Errorf("expected diagnostic line between 5 and 9, got %d", d.Range.Start.Line)
	}
}
