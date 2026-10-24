package compiler_test

import (
	"strings"
	"testing"
	"time"

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

func TestTranspile_NoInfiniteLoopOnUnexpectedAttrs(t *testing.T) {
	cases := []string{
		`func Render() *component.Node { return (<div {props}></div>) }`,
		`func Render() *component.Node { return (<div {...props}></div>) }`,
		`func Render() *component.Node { return (<button onClick= {fn}>Click</button>) }`,
		`func Render() *component.Node { return (<div @click="fn"></div>) }`,
		`func Render() *component.Node { return (<div [attr]="val"></div>) }`,
	}

	for _, tc := range cases {
		done := make(chan struct{})
		go func(c string) {
			_, _ = compiler.Transpile(c)
			close(done)
		}(tc)

		select {
		case <-done:
			// finished within timeout
		case <-time.After(500 * time.Millisecond):
			t.Fatalf("Transpile hung in infinite loop on input: %s", tc)
		}
	}
}
