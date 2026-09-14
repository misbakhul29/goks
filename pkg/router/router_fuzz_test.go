package router_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/misbakhul29/goks/pkg/router"
)

// FuzzRouterMatch verifies that the router can handle arbitrary URLs and methods
// without panics or memory corruption.
func FuzzRouterMatch(f *testing.F) {
	seeds := []struct {
		method string
		path   string
	}{
		{"GET", "/"},
		{"GET", "/users/123"},
		{"POST", "/api/v1/posts"},
		{"DELETE", "/files/a/b/c/d"},
		{"GET", "/path%20with%20spaces/and%2Fslash"},
		{"PUT", "/../../evil"},
		{"GET", "///////multiple-slashes"},
		{"GET", "/users/%E0%A4%A"},
		{"CUSTOM", "/arbitrary/custom/method"},
	}

	for _, s := range seeds {
		f.Add(s.method, s.path)
	}

	r := router.New()
	r.GET("/", func(c *router.Context) error { return c.Status(http.StatusOK).Text("root") })
	r.GET("/users/:id", func(c *router.Context) error { return c.Status(http.StatusOK).Text(c.Param("id")) })
	r.POST("/api/v1/posts", func(c *router.Context) error { return c.Status(http.StatusCreated).Text("created") })
	r.GET("/files/*filepath", func(c *router.Context) error { return c.Status(http.StatusOK).Text(c.Param("filepath")) })

	f.Fuzz(func(t *testing.T, method string, path string) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("Router panicked on %s %s: %v", method, path, r)
			}
		}()

		if method == "" {
			method = "GET"
		}
		if len(path) == 0 || path[0] != '/' {
			path = "/" + path
		}

		req, err := http.NewRequest(method, path, nil)
		if err != nil {
			return
		}
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
	})
}
