// Package router provides file-based HTTP routing for GoKS.
// It scans a directory of Go page handlers and maps them to URL patterns.
package router

import (
	"context"
	"net/http"
	"strings"
)

// Handler is the function signature for GoKS page/API handlers.
type Handler func(ctx *Context) error

// MiddlewareFunc is a function that wraps a Handler.
type MiddlewareFunc func(Handler) Handler

// Router is the GoKS HTTP router.
type Router struct {
	routes      []*Route
	middlewares []MiddlewareFunc
	notFound    Handler
}

// Route represents a registered route.
type Route struct {
	method  string
	pattern string   // e.g. "/blog/:slug"
	parts   []string // split pattern segments
	handler Handler
}

// New creates a new Router.
func New() *Router {
	return &Router{
		notFound: func(ctx *Context) error {
			ctx.Status(http.StatusNotFound).Text("404 Not Found")
			return nil
		},
	}
}

// Use adds global middleware to the router.
func (r *Router) Use(mw ...MiddlewareFunc) {
	r.middlewares = append(r.middlewares, mw...)
}

// Handle registers a route with a specific HTTP method.
func (r *Router) Handle(method, pattern string, h Handler) {
	pattern = normalizePath(pattern)
	r.routes = append(r.routes, &Route{
		method:  strings.ToUpper(method),
		pattern: pattern,
		parts:   strings.Split(strings.Trim(pattern, "/"), "/"),
		handler: h,
	})
}

// GET registers a GET route.
func (r *Router) GET(pattern string, h Handler) { r.Handle("GET", pattern, h) }

// POST registers a POST route.
func (r *Router) POST(pattern string, h Handler) { r.Handle("POST", pattern, h) }

// PUT registers a PUT route.
func (r *Router) PUT(pattern string, h Handler) { r.Handle("PUT", pattern, h) }

// DELETE registers a DELETE route.
func (r *Router) DELETE(pattern string, h Handler) { r.Handle("DELETE", pattern, h) }

// PATCH registers a PATCH route.
func (r *Router) PATCH(pattern string, h Handler) { r.Handle("PATCH", pattern, h) }

// NotFound registers a custom 404 handler.
func (r *Router) NotFound(h Handler) { r.notFound = h }

// ServeHTTP implements http.Handler.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	path := normalizePath(req.URL.Path)
	params := make(map[string]string)

	var matched *Route
	for _, route := range r.routes {
		if route.method != req.Method {
			continue
		}
		if ok, p := matchRoute(route.parts, path); ok {
			matched = route
			params = p
			break
		}
	}

	ctx := newContext(w, req, params)

	var h Handler
	if matched != nil {
		h = matched.handler
	} else {
		h = r.notFound
	}

	// Wrap with middlewares (outermost first)
	for i := len(r.middlewares) - 1; i >= 0; i-- {
		h = r.middlewares[i](h)
	}

	if err := h(ctx); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// matchRoute checks if a URL path matches a route pattern, extracting params.
func matchRoute(patternParts []string, path string) (bool, map[string]string) {
	pathParts := strings.Split(strings.Trim(path, "/"), "/")

	// Handle root
	if len(patternParts) == 1 && patternParts[0] == "" && len(pathParts) == 1 && pathParts[0] == "" {
		return true, nil
	}

	hasWildcard := len(patternParts) > 0 && strings.HasPrefix(patternParts[len(patternParts)-1], "*")

	if !hasWildcard && len(patternParts) != len(pathParts) {
		return false, nil
	}
	if hasWildcard && len(pathParts) < len(patternParts)-1 {
		return false, nil
	}

	params := make(map[string]string)
	for i, pp := range patternParts {
		if strings.HasPrefix(pp, "*") {
			if len(pp) > 1 {
				params[pp[1:]] = strings.Join(pathParts[i:], "/")
			}
			return true, params
		} else if strings.HasPrefix(pp, ":") {
			// Dynamic segment :slug
			params[pp[1:]] = pathParts[i]
		} else if pp != pathParts[i] {
			return false, nil
		}
	}
	return true, params
}

func normalizePath(p string) string {
	if p == "" {
		return "/"
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	return p
}

// -----------------------------------------------------------------------
// Group — route grouping with prefix and middleware
// -----------------------------------------------------------------------

// Group is a set of routes sharing a common prefix and middleware.
type Group struct {
	router      *Router
	prefix      string
	middlewares []MiddlewareFunc
}

// Group creates a new route group.
func (r *Router) Group(prefix string, mw ...MiddlewareFunc) *Group {
	return &Group{router: r, prefix: normalizePath(prefix), middlewares: mw}
}

func (g *Group) handle(method, pattern string, h Handler) {
	full := g.prefix + normalizePath(pattern)
	// Wrap handler with group middlewares
	for i := len(g.middlewares) - 1; i >= 0; i-- {
		h = g.middlewares[i](h)
	}
	g.router.Handle(method, full, h)
}

func (g *Group) GET(pattern string, h Handler)    { g.handle("GET", pattern, h) }
func (g *Group) POST(pattern string, h Handler)   { g.handle("POST", pattern, h) }
func (g *Group) PUT(pattern string, h Handler)    { g.handle("PUT", pattern, h) }
func (g *Group) DELETE(pattern string, h Handler) { g.handle("DELETE", pattern, h) }
func (g *Group) PATCH(pattern string, h Handler)  { g.handle("PATCH", pattern, h) }

// -----------------------------------------------------------------------
// Context key type
// -----------------------------------------------------------------------

type contextKey string

const paramsKey contextKey = "params"

// WithParams stores URL params in a context.
func WithParams(ctx context.Context, params map[string]string) context.Context {
	return context.WithValue(ctx, paramsKey, params)
}
