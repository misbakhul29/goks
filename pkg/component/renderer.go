//go:build js && wasm

// Package component - DOM renderer for the WASM runtime.
// This file uses syscall/js to apply virtual DOM patches to the real browser DOM.

package component

import (
	"fmt"
	"strings"
	"syscall/js"
)

// Renderer handles rendering a virtual DOM tree to the real DOM.
type Renderer struct {
	rootElement js.Value
	current     *Node
	RootFiber   *FiberNode
}

// NewRenderer creates a new renderer attached to a root DOM element ID.
func NewRenderer(rootID string) *Renderer {
	document := js.Global().Get("document")
	id := strings.TrimPrefix(rootID, "#")
	rootElement := document.Call("getElementById", id)
	if rootElement.IsNull() || rootElement.IsUndefined() {
		rootElement = document.Call("querySelector", rootID)
	}
	if rootElement.IsNull() || rootElement.IsUndefined() {
		panic("root element not found")
	}

	return &Renderer{
		rootElement: rootElement,
		RootFiber:   &FiberNode{},
	}
}

// Render performs an initial render or reconciliation update.
func (r *Renderer) Render(node *Node) {
	// Reset RootFiber state for this render pass
	r.RootFiber.ChildIndex = 0
	r.RootFiber.HookIndex = 0

	newTree := Expand(node, func() {
		r.Render(node) // re-render callback
	}, r.RootFiber)

	if r.current == nil {
		// Initial mount: clear root and create the full tree
		r.rootElement.Set("innerHTML", "")
		domNode := r.createDOMNode(newTree)
		r.rootElement.Call("appendChild", domNode)
	} else {
		// Incremental update: diff and patch
		patches := Reconcile(r.current, newTree)
		r.applyPatches(r.rootElement, patches, r.current, newTree)
	}
	r.current = newTree
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

var nextElementID int32 = 1
var elementListeners = make(map[int32]map[string]js.Func)

// applyProps sets attributes and event listeners on a DOM element.
func (r *Renderer) applyProps(el js.Value, newProps, oldProps Props) {
	idVal := el.Get("_goks_id")
	var elID int32
	if idVal.IsUndefined() || idVal.IsNull() {
		elID = nextElementID
		nextElementID++
		el.Set("_goks_id", elID)
	} else {
		elID = int32(idVal.Int())
	}

	if elementListeners[elID] == nil {
		elementListeners[elID] = make(map[string]js.Func)
	}
	listeners := elementListeners[elID]

	// Remove old props that no longer exist
	for k := range oldProps {
		if _, exists := newProps[k]; !exists {
			if strings.HasPrefix(k, "on") {
				event := strings.ToLower(k[2:])
				if oldJsFn, ok := listeners[event]; ok {
					el.Call("removeEventListener", event, oldJsFn)
					oldJsFn.Release()
					delete(listeners, event)
				}
			} else {
				el.Call("removeAttribute", k)
			}
		}
	}

	// Set new / updated props
	for k, v := range newProps {
		if strings.HasPrefix(k, "on") {
			event := strings.ToLower(k[2:])

			// Remove old listener if it exists to prevent stacking
			if oldJsFn, ok := listeners[event]; ok {
				el.Call("removeEventListener", event, oldJsFn)
				oldJsFn.Release()
				delete(listeners, event)
			}

			// Add new listener
			var jsFn js.Func
			if fn, ok := v.(func(js.Value, []js.Value) any); ok {
				jsFn = js.FuncOf(fn)
			} else if fn, ok := v.(func()); ok {
				jsFn = js.FuncOf(func(_ js.Value, _ []js.Value) any {
					fn()
					return nil
				})
			}

			listeners[event] = jsFn
			el.Call("addEventListener", event, jsFn)
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
	for _, p := range patches {
		targetParent := parent
		for _, idx := range p.Path {
			targetParent = targetParent.Get("childNodes").Index(idx)
		}

		children := targetParent.Get("childNodes")

		switch p.Type {
		case PatchCreate:
			newDom := r.createDOMNode(p.NewNode)
			targetParent.Call("appendChild", newDom)

		case PatchRemove:
			if p.Index < children.Length() {
				child := children.Index(p.Index)
				r.cleanupListeners(child)
				targetParent.Call("removeChild", child)
				r.triggerUnmount(p.OldNode)
			}

		case PatchReplace:
			newDom := r.createDOMNode(p.NewNode)
			if p.Index < children.Length() {
				old := children.Index(p.Index)
				r.cleanupListeners(old)
				targetParent.Call("replaceChild", newDom, old)
				r.triggerUnmount(p.OldNode)
			} else {
				targetParent.Call("appendChild", newDom)
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

// cleanupListeners removes event listeners and frees js.Func objects for a DOM node and its descendants.
func (r *Renderer) cleanupListeners(el js.Value) {
	if el.IsNull() || el.IsUndefined() {
		return
	}
	idVal := el.Get("_goks_id")
	if !idVal.IsUndefined() && !idVal.IsNull() {
		elID := int32(idVal.Int())
		if listeners, ok := elementListeners[elID]; ok {
			for event, jsFn := range listeners {
				el.Call("removeEventListener", event, jsFn)
				jsFn.Release()
			}
			delete(elementListeners, elID)
		}
	}

	// recursively cleanup children
	children := el.Get("childNodes")
	if !children.IsUndefined() && !children.IsNull() {
		length := children.Length()
		for i := 0; i < length; i++ {
			r.cleanupListeners(children.Index(i))
		}
	}
}
