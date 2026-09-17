package compiler_test

import (
	"strings"
	"testing"

	"github.com/misbakhul29/goks/internal/compiler"
)

func TestTranspile_LocalComponent(t *testing.T) {
	input := `package app

func Render() *component.Node {
	return (
		<div>
			<Button title="Click Me" primary={true} />
			<Card class="p-4">
				<span>Hello World</span>
			</Card>
		</div>
	)
}`

	out, err := compiler.Transpile(input)
	if err != nil {
		t.Fatalf("unexpected transpile error: %v", err)
	}

	// Button is uppercase -> component.C(&Button{primary: true, title: "Click Me"})
	if !strings.Contains(out, `component.C(&Button{primary: true, title: "Click Me"})`) {
		t.Errorf("expected local component &Button{...} instantiation, got:\n%s", out)
	}

	// Card has children -> component.C(&Card{..., Children: component.Fragment(...)})
	if !strings.Contains(out, `component.C(&Card{`) || !strings.Contains(out, `Children: component.Fragment(html.Span("Hello World"))`) {
		t.Errorf("expected local component &Card{...} with Children, got:\n%s", out)
	}

	// div and span remain html.Div and html.Span
	if !strings.Contains(out, `html.Div(`) || !strings.Contains(out, `html.Span("Hello World")`) {
		t.Errorf("expected standard HTML tags to remain html.Div and html.Span, got:\n%s", out)
	}
}
