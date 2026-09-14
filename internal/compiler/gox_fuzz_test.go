package compiler_test

import (
	"testing"

	"github.com/misbakhul29/goks/internal/compiler"
)

// FuzzTranspile ensures that arbitrary random inputs to the GOX compiler
// never cause unhandled panics or memory crashes.
func FuzzTranspile(f *testing.F) {
	// Seed corpus with realistic and edge-case samples
	seeds := []string{
		`package app; func Render() *component.Node { return (<div>Hello</div>) }`,
		`package app; func Render() *component.Node { return (<div class="test"><span>Child</span></div>) }`,
		`package app; func Render() *component.Node { return (<><h1>Title</h1><p>Body</p></>) }`,
		`package app; func Render() *component.Node { return (<c.Hero count={123} title="test" />) }`,
		`return (<div>`,
		`return (<div></span>)`,
		`return (<div class={func(){}()}>)`,
		`return (<!-- comment --><b>text</b>)`,
		`return (<button onClick={handleClick}>Click</button>)`,
		`random string without return statement`,
		`return ()`,
		`return (<div attr="value" unclosed`,
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		// Fuzz target should handle errors gracefully without panicking
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("Transpile panicked on input %q: %v", input, r)
			}
		}()

		_, _ = compiler.Transpile(input)
	})
}
