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
		if len(args) == 0 {
			return nil
		}
		e := args[0]

		// Only intercept primary (left) mouse clicks without modifier keys (allow Ctrl/Cmd/Shift/Alt clicks to open new tab/window)
		if e.Get("button").Int() != 0 ||
			e.Get("metaKey").Bool() ||
			e.Get("ctrlKey").Bool() ||
			e.Get("shiftKey").Bool() ||
			e.Get("altKey").Bool() {
			return nil
		}

		target := e.Get("target")

		// Find closest anchor tag up the tree
		for !target.IsNull() && !target.IsUndefined() && target.Get("tagName").String() != "A" {
			target = target.Get("parentElement")
		}

		if !target.IsNull() && !target.IsUndefined() {
			// Do not intercept if link specifies target (like _blank) or download attribute or data-native
			targetAttr := target.Get("getAttribute").Call("call", target, "target")
			if !targetAttr.IsNull() && !targetAttr.IsUndefined() && targetAttr.String() != "" && targetAttr.String() != "_self" {
				return nil
			}
			if target.Get("hasAttribute").Call("call", target, "download").Bool() {
				return nil
			}
			if target.Get("hasAttribute").Call("call", target, "data-native").Bool() {
				return nil
			}

			hrefAttr := target.Get("getAttribute").Call("call", target, "href")
			if hrefAttr.IsNull() || hrefAttr.IsUndefined() {
				return nil
			}
			href := hrefAttr.String()

			// Only intercept local links starting with / that are not static file downloads
			if len(href) > 0 && href[0] == '/' && !HasFileExtension(href) {
				e.Call("preventDefault")
				window.Get("history").Call("pushState", nil, "", href)
				CurrentPath.Set(href)
			}
		}
		return nil
	}))

}
