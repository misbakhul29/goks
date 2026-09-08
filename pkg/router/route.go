package router

import (
	"github.com/misbakhulmunir/goks/pkg/component"
)

// CurrentPath is a global store holding the current URL path.
var CurrentPath = component.NewStore("/")

// Push navigates to a new path programmatically (overridden in WASM).
var Push = func(path string) {
	CurrentPath.Set(path)
}

// PageRoute is a UI component that renders its child only if the current path matches.
// Works for both SSR (backend) and SPA (WASM client).
type PageRoute struct {
	component.ComponentBase
	Path      string
	Component component.Renderable
	Exact     bool
	unsub     func()
}

func (r *PageRoute) OnMount() {
	r.unsub = CurrentPath.Subscribe(func(_ string) {
		r.Rerender()
	})
}

func (r *PageRoute) OnUnmount() {
	if r.unsub != nil {
		r.unsub()
	}
}

func (r *PageRoute) Render() *component.Node {
	current := CurrentPath.Get()
	match := false
	if r.Exact {
		match = current == r.Path
	} else {
		// Basic prefix matching for non-exact routes
		match = current == r.Path || (len(current) > len(r.Path) && current[:len(r.Path)] == r.Path && current[len(r.Path)] == '/')
	}

	if match {
		if r.Component != nil {
			return component.C(r.Component)
		}
	}
	return component.Text("")
}
