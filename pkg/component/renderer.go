//go:build js && wasm

// Package component - DOM renderer for the WASM runtime.
// This file uses syscall/js to apply virtual DOM patches to the real browser DOM.

package component

import (
	"fmt"
	"strings"
	"syscall/js"
)

// Renderer applies a virtual DOM tree to the real browser DOM.
type Renderer struct {
	root    js.Value // The root DOM element (e.g., document.getElementById("app"))
	current *Node    // The last rendered virtual DOM tree
}

// NewRenderer creates a renderer mounted at the given DOM selector.
func NewRenderer(selector string) *Renderer {
	doc := js.Global().Get("document")
	el := doc.Call("querySelector", selector)
	if el.IsNull() || el.IsUndefined() {
		panic(fmt.Sprintf("goks: mount point '%s' not found in DOM", selector))
	}
	return &Renderer{root: el}
}

// Render performs an initial render or reconciliation update.
func (r *Renderer) Render(node *Node) {
	if r.current == nil {
		// Initial mount: clear root and create the full tree
		r.root.Set("innerHTML", "")
		domNode := r.createDOMNode(node)
		r.root.Call("appendChild", domNode)
	} else {
		// Incremental update: diff and patch
		patches := Reconcile(r.current, node)
		r.applyPatches(r.root, patches, r.current, node)
	}
	r.current = node
}

// createDOMNode creates a real DOM node from a virtual node.
func (r *Renderer) createDOMNode(node *Node) js.Value {
	doc := js.Global().Get("document")

	switch node.Type {
	case NodeTypeText:
		return doc.Call("createTextNode", node.Text)

	case NodeTypeFragment:
		frag := doc.Call("createDocumentFragment")
		for _, child := range node.Children {
			frag.Call("appendChild", r.createDOMNode(child))
		}
		return frag

	case NodeTypeComponent:
		if node.Component != nil {
			return r.createDOMNode(node.Component.Render())
		}
		return doc.Call("createTextNode", "")

	default: // NodeTypeElement
		el := doc.Call("createElement", node.Tag)
		r.applyProps(el, node.Props, nil)
		for _, child := range node.Children {
			el.Call("appendChild", r.createDOMNode(child))
		}
		
		// Fire OnMount if it's attached to a component
		if node.Component != nil {
			if cb, ok := node.Component.(interface{ setMounted(bool) }); ok {
				cb.setMounted(true)
			}
			if m, ok := node.Component.(Mounter); ok {
				// Gunakan setTimeout agar OnMount berjalan SETELAH DOM benar-benar ter-append (next tick)
				js.Global().Call("setTimeout", js.FuncOf(func(_ js.Value, _ []js.Value) any {
					m.OnMount()
					return nil
				}), 0)
			}
		}
		
		return el
	}
}

// applyProps sets attributes and event listeners on a DOM element.
func (r *Renderer) applyProps(el js.Value, newProps, oldProps Props) {
	// Remove old props that no longer exist
	for k := range oldProps {
		if _, exists := newProps[k]; !exists {
			if strings.HasPrefix(k, "on") {
				el.Call("removeEventListener", strings.ToLower(k[2:]), js.Null())
			} else {
				el.Call("removeAttribute", k)
			}
		}
	}

	// Set new / updated props
	for k, v := range newProps {
		if strings.HasPrefix(k, "on") {
			// Event handler: onClick → "click"
			event := strings.ToLower(k[2:])
			if fn, ok := v.(func(js.Value, []js.Value) any); ok {
				el.Call("addEventListener", event, js.FuncOf(fn))
			} else if fn, ok := v.(func()); ok {
				el.Call("addEventListener", event, js.FuncOf(func(_ js.Value, _ []js.Value) any {
					fn()
					return nil
				}))
			}
		} else if k == "class" {
			el.Set("className", fmt.Sprintf("%v", v))
		} else if k == "style" {
			el.Set("style", fmt.Sprintf("%v", v))
		} else if k == "innerHTML" {
			el.Set("innerHTML", fmt.Sprintf("%v", v))
		} else {
			el.Call("setAttribute", k, fmt.Sprintf("%v", v))
		}
	}
}

// applyPatches applies a list of patches to the real DOM.
func (r *Renderer) applyPatches(parent js.Value, patches []Patch, oldTree, newTree *Node) {
	children := parent.Get("childNodes")

	for _, p := range patches {
		switch p.Type {
		case PatchCreate:
			newDom := r.createDOMNode(p.NewNode)
			parent.Call("appendChild", newDom)

		case PatchRemove:
			if p.Index < children.Length() {
				child := children.Index(p.Index)
				parent.Call("removeChild", child)
				r.triggerUnmount(p.OldNode)
			}

		case PatchReplace:
			newDom := r.createDOMNode(p.NewNode)
			if p.Index < children.Length() {
				old := children.Index(p.Index)
				parent.Call("replaceChild", newDom, old)
				r.triggerUnmount(p.OldNode)
			} else {
				parent.Call("appendChild", newDom)
			}

		case PatchText:
			if p.Index < children.Length() {
				children.Index(p.Index).Set("textContent", p.NewNode.Text)
			}

		case PatchUpdate:
			if p.Index < children.Length() {
				el := children.Index(p.Index)
				r.applyProps(el, p.NewNode.Props, p.OldNode.Props)
				
				// Fire OnUpdate if it has a component
				if p.NewNode.Component != nil {
					if up, ok := p.NewNode.Component.(Updater); ok {
						up.OnUpdate(p.OldNode.Props)
					}
				}
			}
		}
	}
}

func (r *Renderer) triggerUnmount(node *Node) {
	if node == nil {
		return
	}
	if node.Component != nil {
		if cb, ok := node.Component.(interface{ setMounted(bool) }); ok {
			cb.setMounted(false)
		}
		if u, ok := node.Component.(Unmounter); ok {
			u.OnUnmount()
		}
	}
	for _, child := range node.Children {
		r.triggerUnmount(child)
	}
}
