//go:build js && wasm

// Package client provides the GoKS client-side WASM runtime.
// It bootstraps component rendering in the browser.

package client

import (
	"fmt"
	"syscall/js"

	"github.com/misbakhul29/goks/pkg/component"
)

// App is the root GoKS client application.
type App struct {
	renderer *component.Renderer
	root     component.Renderable
}

// Mount creates a new App mounted at a CSS selector and renders the root component.
//
//	func main() {
//	    client.Mount("#app", &MyRootComponent{})
//	    select {} // keep WASM alive
//	}
func Mount(selector string, root component.Renderable) *App {
	renderer := component.NewRenderer(selector)

	app := &App{
		renderer: renderer,
		root:     root,
	}

	// Initial render will expand and mount components
	app.render()

	return app
}

func (a *App) render() {
	node := a.root.Render()
	// Expand handles sub-component rerender bindings
	expanded := component.Expand(node, a.render)
	
	a.renderer.Render(expanded)
}

// -----------------------------------------------------------------------
// JS interop helpers — usable from component code
// -----------------------------------------------------------------------

// Console provides access to browser console methods.
var Console = struct {
	Log   func(args ...any)
	Warn  func(args ...any)
	Error func(args ...any)
}{
	Log: func(args ...any) {
		js.Global().Get("console").Call("log", toJSValues(args)...)
	},
	Warn: func(args ...any) {
		js.Global().Get("console").Call("warn", toJSValues(args)...)
	},
	Error: func(args ...any) {
		js.Global().Get("console").Call("error", toJSValues(args)...)
	},
}

func toJSValues(args []any) []any {
	out := make([]any, len(args))
	for i, a := range args {
		out[i] = js.ValueOf(a)
	}
	return out
}

// Fetch makes a browser fetch() request and returns the response body as a string.
// For a full fetch API, use a dedicated HTTP client package.
func Fetch(url string) (string, error) {
	ch := make(chan string, 1)
	errCh := make(chan error, 1)

	js.Global().Call("fetch", url).
		Call("then", js.FuncOf(func(_ js.Value, args []js.Value) any {
			args[0].Call("text").Call("then", js.FuncOf(func(_ js.Value, a []js.Value) any {
				ch <- a[0].String()
				return nil
			}))
			return nil
		})).
		Call("catch", js.FuncOf(func(_ js.Value, args []js.Value) any {
			errCh <- fmt.Errorf("fetch error: %s", args[0].Get("message").String())
			return nil
		}))

	select {
	case body := <-ch:
		return body, nil
	case err := <-errCh:
		return "", err
	}
}
