// Package router - Request/response context for GoKS handlers.
package router

import (
	"encoding/json"
	"io"
	"net/http"
)

// Context wraps http.ResponseWriter and *http.Request with a rich API.
type Context struct {
	w       http.ResponseWriter
	r       *http.Request
	params  map[string]string
	status  int
	written bool
}

func newContext(w http.ResponseWriter, r *http.Request, params map[string]string) *Context {
	return &Context{w: w, r: r, params: params, status: http.StatusOK}
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

// Header returns a request header value.
func (c *Context) Header(key string) string {
	return c.r.Header.Get(key)
}

// SetHeader sets a response header.
func (c *Context) SetHeader(key, value string) *Context {
	c.w.Header().Set(key, value)
	return c
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

// Bind decodes a JSON request body into v.
func (c *Context) Bind(v any) error {
	if c.r.Body == nil {
		return io.EOF
	}
	defer c.r.Body.Close()
	return json.NewDecoder(c.r.Body).Decode(v)
}

// Method returns the HTTP method of the request.
func (c *Context) Method() string {
	return c.r.Method
}

// Path returns the URL path of the request.
func (c *Context) Path() string {
	return c.r.URL.Path
}
