package router_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
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

	// Test group with trailing slash in prefix and leading slash in pattern
	api2 := r.Group("/v2/")
	api2.GET("/users", func(ctx *router.Context) error {
		return ctx.Text("users-list")
	})

	req2 := httptest.NewRequest(http.MethodGet, "/v2/users", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK || w2.Body.String() != "users-list" {
		t.Fatalf("expected 200 'users-list', got %d %q", w2.Code, w2.Body.String())
	}
}

func TestUseRouter(t *testing.T) {
	r := router.UseRouter()
	if r == nil {
		t.Fatal("expected non-nil router")
	}

	r.Push("/dashboard")
	if r.Path() != "/dashboard" {
		t.Fatalf("expected path '/dashboard', got %q", r.Path())
	}

	r.Replace("/settings")
	if r.Path() != "/settings" {
		t.Fatalf("expected path '/settings', got %q", r.Path())
	}

	// Direct router package helpers
	router.Push("/profile")
	if router.Path() != "/profile" {
		t.Fatalf("expected path '/profile', got %q", router.Path())
	}

	// Back and forward (safe no-op in non-wasm environment)
	r.Back()
	r.Forward()
}

func TestRouter_FileBasedAPIRoutes(t *testing.T) {
	r := router.New()
	r.Use(router.Recover())

	// Simulate app/api/users/_id/route.go (GET and DELETE)
	r.GET("/api/users/:id", func(ctx *router.Context) error {
		id := ctx.Param("id")
		return ctx.JSON(map[string]string{"id": id, "action": "get"})
	})

	r.DELETE("/api/users/:id", func(ctx *router.Context) error {
		_ = ctx.Param("id")
		return ctx.Status(http.StatusNoContent).Text("")
	})

	// Simulate app/api/files/*slug/route.go (GET wildcard)
	r.GET("/api/files/*slug", func(ctx *router.Context) error {
		slug := ctx.Param("slug")
		return ctx.JSON(map[string]string{"path": slug})
	})

	// 1. Test GET /api/users/42
	req1 := httptest.NewRequest(http.MethodGet, "/api/users/42", nil)
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)
	if w1.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w1.Code)
	}
	var res1 map[string]string
	_ = json.NewDecoder(w1.Body).Decode(&res1)
	if res1["id"] != "42" || res1["action"] != "get" {
		t.Fatalf("unexpected response: %+v", res1)
	}

	// 2. Test DELETE /api/users/42
	req2 := httptest.NewRequest(http.MethodDelete, "/api/users/42", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w2.Code)
	}

	// 3. Test Wildcard GET /api/files/docs/2026/report.pdf
	req3 := httptest.NewRequest(http.MethodGet, "/api/files/docs/2026/report.pdf", nil)
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)
	if w3.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w3.Code)
	}
	var res3 map[string]string
	_ = json.NewDecoder(w3.Body).Decode(&res3)
	if res3["path"] != "docs/2026/report.pdf" {
		t.Fatalf("expected wildcard path 'docs/2026/report.pdf', got %q", res3["path"])
	}
}

func TestRouter_BindAndCustomError(t *testing.T) {
	r := router.New()

	type UserInput struct {
		Username string `json:"username"`
	}

	r.POST("/api/users", func(ctx *router.Context) error {
		var input UserInput
		if err := ctx.Bind(&input); err != nil {
			return ctx.Status(http.StatusBadRequest).JSON(map[string]string{"error": "invalid payload"})
		}
		if input.Username == "" {
			return ctx.Status(http.StatusUnprocessableEntity).JSON(map[string]string{"error": "username required"})
		}
		return ctx.Status(http.StatusCreated).JSON(map[string]string{"status": "created", "user": input.Username})
	})

	// Valid POST
	body := strings.NewReader(`{"username":"goks_user"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/users", body)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d", w.Code)
	}

	// Invalid JSON body
	badReq := httptest.NewRequest(http.MethodPost, "/api/users", strings.NewReader("invalid-json"))
	wBad := httptest.NewRecorder()
	r.ServeHTTP(wBad, badReq)
	if wBad.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request, got %d", wBad.Code)
	}

	// Validation error
	emptyReq := httptest.NewRequest(http.MethodPost, "/api/users", strings.NewReader(`{"username":""}`))
	wEmpty := httptest.NewRecorder()
	r.ServeHTTP(wEmpty, emptyReq)
	if wEmpty.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 Unprocessable Entity, got %d", wEmpty.Code)
	}
}

func TestRouter_PanicRecoveryInHandler(t *testing.T) {
	r := router.New()
	r.Use(router.Recover())

	r.GET("/api/panic", func(ctx *router.Context) error {
		var ptr *string
		_ = *ptr // deliberate nil pointer dereference
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/api/panic", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 Internal Server Error on panic, got %d", w.Code)
	}
}

func TestRouter_Precedence(t *testing.T) {
	r := router.New()

	// Register in reverse order of precedence: wildcard first, dynamic second, static third
	r.GET("/posts/*all", func(ctx *router.Context) error {
		return ctx.Text("wildcard:" + ctx.Param("all"))
	})
	r.GET("/posts/:id", func(ctx *router.Context) error {
		return ctx.Text("dynamic:" + ctx.Param("id"))
	})
	r.GET("/posts/featured", func(ctx *router.Context) error {
		return ctx.Text("static:featured")
	})

	// 1. Static must win over dynamic and wildcard
	req1 := httptest.NewRequest("GET", "/posts/featured", nil)
	rec1 := httptest.NewRecorder()
	r.ServeHTTP(rec1, req1)
	if rec1.Body.String() != "static:featured" {
		t.Fatalf("expected static match, got %q", rec1.Body.String())
	}

	// 2. Dynamic must win over wildcard for single segment
	req2 := httptest.NewRequest("GET", "/posts/123", nil)
	rec2 := httptest.NewRecorder()
	r.ServeHTTP(rec2, req2)
	if rec2.Body.String() != "dynamic:123" {
		t.Fatalf("expected dynamic match, got %q", rec2.Body.String())
	}

	// 3. Multi-segment falls through to wildcard
	req3 := httptest.NewRequest("GET", "/posts/123/comments", nil)
	rec3 := httptest.NewRecorder()
	r.ServeHTTP(rec3, req3)
	if rec3.Body.String() != "wildcard:123/comments" {
		t.Fatalf("expected wildcard match, got %q", rec3.Body.String())
	}
}

func TestRouter_ParamUnescaping(t *testing.T) {
	r := router.New()
	r.GET("/profile/:name", func(ctx *router.Context) error {
		return ctx.Text(ctx.Param("name"))
	})

	req := httptest.NewRequest("GET", "/profile/Jane%20Doe", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Body.String() != "Jane Doe" {
		t.Fatalf("expected unescaped param 'Jane Doe', got %q", rec.Body.String())
	}
}

func TestRouter_MethodNotAllowed(t *testing.T) {
	r := router.New()
	r.POST("/submit", func(ctx *router.Context) error {
		return ctx.Text("submitted")
	})

	// GET to a POST-only route
	req := httptest.NewRequest("GET", "/submit", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405 Method Not Allowed, got %d", rec.Code)
	}
	if rec.Header().Get("Allow") != "POST" {
		t.Fatalf("expected Allow: POST header, got %q", rec.Header().Get("Allow"))
	}
}

func TestRouter_DuplicateRouteReplacement(t *testing.T) {
	r := router.New()
	r.GET("/dup", func(ctx *router.Context) error {
		return ctx.Text("v1")
	})
	r.GET("/dup", func(ctx *router.Context) error {
		return ctx.Text("v2")
	})

	if len(r.Routes()) != 1 {
		t.Fatalf("expected duplicate route to update existing entry, got count %d", len(r.Routes()))
	}

	req := httptest.NewRequest("GET", "/dup", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Body.String() != "v2" {
		t.Fatalf("expected updated handler response 'v2', got %q", rec.Body.String())
	}
}

func TestRouter_ContextErrorMethod(t *testing.T) {
	r := router.New()
	r.Use(router.RequestID())
	r.GET("/api/item", func(ctx *router.Context) error {
		return ctx.Error(http.StatusNotFound, "Item not found")
	})

	req := httptest.NewRequest("GET", "/api/item", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
	var errResp router.ErrorResponse
	if err := json.NewDecoder(rec.Body).Decode(&errResp); err != nil {
		t.Fatalf("failed to decode JSON error: %v", err)
	}
	if errResp.Error.Code != 404 || errResp.Error.Message != "Item not found" {
		t.Fatalf("unexpected error payload: %+v", errResp)
	}
	if errResp.Error.RequestID == "" {
		t.Fatal("expected request_id in error payload")
	}
}

func TestRouter_NestedGroups(t *testing.T) {
	r := router.New()

	var order []string

	mw1 := func(next router.Handler) router.Handler {
		return func(ctx *router.Context) error {
			order = append(order, "mw1")
			return next(ctx)
		}
	}
	mw2 := func(next router.Handler) router.Handler {
		return func(ctx *router.Context) error {
			order = append(order, "mw2")
			return next(ctx)
		}
	}

	api := r.Group("/api", mw1)
	v1 := api.Group("/v1", mw2)

	v1.GET("/users", func(ctx *router.Context) error {
		return ctx.Text("users-list")
	})

	req := httptest.NewRequest("GET", "/api/v1/users", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if rec.Body.String() != "users-list" {
		t.Fatalf("expected 'users-list', got %q", rec.Body.String())
	}

	if len(order) != 2 || order[0] != "mw1" || order[1] != "mw2" {
		t.Fatalf("expected middlewares mw1 then mw2, got %v", order)
	}
}

func TestRouter_ConcurrentAccess(t *testing.T) {
	r := router.New()
	r.GET("/static", func(ctx *router.Context) error {
		return ctx.Text("static")
	})

	var wg sync.WaitGroup

	// Concurrently serve requests
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				req := httptest.NewRequest("GET", "/static", nil)
				rec := httptest.NewRecorder()
				r.ServeHTTP(rec, req)
				if rec.Code != 200 {
					t.Errorf("expected 200, got %d", rec.Code)
				}
			}
		}()
	}

	// Concurrently add new routes and inspect routes
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				path := fmt.Sprintf("/dyn-%d-%d", id, j)
				r.GET(path, func(ctx *router.Context) error {
					return ctx.Text("dyn")
				})
				_ = r.Routes()
			}
		}(i)
	}

	wg.Wait()
}
