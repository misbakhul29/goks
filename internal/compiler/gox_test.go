package compiler_test

import (
	"strings"
	"testing"

	"github.com/misbakhul29/goks/internal/compiler"
)

func TestTranspile_AttrWithQuotedStringExpr(t *testing.T) {
	// This used to fail because the XML parser choked on " inside {}
	input := `package app

func (l *Layout) Render() *component.Node {
	return (
		<body class={geistSans.Variable() + " " + fontGlitch.Variable() + " antialiased"}>
		</body>
	)
}`
	out, err := compiler.Transpile(input)
	if err != nil {
		t.Fatalf("Transpile() error = %v", err)
	}
	if !strings.Contains(out, `geistSans.Variable() + " " + fontGlitch.Variable() + " antialiased"`) {
		t.Errorf("Transpile() did not preserve the full expression, got:\n%s", out)
	}
}

func TestTranspile_AttrWithGoogleClasses(t *testing.T) {
	// The simple form: google.Classes(font1, font2, "antialiased")
	input := `package app

func (l *Layout) Render() *component.Node {
	return (
		<body class={google.Classes(geistSans, fontGlitch, "antialiased")}>
		</body>
	)
}`
	out, err := compiler.Transpile(input)
	if err != nil {
		t.Fatalf("Transpile() error = %v", err)
	}
	if !strings.Contains(out, `google.Classes(geistSans, fontGlitch, "antialiased")`) {
		t.Errorf("Transpile() did not preserve google.Classes() call, got:\n%s", out)
	}
}

func TestTranspile_AttrSimpleDynamic(t *testing.T) {
	// Existing {expr} without quotes should still work
	input := `package app

func (l *Layout) Render() *component.Node {
	return (
		<div class={myVar}>
		</div>
	)
}`
	out, err := compiler.Transpile(input)
	if err != nil {
		t.Fatalf("Transpile() error = %v", err)
	}
	if !strings.Contains(out, `.Class(myVar)`) {
		t.Errorf("Transpile() did not generate .Class(myVar), got:\n%s", out)
	}
}

func TestTranspile_AttrStaticClass(t *testing.T) {
	// Static class string should still work
	input := `package app

func (l *Layout) Render() *component.Node {
	return (
		<div class="antialiased bg-black">
		</div>
	)
}`
	out, err := compiler.Transpile(input)
	if err != nil {
		t.Fatalf("Transpile() error = %v", err)
	}
	if !strings.Contains(out, `.Class("antialiased bg-black")`) {
		t.Errorf("Transpile() did not generate correct static class, got:\n%s", out)
	}
}

func TestTranspile_AttrNestedBraces(t *testing.T) {
	// Expressions with nested braces (e.g. function call with map literal)
	input := `package app

func (l *Layout) Render() *component.Node {
	return (
		<div style={fmt.Sprintf("color: %s", colors["red"])}>
		</div>
	)
}`
	out, err := compiler.Transpile(input)
	if err != nil {
		t.Fatalf("Transpile() error = %v", err)
	}
	if !strings.Contains(out, `fmt.Sprintf("color: %s", colors["red"])`) {
		t.Errorf("Transpile() did not preserve nested expression, got:\n%s", out)
	}
}
