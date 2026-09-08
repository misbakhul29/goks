// Package router - Request/response context for GoKS handlers.
package router

import (
	"encoding/json"
	"net/http"
)

// Context wraps http.ResponseWriter and *http.Request with a rich API.
type Context struct {
	w      http.ResponseWriter
	r      *http.Request
	params map[string]string
	status int
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

// Request returns the underlying *http.Request.
func (c *Context) Request() *http.Request {
	return c.r
}

// SetRequest replaces the underlying *http.Request.
// Useful for middleware that needs to inject values into the request context.
func (c *Context) SetRequest(r *http.Request) {
	c.r = r
}

// Response returns the underlying http.ResponseWriter.
func (c *Context) Response() http.ResponseWriter {
	return c.w
}

// JSON writes a JSON response.
func (c *Context) JSON(data any) error {
	c.w.Header().Set("Content-Type", "application/json")
	c.w.WriteHeader(c.status)
	return json.NewEncoder(c.w).Encode(data)
}

// Text writes a plain-text response.
func (c *Context) Text(text string) error {
	c.w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	c.w.WriteHeader(c.status)
	_, err := c.w.Write([]byte(text))
	return err
}

// HTML writes an HTML response.
func (c *Context) HTML(html string) error {
	c.w.Header().Set("Content-Type", "text/html; charset=utf-8")
	c.w.WriteHeader(c.status)
	_, err := c.w.Write([]byte(html))
	return err
}

// Redirect performs an HTTP redirect.
func (c *Context) Redirect(url string, code ...int) error {
	statusCode := http.StatusFound
	if len(code) > 0 {
		statusCode = code[0]
	}
	http.Redirect(c.w, c.r, url, statusCode)
	return nil
}

// Bind decodes a JSON request body into v.
func (c *Context) Bind(v any) error {
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
