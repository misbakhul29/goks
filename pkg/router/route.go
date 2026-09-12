package router

import (
	"github.com/misbakhul29/goks/pkg/component"
)

// CurrentPath is a global store holding the current URL path.
var CurrentPath = component.NewStore("/")

// Push navigates to a new path programmatically (overridden in WASM).
var Push = func(path string) {
	CurrentPath.Set(path)
}

// Replace navigates to a new path replacing the current history entry (overridden in WASM).
var Replace = func(path string) {
	CurrentPath.Set(path)
}

// Back navigates back to the previous page in history (overridden in WASM).
var Back = func() {}

// Forward navigates forward to the next page in history (overridden in WASM).
var Forward = func() {}

// Path returns the current URL pathname.
func Path() string {
	return CurrentPath.Get()
}

// ClientRouter provides Next.js-like useRouter methods for programmatic navigation.
type ClientRouter struct{}

// UseRouter returns a ClientRouter instance for navigation.
//
// Example:
//
//	r := router.UseRouter()
//	r.Push("/login")
func UseRouter() *ClientRouter {
	return &ClientRouter{}
}

// Push navigates to the specified URL path and pushes to browser history.
func (r *ClientRouter) Push(path string) {
	Push(path)
}

// Replace navigates to the specified URL path replacing the current history entry.
func (r *ClientRouter) Replace(path string) {
	Replace(path)
}

// Back navigates back to the previous page in history.
func (r *ClientRouter) Back() {
	Back()
}

// Forward navigates forward to the next page in history.
func (r *ClientRouter) Forward() {
	Forward()
}

// Path returns the current URL pathname.
func (r *ClientRouter) Path() string {
	return CurrentPath.Get()
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
