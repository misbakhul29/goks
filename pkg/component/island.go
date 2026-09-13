package component

import "strings"

// ClientComponent is an interface for components that explicitly declare
// whether they require client-side WebAssembly hydration.
type ClientComponent interface {
	ClientHydrated() bool
}

// ClientBase can be embedded in any component struct to mark it as requiring
// client-side WebAssembly hydration (interactive island).
//
//	type Counter struct {
//	    component.ComponentBase
//	    component.ClientBase
//	}
type ClientBase struct{}

// ClientHydrated returns true to mark this component as an interactive client island.
func (ClientBase) ClientHydrated() bool { return true }

// NeedsHydration recursively inspects a node tree to check if any interactive
// event handlers (e.g., onClick, onChange, onInput) or client island markers exist.
func NeedsHydration(node *Node) bool {
	if node == nil {
		return false
	}

	if node.Component != nil {
		if c, ok := node.Component.(ClientComponent); ok && c.ClientHydrated() {
			return true
		}
	}

	if node.Props != nil {
		for k, v := range node.Props {
			// Any event handler (onClick, onInput, etc.) requires client hydration
			if strings.HasPrefix(k, "on") && len(k) > 2 && v != nil {
				return true
			}
		}
	}

	for _, child := range node.Children {
		if NeedsHydration(child) {
			return true
		}
	}

	return false
}
