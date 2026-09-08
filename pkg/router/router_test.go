package router_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/misbakhul29/goks/pkg/router"
)

func TestRouter_StaticRoute(t *testing.T) {
	r := router.New()
	r.GET("/hello", func(ctx *router.Context) error {
		return ctx.Text("hello world")
	})

	req := httptest.NewRequest(http.MethodGet, "/hello", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if w.Body.String() != "hello world" {
		t.Fatalf("expected 'hello world', got %q", w.Body.String())
	}
}

func TestRouter_DynamicParam(t *testing.T) {
	r := router.New()
	r.GET("/blog/:slug", func(ctx *router.Context) error {
		return ctx.Text(ctx.Param("slug"))
	})

	req := httptest.NewRequest(http.MethodGet, "/blog/hello-world", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if w.Body.String() != "hello-world" {
		t.Fatalf("expected 'hello-world', got %q", w.Body.String())
	}
}

func TestRouter_JSONResponse(t *testing.T) {
	r := router.New()
	r.GET("/api/users", func(ctx *router.Context) error {
		return ctx.JSON(map[string]string{"name": "Alice"})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Fatalf("expected JSON content-type, got %q", ct)
	}

	var result map[string]string
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("decode JSON: %v", err)
	}
	if result["name"] != "Alice" {
		t.Fatalf("expected 'Alice', got %q", result["name"])
	}
}

func TestRouter_NotFound(t *testing.T) {
	r := router.New()
	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestRouter_Middleware(t *testing.T) {
	r := router.New()

	var order []string
	mw1 := func(next router.Handler) router.Handler {
		return func(ctx *router.Context) error {
			order = append(order, "mw1-before")
			err := next(ctx)
			order = append(order, "mw1-after")
			return err
		}
	}
	mw2 := func(next router.Handler) router.Handler {
		return func(ctx *router.Context) error {
			order = append(order, "mw2-before")
			err := next(ctx)
			order = append(order, "mw2-after")
			return err
		}
	}

	r.Use(mw1, mw2)
	r.GET("/test", func(ctx *router.Context) error {
		order = append(order, "handler")
		return ctx.Text("ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	expected := []string{"mw1-before", "mw2-before", "handler", "mw2-after", "mw1-after"}
	for i, v := range expected {
		if i >= len(order) || order[i] != v {
			t.Fatalf("middleware order mismatch at %d: want %q, got %v", i, v, order)
		}
	}
}

func TestRouter_POST(t *testing.T) {
	r := router.New()
	r.POST("/api/create", func(ctx *router.Context) error {
		var body map[string]string
		if err := ctx.Bind(&body); err != nil {
			return ctx.Status(http.StatusBadRequest).Text("bad request")
		}
		return ctx.Status(http.StatusCreated).JSON(body)
	})

	payload := `{"title":"Hello"}`
	req := httptest.NewRequest(http.MethodPost, "/api/create", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}
}

func TestRouter_Group(t *testing.T) {
	r := router.New()
	api := r.Group("/api/v1")
	api.GET("/ping", func(ctx *router.Context) error {
		return ctx.Text("pong")
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/ping", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if w.Body.String() != "pong" {
		t.Fatalf("expected 'pong', got %q", w.Body.String())
	}
}
