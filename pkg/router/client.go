//go:build js && wasm

package router

import (
	"syscall/js"
)

// InitClientRouter wires up the browser's history API.
// It intercepts popstate events and link clicks to update CurrentPath.
// Call this once in your client/main.go.
func InitClientRouter() {
	window := js.Global().Get("window")

	// Set initial path
	CurrentPath.Set(window.Get("location").Get("pathname").String())

	// Override Push for WASM
	Push = func(path string) {
		if HasFileExtension(path) {
			js.Global().Get("window").Get("location").Set("href", path)
			return
		}
		js.Global().Get("window").Get("history").Call("pushState", nil, "", path)
		CurrentPath.Set(path)
	}

	// Override Replace for WASM
	Replace = func(path string) {
		if HasFileExtension(path) {
			js.Global().Get("window").Get("location").Call("replace", path)
			return
		}
		js.Global().Get("window").Get("history").Call("replaceState", nil, "", path)
		CurrentPath.Set(path)
	}

	// Override Back for WASM
	Back = func() {
		js.Global().Get("window").Get("history").Call("back")
	}

	// Override Forward for WASM
	Forward = func() {
		js.Global().Get("window").Get("history").Call("forward")
	}

	window.Call("addEventListener", "popstate", js.FuncOf(func(_ js.Value, _ []js.Value) any {
		CurrentPath.Set(window.Get("location").Get("pathname").String())
		return nil
	}))

	// Intercept link clicks globally
	js.Global().Get("document").Call("addEventListener", "click", js.FuncOf(func(_ js.Value, args []js.Value) any {
		e := args[0]
		target := e.Get("target")

		// Find closest anchor tag up the tree
		for !target.IsNull() && !target.IsUndefined() && target.Get("tagName").String() != "A" {
			target = target.Get("parentElement")
		}

		if !target.IsNull() && !target.IsUndefined() {
			href := target.Get("getAttribute").Call("call", target, "href").String()
			// Only intercept local links starting with / that are not static file downloads
			if len(href) > 0 && href[0] == '/' && !HasFileExtension(href) && !target.Get("hasAttribute").Call("call", target, "data-native").Bool() {
				e.Call("preventDefault")
				window.Get("history").Call("pushState", nil, "", href)
				CurrentPath.Set(href)
			}
		}
		return nil
	}))

}
