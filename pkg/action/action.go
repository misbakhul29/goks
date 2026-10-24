// Package action provides Progressive Server Actions for GoKS.
// Server Actions allow executing Go backend logic directly from forms
// with zero JavaScript fallback (standard HTTP POST + redirect) and
// progressive enhancement via WebAssembly without full page reloads.
package action

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"sync"

	"github.com/misbakhul29/goks/pkg/auth"
	"github.com/misbakhul29/goks/pkg/component"
	"github.com/misbakhul29/goks/pkg/rbac"
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
// It inspects parsed PostForm, multipart form values, and request FormValue.
func (c *Context) Get(key string) string {
	if c.Form != nil {
		if val := c.Form.Get(key); val != "" {
			return val
		}
	}
	if c.Request != nil {
		if c.Request.MultipartForm != nil && c.Request.MultipartForm.Value != nil {
			if vals, ok := c.Request.MultipartForm.Value[key]; ok && len(vals) > 0 {
				return vals[0]
			}
		}
		return c.Request.FormValue(key)
	}
	return ""
}

// GetAll returns all values associated with the given key from submitted form data.
func (c *Context) GetAll(key string) []string {
	var results []string
	if c.Form != nil {
		if vals, ok := c.Form[key]; ok {
			results = append(results, vals...)
		}
	}
	if c.Request != nil && c.Request.MultipartForm != nil && c.Request.MultipartForm.Value != nil {
		if vals, ok := c.Request.MultipartForm.Value[key]; ok {
			for _, v := range vals {
				already := false
				for _, r := range results {
					if r == v {
						already = true
						break
					}
				}
				if !already {
					results = append(results, v)
				}
			}
		}
	}
	return results
}

// File retrieves the uploaded file for the given key from multipart form data.
func (c *Context) File(key string) (multipart.File, *multipart.FileHeader, error) {
	if c.Request == nil {
		return nil, nil, http.ErrMissingFile
	}
	return c.Request.FormFile(key)
}

// User returns the authenticated user from the request, or nil if unauthenticated.
func (c *Context) User() *auth.User {
	if c.Request == nil {
		return nil
	}
	return auth.CurrentUser(c.Request)
}

// IsAuthenticated reports whether the current request has an active authenticated session.
func (c *Context) IsAuthenticated() bool {
	return c.User() != nil
}

// Login creates a new session for the given user and sets the auth cookie.
func (c *Context) Login(user *auth.User, opts ...auth.LoginOption) (*auth.Session, error) {
	if c.Response == nil || c.Request == nil {
		return nil, http.ErrNotSupported
	}
	sess, err := auth.Default.Login(c.Response, c.Request, user, opts...)
	if err != nil {
		return nil, err
	}
	c.Request = c.Request.WithContext(auth.ContextWithSession(c.Request.Context(), sess))
	return sess, nil
}

// Logout clears the current session and invalidates the session cookie.
func (c *Context) Logout() {
	if c.Response != nil && c.Request != nil {
		auth.Default.Logout(c.Response, c.Request)
		c.Request = c.Request.WithContext(auth.ContextWithSession(c.Request.Context(), nil))
	}
}

// Flash stores a one-time flash message in a cookie for the given key.
func (c *Context) Flash(key, val string) {
	if c.Response != nil && c.Request != nil {
		auth.SetFlash(c.Response, c.Request, key, val)
	}
}

// GetFlash retrieves and clears the one-time flash message for the given key.
func (c *Context) GetFlash(key string) string {
	if c.Response != nil && c.Request != nil {
		return auth.GetFlash(c.Response, c.Request, key)
	}
	return ""
}

// RedirectIntended redirects to the previously intended URL if present, or to fallbackURL.
func (c *Context) RedirectIntended(fallbackURL string, code ...int) (any, error) {
	statusCode := http.StatusFound
	if len(code) > 0 {
		statusCode = code[0]
	}
	target := auth.GetIntendedURL(c.Response, c.Request, fallbackURL)
	if c.Response != nil && c.Request != nil {
		http.Redirect(c.Response, c.Request, target, statusCode)
	}
	return map[string]string{"redirect": target}, nil
}

// HasRole checks if the authenticated user has any of the specified roles.
func (c *Context) HasRole(roles ...string) bool {
	u := c.User()
	if u == nil {
		return false
	}
	for _, required := range roles {
		for _, r := range u.Roles {
			if r == required {
				return true
			}
		}
	}
	return false
}

// Can checks if the authenticated user has a specific permission via rbac.Default.
func (c *Context) Can(perm string) bool {
	u := c.User()
	if u == nil {
		return false
	}
	return rbac.Default.UserHas(u, rbac.Permission(perm))
}

var (
	registryMu sync.RWMutex
	actions    = make(map[string]Action)

	secretMu   sync.RWMutex
	csrfSecret []byte
)

// SetSecret sets the HMAC secret used to generate and validate CSRF tokens.
// Secret must be at least 32 bytes long (panics if non-nil and shorter than 32 bytes).
func SetSecret(secret []byte) {
	if secret != nil && len(secret) < 32 {
		panic("goks/action: CSRF secret must be at least 32 bytes long")
	}
	secretMu.Lock()
	defer secretMu.Unlock()
	if secret == nil {
		csrfSecret = nil
		return
	}
	csrfSecret = make([]byte, len(secret))
	copy(csrfSecret, secret)
}

// GenerateToken generates a cryptographically signed HMAC-SHA256 token for an action and session key.
func GenerateToken(actionName, sessionKey string) string {
	secretMu.RLock()
	sec := csrfSecret
	secretMu.RUnlock()

	if len(sec) == 0 {
		return ""
	}

	mac := hmac.New(sha256.New, sec)
	mac.Write([]byte(actionName + ":" + sessionKey))
	return hex.EncodeToString(mac.Sum(nil))
}

// ValidateToken verifies an HMAC-SHA256 CSRF token for the specified action and session key.
func ValidateToken(actionName, sessionKey, token string) bool {
	secretMu.RLock()
	sec := csrfSecret
	secretMu.RUnlock()

	if len(sec) == 0 {
		return !isProduction()
	}

	if token == "" {
		return false
	}

	expected := GenerateToken(actionName, sessionKey)
	return hmac.Equal([]byte(expected), []byte(token))
}

func isProduction() bool {
	env := strings.ToLower(os.Getenv("GOKS_ENV"))
	if env == "" {
		env = strings.ToLower(os.Getenv("ENV"))
	}
	if env == "" {
		env = strings.ToLower(os.Getenv("APP_ENV"))
	}
	return env == "production" || env == "prod"
}

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

// Form renders a progressive HTML <form> node configured to invoke the named Server Action.
// It automatically injects the action endpoint, method="POST", hidden input "_action", and CSRF token (if configured).
func Form(actionName string, props component.Props, children ...*component.Node) *component.Node {
	if props == nil {
		props = component.Props{}
	}
	props["action"] = URL(actionName)
	props["method"] = "POST"

	hiddenNodes := []*component.Node{
		component.H("input", component.Props{
			"type":  "hidden",
			"name":  "_action",
			"value": actionName,
		}),
	}

	secretMu.RLock()
	hasSecret := len(csrfSecret) > 0
	secretMu.RUnlock()

	if hasSecret {
		csrfToken, ok := props["csrf"].(string)
		if !ok || csrfToken == "" {
			sessionKey, _ := props["sessionKey"].(string)
			csrfToken = GenerateToken(actionName, sessionKey)
		}
		delete(props, "csrf")
		delete(props, "sessionKey")

		if csrfToken != "" {
			hiddenNodes = append(hiddenNodes, component.H("input", component.Props{
				"type":  "hidden",
				"name":  "_csrf",
				"value": csrfToken,
			}))
		}
	}

	allChildren := append(hiddenNodes, children...)
	return component.H("form", props, allChildren...)
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
				log.Printf("[GoKS/action] PANIC in action handler: %v", rec)
				http.Error(w, "internal server error", http.StatusInternalServerError)
			}
		}()

		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// CSRF protection: validate Sec-Fetch-Site, Origin, and Referer.
		secSite := r.Header.Get("Sec-Fetch-Site")
		if secSite == "cross-site" {
			http.Error(w, "cross-origin actions not allowed", http.StatusForbidden)
			return
		}

		origin := r.Header.Get("Origin")
		referer := r.Header.Get("Referer")

		if origin != "" {
			if !sameOrigin(r, origin) {
				http.Error(w, "cross-origin actions not allowed", http.StatusForbidden)
				return
			}
		} else if referer != "" {
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

		// Validate CSRF token if secret is configured or in production
		secretMu.RLock()
		hasSecret := len(csrfSecret) > 0
		secretMu.RUnlock()

		if isProduction() && !hasSecret {
			http.Error(w, "CSRF protection secret required in production", http.StatusForbidden)
			return
		}

		if hasSecret || isProduction() {
			token := r.Header.Get("X-CSRF-Token")
			if token == "" {
				token = r.FormValue("_csrf")
			}

			// Prioritize authenticated session cookie to prevent cross-session token reuse
			sessionKey := ""
			if sessCookie, err := r.Cookie("goks_session"); err == nil && sessCookie.Value != "" {
				sessionKey = sessCookie.Value
			} else {
				sessionKey = r.Header.Get("X-Session-Key")
				if sessionKey == "" {
					sessionKey = r.FormValue("_session_key")
				}
			}

			// In production, require CSRF token especially if Origin/Referer headers are omitted
			if isProduction() && token == "" && origin == "" && referer == "" {
				http.Error(w, "invalid or missing CSRF token", http.StatusForbidden)
				return
			}

			if !ValidateToken(name, sessionKey, token) {
				http.Error(w, "invalid or missing CSRF token", http.StatusForbidden)
				return
			}
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

		http.Redirect(w, r, safeRedirectTarget(r, targetURL), http.StatusSeeOther)
	}
}

func safeRedirectTarget(r *http.Request, rawTarget string) string {
	if rawTarget == "" || strings.ContainsAny(rawTarget, "\r\n\\") {
		return "/"
	}

	target, err := url.Parse(rawTarget)
	if err != nil || target.User != nil {
		return "/"
	}

	// If absolute URL, verify same origin before redirecting
	if target.IsAbs() || target.Host != "" {
		if sameOriginReferer(r, rawTarget) {
			uri := target.RequestURI()
			if strings.HasPrefix(uri, "/") && !strings.HasPrefix(uri, "//") {
				return uri
			}
		}
		return "/"
	}

	if !strings.HasPrefix(target.Path, "/") || strings.HasPrefix(target.Path, "//") {
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
	if err != nil {
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
