// Package router - Request/response context for GoKS handlers.
package router

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Context wraps http.ResponseWriter and *http.Request with a rich API.
type Context struct {
	w       http.ResponseWriter
	r       *http.Request
	params  map[string]string
	store   map[string]any
	status  int
	written bool
}

func newContext(w http.ResponseWriter, r *http.Request, params map[string]string) *Context {
	return &Context{
		w:      w,
		r:      r,
		params: params,
		status: http.StatusOK,
	}
}

// NewTestContext creates a Context instance for testing handlers and middleware.
func NewTestContext(w http.ResponseWriter, r *http.Request) *Context {
	return &Context{
		w:      w,
		r:      r,
		params: make(map[string]string),
		status: http.StatusOK,
	}
}

// Param returns a URL path parameter by name.
// For route /blog/:slug matched against /blog/hello, Param("slug") = "hello"
func (c *Context) Param(key string) string {
	return c.params[key]
}

// Query returns a URL query parameter by name.
func (c *Context) Query(key string) string {
	return c.r.URL.Query().Get(key)
}

// FormValue returns the first value for the named component of the query or POST/PUT body.
func (c *Context) FormValue(key string) string {
	return c.r.FormValue(key)
}

// Header returns a request header value.
func (c *Context) Header(key string) string {
	return c.r.Header.Get(key)
}

// SetHeader sets a response header.
func (c *Context) SetHeader(key, value string) *Context {
	c.w.Header().Set(key, value)
	return c
}

// Set saves arbitrary data in the request context store.
func (c *Context) Set(key string, val any) {
	if c.store == nil {
		c.store = make(map[string]any)
	}
	c.store[key] = val
}

// Get retrieves arbitrary data from the request context store.
func (c *Context) Get(key string) any {
	if c.store == nil {
		return nil
	}
	return c.store[key]
}

// Status sets the HTTP response status code (chainable).
func (c *Context) Status(code int) *Context {
	c.status = code
	return c
}

// WriteHeader sends an HTTP response header with the provided status code.
func (c *Context) WriteHeader(code int) {
	if !c.written {
		c.status = code
		c.w.WriteHeader(code)
		c.written = true
	}
}

// IsWritten reports whether response headers have already been sent.
func (c *Context) IsWritten() bool {
	return c.written
}

// Request returns the underlying *http.Request.
func (c *Context) Request() *http.Request {
	return c.r
}

// SetRequest replaces the underlying *http.Request.
// Useful for middleware that needs to inject values into the request context.
func (c *Context) SetRequest(r *http.Request) {
	c.r = r
}

// RequestID returns the request ID from the header or context.
func (c *Context) RequestID() string {
	if id := c.Header("X-Request-Id"); id != "" {
		return id
	}
	if c.r != nil {
		return RequestIDFromContext(c.r.Context())
	}
	return ""
}

// Response returns the underlying http.ResponseWriter.
func (c *Context) Response() http.ResponseWriter {
	return c.w
}

// JSON writes a JSON response.
func (c *Context) JSON(data any) error {
	c.w.Header().Set("Content-Type", "application/json")
	if !c.written {
		c.w.WriteHeader(c.status)
		c.written = true
	}
	return json.NewEncoder(c.w).Encode(data)
}

// Text writes a plain-text response.
func (c *Context) Text(text string) error {
	c.w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	if !c.written {
		c.w.WriteHeader(c.status)
		c.written = true
	}
	_, err := c.w.Write([]byte(text))
	return err
}

// HTML writes an HTML response.
func (c *Context) HTML(html string) error {
	c.w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if !c.written {
		c.w.WriteHeader(c.status)
		c.written = true
	}
	_, err := c.w.Write([]byte(html))
	return err
}

// StreamHTML sets Content-Type to text/html with chunked transfer encoding,
// and passes the io.Writer and http.Flusher to streamFunc.
func (c *Context) StreamHTML(streamFunc func(w io.Writer, flusher http.Flusher) error) error {
	c.w.Header().Set("Content-Type", "text/html; charset=utf-8")
	c.w.Header().Set("Transfer-Encoding", "chunked")
	c.w.Header().Set("X-Content-Type-Options", "nosniff")
	if !c.written {
		c.w.WriteHeader(c.status)
		c.written = true
	}
	var flusher http.Flusher
	if f, ok := c.w.(http.Flusher); ok {
		flusher = f
	}
	return streamFunc(c.w, flusher)
}

// Redirect performs an HTTP redirect.
func (c *Context) Redirect(url string, code ...int) error {
	statusCode := http.StatusFound
	if len(code) > 0 {
		statusCode = code[0]
	}
	c.written = true
	http.Redirect(c.w, c.r, url, statusCode)
	return nil
}

// Flash stores a one-time flash message in a cookie for the given key.
func (c *Context) Flash(key, val string) {
	cookieName := "goks_flash_" + sanitizeKey(key)
	encoded := base64.RawURLEncoding.EncodeToString([]byte(val))
	secure := c.r != nil && (c.r.TLS != nil || c.r.Header.Get("X-Forwarded-Proto") == "https")

	http.SetCookie(c.w, &http.Cookie{
		Name:     cookieName,
		Value:    encoded,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

// GetFlash retrieves and clears the one-time flash message for the given key.
func (c *Context) GetFlash(key string) string {
	if c.r == nil {
		return ""
	}
	cookieName := "goks_flash_" + sanitizeKey(key)
	cookie, err := c.r.Cookie(cookieName)
	if err != nil || cookie.Value == "" {
		return ""
	}

	http.SetCookie(c.w, &http.Cookie{
		Name:     cookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
	})

	decoded, err := base64.RawURLEncoding.DecodeString(cookie.Value)
	if err != nil {
		return ""
	}
	return string(decoded)
}

// RedirectIntended redirects to the saved intended URL from cookie,
// or to fallbackURL if none or invalid.
func (c *Context) RedirectIntended(fallbackURL string, code ...int) error {
	if fallbackURL == "" {
		fallbackURL = "/"
	}
	target := fallbackURL
	if c.r != nil {
		if cookie, err := c.r.Cookie("goks_intended_url"); err == nil && cookie.Value != "" {
			http.SetCookie(c.w, &http.Cookie{
				Name:     "goks_intended_url",
				Value:    "",
				Path:     "/",
				Expires:  time.Unix(0, 0),
				MaxAge:   -1,
				HttpOnly: true,
			})
			if raw, err := base64.RawURLEncoding.DecodeString(cookie.Value); err == nil {
				candidate := string(raw)
				if isSafeInternalPath(candidate) {
					target = candidate
				}
			}
		}
	}
	return c.Redirect(target, code...)
}

var (
	resolverMu       sync.RWMutex
	authUserResolver func(r *http.Request) any
)

// RegisterAuthUserResolver registers a resolver function for AuthUser().
func RegisterAuthUserResolver(fn func(r *http.Request) any) {
	resolverMu.Lock()
	defer resolverMu.Unlock()
	authUserResolver = fn
}

// AuthUser returns the authenticated user (if any) via the registered resolver.
func (c *Context) AuthUser() any {
	resolverMu.RLock()
	fn := authUserResolver
	resolverMu.RUnlock()

	if fn == nil || c.r == nil {
		return nil
	}
	return fn(c.r)
}

// Bind decodes a JSON request body into v.
func (c *Context) Bind(v any) error {
	if c.r.Body == nil {
		return io.EOF
	}
	defer c.r.Body.Close()
	return json.NewDecoder(c.r.Body).Decode(v)
}

// APIError represents the standard error response payload for API routes.
type APIError struct {
	Code      int    `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id,omitempty"`
}

// ErrorResponse wraps APIError in an standard "error" envelope.
type ErrorResponse struct {
	Error APIError `json:"error"`
}

// Error writes a standardized JSON error response with status code, message, and request ID.
func (c *Context) Error(status int, message string) error {
	c.Status(status)
	return c.JSON(ErrorResponse{
		Error: APIError{
			Code:      status,
			Message:   message,
			RequestID: c.RequestID(),
		},
	})
}

// Method returns the HTTP method of the request.
func (c *Context) Method() string {
	return c.r.Method
}

// Path returns the URL path of the request.
func (c *Context) Path() string {
	return c.r.URL.Path
}

func sanitizeKey(s string) string {
	return strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			return r
		}
		return '_'
	}, s)
}

func isSafeInternalPath(target string) bool {
	if target == "" || !strings.HasPrefix(target, "/") || strings.HasPrefix(target, "//") || strings.Contains(target, `\`) {
		return false
	}
	u, err := url.Parse(target)
	if err != nil {
		return false
	}
	return u.Host == "" && u.Scheme == ""
}
