// Package router provides file-based HTTP routing for GoKS.
// It scans a directory of Go page handlers and maps them to URL patterns.
package router

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

// Handler is the function signature for GoKS page/API handlers.
type Handler func(ctx *Context) error

// MiddlewareFunc is a function that wraps a Handler.
type MiddlewareFunc func(Handler) Handler

// Router is the GoKS HTTP router.
type Router struct {
	mu               sync.RWMutex
	routes           []*Route
	middlewares      []MiddlewareFunc
	notFound         Handler
	methodNotAllowed Handler
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
		methodNotAllowed: func(ctx *Context) error {
			ctx.Status(http.StatusMethodNotAllowed).Text("405 Method Not Allowed")
			return nil
		},
	}
}

// Use adds global middleware to the router.
func (r *Router) Use(mw ...MiddlewareFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.middlewares = append(r.middlewares, mw...)
}

// Handle registers a route with a specific HTTP method.
func (r *Router) Handle(method, pattern string, h Handler) {
	method = strings.ToUpper(method)
	pattern = normalizePath(pattern)
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, existing := range r.routes {
		if existing.method == method && existing.pattern == pattern {
			existing.handler = h
			return
		}
	}
	r.routes = append(r.routes, &Route{
		method:  method,
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

// RouteInfo represents inspectable information about a registered route.
type RouteInfo struct {
	Method  string `json:"method"`
	Pattern string `json:"pattern"`
}

// Routes returns a copy of all registered routes for inspection (e.g. by DevTools / Studio).
func (r *Router) Routes() []RouteInfo {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	res := make([]RouteInfo, 0, len(r.routes))
	for _, rt := range r.routes {
		res = append(res, RouteInfo{
			Method:  rt.method,
			Pattern: rt.pattern,
		})
	}
	return res
}

// NotFound registers a custom 404 handler.
func (r *Router) NotFound(h Handler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.notFound = h
}

// MethodNotAllowed registers a custom 405 handler.
func (r *Router) MethodNotAllowed(h Handler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.methodNotAllowed = h
}

// calculateRouteScore determines precedence: static (100) > dynamic (10) > wildcard (1)
func calculateRouteScore(parts []string) int {
	score := 0
	for _, p := range parts {
		if strings.HasPrefix(p, "*") {
			score += 1
		} else if strings.HasPrefix(p, ":") {
			score += 10
		} else if p != "" {
			score += 100
		}
	}
	return score
}

// ServeHTTP implements http.Handler.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	path := normalizePath(req.URL.Path)

	r.mu.RLock()
	routes := make([]*Route, len(r.routes))
	copy(routes, r.routes)
	middlewares := make([]MiddlewareFunc, len(r.middlewares))
	copy(middlewares, r.middlewares)
	notFound := r.notFound
	methodNotAllowed := r.methodNotAllowed
	r.mu.RUnlock()

	var matched *Route
	var matchedParams map[string]string
	bestScore := -1
	var allowedMethods []string

	for _, route := range routes {
		if ok, p := matchRoute(route.parts, path); ok {
			if route.method == req.Method || (req.Method == "HEAD" && route.method == "GET") {
				score := calculateRouteScore(route.parts)
				if score > bestScore {
					bestScore = score
					matched = route
					matchedParams = p
				}
			} else {
				allowedMethods = append(allowedMethods, route.method)
			}
		}
	}

	ctx := newContext(w, req, matchedParams)

	var h Handler
	if matched != nil {
		h = matched.handler
	} else if len(allowedMethods) > 0 {
		w.Header().Set("Allow", strings.Join(allowedMethods, ", "))
		h = methodNotAllowed
	} else {
		h = notFound
	}

	// Wrap with middlewares (outermost first)
	for i := len(middlewares) - 1; i >= 0; i-- {
		h = middlewares[i](h)
	}

	if err := h(ctx); err != nil && !ctx.IsWritten() {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// matchRoute checks if a URL path matches a route pattern, extracting params with URL unescaping.
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
				raw := strings.Join(pathParts[i:], "/")
				if unescaped, err := url.PathUnescape(raw); err == nil {
					params[pp[1:]] = unescaped
				} else {
					params[pp[1:]] = raw
				}
			}
			return true, params
		} else if strings.HasPrefix(pp, ":") {
			// Dynamic segment :slug
			raw := pathParts[i]
			if unescaped, err := url.PathUnescape(raw); err == nil {
				params[pp[1:]] = unescaped
			} else {
				params[pp[1:]] = raw
			}
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
	for strings.Contains(p, "//") {
		p = strings.ReplaceAll(p, "//", "/")
	}
	if len(p) > 1 && strings.HasSuffix(p, "/") {
		p = strings.TrimSuffix(p, "/")
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
	full := normalizePath(g.prefix + "/" + strings.TrimPrefix(pattern, "/"))
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

// Group creates a nested route group with a combined prefix and chained middlewares.
func (g *Group) Group(prefix string, mw ...MiddlewareFunc) *Group {
	fullPrefix := normalizePath(g.prefix + "/" + strings.TrimPrefix(prefix, "/"))
	combinedMW := make([]MiddlewareFunc, 0, len(g.middlewares)+len(mw))
	combinedMW = append(combinedMW, g.middlewares...)
	combinedMW = append(combinedMW, mw...)
	return &Group{
		router:      g.router,
		prefix:      fullPrefix,
		middlewares: combinedMW,
	}
}

// Use adds middleware to the route group.
func (g *Group) Use(mw ...MiddlewareFunc) {
	g.middlewares = append(g.middlewares, mw...)
}

// -----------------------------------------------------------------------
// Context key type
// -----------------------------------------------------------------------

type contextKey string

const paramsKey contextKey = "params"

// WithParams stores URL params in a context.
func WithParams(ctx context.Context, params map[string]string) context.Context {
	return context.WithValue(ctx, paramsKey, params)
}
