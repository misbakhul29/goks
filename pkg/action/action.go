// Package action provides Progressive Server Actions for GoKS.
// Server Actions allow executing Go backend logic directly from forms
// with zero JavaScript fallback (standard HTTP POST + redirect) and
// progressive enhancement via WebAssembly without full page reloads.
package action

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
)

// Action is a server-side action handler.
type Action func(ctx *Context) (any, error)

// Context encapsulates the request, response, and submitted form parameters.
type Context struct {
	Request  *http.Request
	Response http.ResponseWriter
	Form     url.Values
}

// Get returns the first value associated with the given key from submitted form data.
func (c *Context) Get(key string) string {
	if c.Form == nil {
		return ""
	}
	return c.Form.Get(key)
}

// GetAll returns all values associated with the given key from submitted form data.
func (c *Context) GetAll(key string) []string {
	if c.Form == nil {
		return nil
	}
	return c.Form[key]
}

var (
	registryMu sync.RWMutex
	actions    = make(map[string]Action)
)

// Register registers a named server action that can be invoked via form or WASM client.
func Register(name string, fn Action) {
	registryMu.Lock()
	defer registryMu.Unlock()
	actions[name] = fn
}

// RegisteredActions returns a sorted list of all registered Server Action names.
func RegisteredActions() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()
	list := make([]string, 0, len(actions))
	for a := range actions {
		list = append(list, a)
	}
	sort.Strings(list)
	return list
}

// URL returns the action endpoint URL to be used in HTML <form action="..."> attributes.
//
//	<form action={action.URL("createPost")} method="POST">
func URL(name string) string {
	return "/__goks_action?name=" + url.QueryEscape(name)
}

// Response represents the JSON response sent to AJAX/WASM clients.
type Response struct {
	Success bool   `json:"success"`
	Data    any    `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
}

// Handler returns the HTTP handler that processes server action submissions.
func Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				http.Error(w, fmt.Sprintf("action panic: %v", rec), http.StatusInternalServerError)
			}
		}()

		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// CSRF protection: validate Sec-Fetch-Site, Origin, and Referer.
		if secSite := r.Header.Get("Sec-Fetch-Site"); secSite == "cross-site" {
			http.Error(w, "cross-origin actions not allowed", http.StatusForbidden)
			return
		}

		origin := r.Header.Get("Origin")
		if origin != "" {
			if !sameOrigin(r, origin) {
				http.Error(w, "cross-origin actions not allowed", http.StatusForbidden)
				return
			}
		} else if referer := r.Header.Get("Referer"); referer != "" {
			if !sameOriginReferer(r, referer) {
				http.Error(w, "cross-origin actions not allowed", http.StatusForbidden)
				return
			}
		}

		// Limit payload to 10MB to avoid memory exhaustion
		r.Body = http.MaxBytesReader(w, r.Body, 10<<20)

		name := r.URL.Query().Get("name")
		if name == "" {
			name = r.FormValue("_action")
		}

		registryMu.RLock()
		act, ok := actions[name]
		registryMu.RUnlock()

		if !ok {
			http.Error(w, fmt.Sprintf("action not found: %s", name), http.StatusNotFound)
			return
		}

		// Parse form data
		_ = r.ParseMultipartForm(10 << 20)
		if r.PostForm == nil {
			_ = r.ParseForm()
		}

		ctx := &Context{
			Request:  r,
			Response: w,
			Form:     r.PostForm,
		}

		res, err := act(ctx)

		// Check if client expects JSON (AJAX / WASM client)
		isJSON := strings.Contains(r.Header.Get("Accept"), "application/json") ||
			r.Header.Get("X-GoKS-Action") == "1"

		if isJSON {
			w.Header().Set("Content-Type", "application/json")
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(Response{Success: false, Error: err.Error()})
				return
			}
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(Response{Success: true, Data: res})
			return
		}

		// Standard HTML Form fallback (Progressive Enhancement):
		// Redirect back to Referer or specified redirect target
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		targetURL := r.URL.Query().Get("redirect")
		if targetURL == "" {
			targetURL = r.Header.Get("Referer")
		}
		if targetURL == "" {
			targetURL = "/"
		}

		http.Redirect(w, r, safeRedirectTarget(targetURL), http.StatusSeeOther)
	}
}

func safeRedirectTarget(rawTarget string) string {
	if rawTarget == "" || strings.ContainsAny(rawTarget, "\r\n\\") {
		return "/"
	}

	target, err := url.Parse(rawTarget)
	if err != nil || target.IsAbs() || target.Host != "" || target.User != nil ||
		!strings.HasPrefix(target.Path, "/") || strings.HasPrefix(target.Path, "//") {
		return "/"
	}
	return rawTarget
}

func sameOrigin(r *http.Request, rawOrigin string) bool {
	origin, err := url.Parse(rawOrigin)
	if err != nil || origin.Scheme == "" || origin.Host == "" || origin.User != nil ||
		origin.Path != "" || origin.RawQuery != "" || origin.Fragment != "" {
		return false
	}

	requestScheme := "http"
	if r.TLS != nil {
		requestScheme = "https"
	}
	if r.URL != nil && r.URL.Scheme != "" {
		requestScheme = strings.ToLower(r.URL.Scheme)
	}
	if !strings.EqualFold(origin.Scheme, requestScheme) {
		return false
	}

	requestHost := r.Host
	if requestHost == "" && r.URL != nil {
		requestHost = r.URL.Host
	}
	requestURL, err := url.Parse(requestScheme + "://" + requestHost)
	if err != nil || requestURL.Host == "" {
		return false
	}

	return strings.EqualFold(strings.TrimSuffix(origin.Hostname(), "."), strings.TrimSuffix(requestURL.Hostname(), ".")) &&
		effectivePort(origin) == effectivePort(requestURL)
}

func effectivePort(u *url.URL) string {
	if port := u.Port(); port != "" {
		return port
	}
	if strings.EqualFold(u.Scheme, "https") {
		return "443"
	}
	return "80"
}

func sameOriginReferer(r *http.Request, rawReferer string) bool {
	if strings.HasPrefix(rawReferer, "/") && !strings.HasPrefix(rawReferer, "//") {
		return true // relative referer path is same-origin
	}
	refURL, err := url.Parse(rawReferer)
	if err != nil || refURL.Scheme == "" || refURL.Host == "" {
		return false
	}
	originFromRef := refURL.Scheme + "://" + refURL.Host
	return sameOrigin(r, originFromRef)
}
