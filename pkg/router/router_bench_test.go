package router_test

import (
	"net/http/httptest"
	"testing"

	"github.com/misbakhul29/goks/pkg/router"
)

func BenchmarkRouter_StaticMatch(b *testing.B) {
	r := router.New()
	r.GET("/api/v1/users/profile", func(c *router.Context) error {
		return c.Text("profile")
	})
	r.GET("/about", func(c *router.Context) error {
		return c.Text("about")
	})

	req := httptest.NewRequest("GET", "/api/v1/users/profile", nil)
	w := httptest.NewRecorder()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		r.ServeHTTP(w, req)
	}
}

func BenchmarkRouter_DynamicParamMatch(b *testing.B) {
	r := router.New()
	r.GET("/users/:id/posts/:post_id", func(c *router.Context) error {
		_ = c.Param("id")
		_ = c.Param("post_id")
		return c.Text("post")
	})

	req := httptest.NewRequest("GET", "/users/42/posts/100", nil)
	w := httptest.NewRecorder()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		r.ServeHTTP(w, req)
	}
}
