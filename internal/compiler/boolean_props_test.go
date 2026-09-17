package compiler_test

import (
	"strings"
	"testing"

	"github.com/misbakhul29/goks/internal/compiler"
)

func TestTranspile_BooleanPropsShorthand(t *testing.T) {
	input := `package app

func Render() *component.Node {
	return (
		<div>
			<input type="text" required autofocus disabled />
			<Button primary disabled />
			<button disabled>Click</button>
		</div>
	)
}`

	out, err := compiler.Transpile(input)
	if err != nil {
		t.Fatalf("unexpected transpile error: %v", err)
	}

	// input element should have .Attr("autofocus", true), .Attr("disabled", true), .Attr("required", true)
	if !strings.Contains(out, `.Attr("autofocus", true)`) {
		t.Errorf("expected .Attr(\"autofocus\", true), got:\n%s", out)
	}
	if !strings.Contains(out, `.Attr("disabled", true)`) {
		t.Errorf("expected .Attr(\"disabled\", true), got:\n%s", out)
	}
	if !strings.Contains(out, `.Attr("required", true)`) {
		t.Errorf("expected .Attr(\"required\", true), got:\n%s", out)
	}

	// Button component should have disabled: true, primary: true
	if !strings.Contains(out, `component.C(&Button{disabled: true, primary: true})`) {
		t.Errorf("expected component.C(&Button{disabled: true, primary: true}), got:\n%s", out)
	}

	// standard button element should have .Attr("disabled", true)
	if !strings.Contains(out, `html.Button("Click").Attr("disabled", true)`) {
		t.Errorf("expected html.Button(\"Click\").Attr(\"disabled\", true), got:\n%s", out)
	}
}
