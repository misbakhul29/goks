package router

import (
	"path/filepath"
	"strings"

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

// HasFileExtension returns true if path appears to be a static file asset (e.g. /Resume.pdf, /image.png).
func HasFileExtension(path string) bool {
	clean := strings.Split(strings.Split(path, "?")[0], "#")[0]
	ext := filepath.Ext(clean)
	return len(ext) > 1
}

// MatchPath checks if current URL path matches a route pattern.
func MatchPath(pattern, current string, exact bool) bool {
	if exact {
		if pattern == current {
			return true
		}
		patParts := strings.Split(strings.Trim(pattern, "/"), "/")
		curParts := strings.Split(strings.Trim(current, "/"), "/")
		if len(patParts) != len(curParts) {
			return false
		}
		for i := range patParts {
			if strings.HasPrefix(patParts[i], ":") {
				continue
			}
			if patParts[i] != curParts[i] {
				return false
			}
		}
		return true
	}
	// Prefix match for nested layouts
	return current == pattern || (len(current) > len(pattern) && current[:len(pattern)] == pattern && current[len(pattern)] == '/')
}

// MatchAnyRoute returns true if current path matches any of the registered route patterns.
func MatchAnyRoute(patterns []string, current string) bool {
	for _, p := range patterns {
		if MatchPath(p, current, true) {
			return true
		}
	}
	return false
}

// DefaultNotFound is the fallback component rendered when a route is not found and app/not-found.gox is not defined.
type DefaultNotFound struct {
	component.ComponentBase
}

func (n *DefaultNotFound) Render() *component.Node {
	return component.H("div", component.Props{
		"style": "min-height: 60vh; display: flex; flex-direction: column; align-items: center; justify-content: center; text-align: center; font-family: system-ui, -apple-system, sans-serif; padding: 2rem;",
	},
		component.H("div", component.Props{
			"style": "display: flex; align-items: center; gap: 1rem; margin-bottom: 1.5rem;",
		},
			component.H("h1", component.Props{
				"style": "font-size: 2.25rem; font-weight: 800; margin: 0; padding-right: 1.25rem; border-right: 1px solid #9ca3af; color: #1f2937;",
			}, component.Text("404")),
			component.H("p", component.Props{
				"style": "font-size: 1rem; color: #4b5563; margin: 0;",
			}, component.Text("This page could not be found.")),
		),
		component.H("a", component.Props{
			"href":  "/",
			"style": "display: inline-block; padding: 0.5rem 1.25rem; background-color: #2563eb; color: #ffffff; border-radius: 0.375rem; text-decoration: none; font-size: 0.875rem; font-weight: 600; box-shadow: 0 1px 3px rgba(0,0,0,0.1);",
		}, component.Text("Return Home")),
	)
}
