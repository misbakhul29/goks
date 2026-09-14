package compiler_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/misbakhul29/goks/internal/compiler"
)

func TestTranspile_GoldenCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains []string
	}{
		{
			name: "Basic element with text and class",
			input: `package app
func Render() *component.Node {
	return (
		<div class="card text-center" id="main">
			Hello World
		</div>
	)
}`,
			contains: []string{
				`html.Div("Hello World").Class("card text-center").ID("main")`,
			},
		},
		{
			name: "Fragment shorthand <>...</>",
			input: `package app
func Render() *component.Node {
	return (
		<>
			<h1>Title</h1>
			<p>Subtitle</p>
		</>
	)
}`,
			contains: []string{
				`component.Fragment(html.H1("Title"), html.P("Subtitle"))`,
			},
		},
		{
			name: "Explicit Fragment tag",
			input: `package app
func Render() *component.Node {
	return (
		<Fragment>
			<span>First</span>
			<span>Second</span>
		</Fragment>
	)
}`,
			contains: []string{
				`component.Fragment(html.Span("First"), html.Span("Second"))`,
			},
		},
		{
			name: "Component with props and expressions",
			input: `package app
func Render() *component.Node {
	return (
		<div>
			<c.Hero title="GoKS Framework" count={42} active={true} />
		</div>
	)
}`,
			contains: []string{
				`component.C(&c.Hero{active: true, count: 42, title: "GoKS Framework"})`,
			},
		},
		{
			name: "Event handler binding",
			input: `package app
func Render() *component.Node {
	return (
		<button class="btn" onClick={handleClick} onInput={handleInput}>
			Click Me
		</button>
	)
}`,
			contains: []string{
				`.On("onClick", handleClick)`,
				`.On("onInput", handleInput)`,
				`.Class("btn")`,
			},
		},
		{
			name: "HTML comments are safely ignored",
			input: `package app
func Render() *component.Node {
	return (
		<div>
			<!-- This is a comment that should not appear in AST -->
			<span>Content</span>
		</div>
	)
}`,
			contains: []string{
				`html.Div(html.Span("Content"))`,
			},
		},
		{
			name: "Dynamic expressions with component.Any",
			input: `package app
func Render() *component.Node {
	return (
		<div class="user-info">
			{userName}
			<span>{userAge}</span>
		</div>
	)
}`,
			contains: []string{
				`component.Any(userName)`,
				`component.Any(userAge)`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := compiler.Transpile(tt.input)
			if err != nil {
				t.Fatalf("Transpile() unexpected error: %v", err)
			}
			for _, exp := range tt.contains {
				if !strings.Contains(out, exp) {
					t.Errorf("Transpile output missing expected snippet:\nExpected: %s\nGot:\n%s", exp, out)
				}
			}
		})
	}
}

func TestTranspile_DeterministicOutput(t *testing.T) {
	input := `package app
func Render() *component.Node {
	return (
		<div zIndex={10} class="box" id="unique" data-test="unit" onClick={fn}>
			<c.Item zebra={true} alpha="start" count={100} />
		</div>
	)
}`

	out1, err1 := compiler.Transpile(input)
	if err1 != nil {
		t.Fatalf("run 1 failed: %v", err1)
	}

	for i := 0; i < 10; i++ {
		out2, err2 := compiler.Transpile(input)
		if err2 != nil {
			t.Fatalf("run %d failed: %v", i+2, err2)
		}
		if out1 != out2 {
			t.Fatalf("non-deterministic output detected on run %d:\nFirst:\n%s\nSecond:\n%s", i+2, out1, out2)
		}
	}
}

func TestTranspile_CompileErrorDiagnostics(t *testing.T) {
	t.Run("Unclosed return parenthesis", func(t *testing.T) {
		input := `package app

func Render() *component.Node {
	return (
		<div>Hello
}`
		_, err := compiler.TranspileWithSource(input, "page.gox")
		if err == nil {
			t.Fatal("expected compilation error for unclosed return")
		}

		var compErr *compiler.CompileError
		if !errors.As(err, &compErr) {
			t.Fatalf("expected *compiler.CompileError, got %T: %v", err, err)
		}

		if compErr.File != "page.gox" {
			t.Errorf("expected File 'page.gox', got %q", compErr.File)
		}
		if compErr.Line != 4 {
			t.Errorf("expected Line 4, got %d", compErr.Line)
		}
		if !strings.Contains(err.Error(), "unclosed 'return (' block") {
			t.Errorf("expected error message about unclosed block, got: %v", err)
		}
		if !strings.Contains(err.Error(), "4 |") || !strings.Contains(err.Error(), "^") {
			t.Errorf("expected visual snippet with caret, got: %s", err.Error())
		}
	})

	t.Run("Mismatched XML tags", func(t *testing.T) {
		input := `package app

func Render() *component.Node {
	return (
		<div>
			<span>Mismatched</div>
		</div>
	)
}`
		_, err := compiler.TranspileWithSource(input, "components/card.gox")
		if err == nil {
			t.Fatal("expected compilation error for mismatched tags")
		}

		var compErr *compiler.CompileError
		if !errors.As(err, &compErr) {
			t.Fatalf("expected *compiler.CompileError, got %T: %v", err, err)
		}

		if compErr.File != "components/card.gox" {
			t.Errorf("expected File 'components/card.gox', got %q", compErr.File)
		}
		if compErr.Line < 4 {
			t.Errorf("expected Line >= 4, got %d", compErr.Line)
		}
		if !strings.Contains(err.Error(), "components/card.gox:") {
			t.Errorf("expected filename prefix in error, got: %s", err.Error())
		}
	})
}
