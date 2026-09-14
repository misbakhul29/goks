package router_test

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/misbakhul29/goks/pkg/router"
)

func TestMiddlewareOrderAndExecution(t *testing.T) {
	var executionOrder []string
	var mu sync.Mutex

	record := func(name string) router.MiddlewareFunc {
		return func(next router.Handler) router.Handler {
			return func(c *router.Context) error {
				mu.Lock()
				executionOrder = append(executionOrder, name+"_enter")
				mu.Unlock()

				err := next(c)

				mu.Lock()
				executionOrder = append(executionOrder, name+"_exit")
				mu.Unlock()
				return err
			}
		}
	}

	r := router.New()
	r.Use(record("first"), record("second"), record("third"))
	r.GET("/order", func(c *router.Context) error {
		mu.Lock()
		executionOrder = append(executionOrder, "handler")
		mu.Unlock()
		return c.Text("ok")
	})

	req := httptest.NewRequest("GET", "/order", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	expected := []string{
		"first_enter", "second_enter", "third_enter",
		"handler",
		"third_exit", "second_exit", "first_exit",
	}

	if len(executionOrder) != len(expected) {
		t.Fatalf("expected %d steps, got %d: %v", len(expected), len(executionOrder), executionOrder)
	}
	for i, exp := range expected {
		if executionOrder[i] != exp {
			t.Errorf("step %d: expected %s, got %s", i, exp, executionOrder[i])
		}
	}
}

func TestMiddlewareShortCircuit(t *testing.T) {
	r := router.New()
	handlerReached := false

	blockingMiddleware := func(next router.Handler) router.Handler {
		return func(c *router.Context) error {
			c.Status(http.StatusForbidden)
			return c.Text("forbidden by middleware")
		}
	}

	r.Use(blockingMiddleware)
	r.GET("/secret", func(c *router.Context) error {
		handlerReached = true
		return c.Text("super secret")
	})

	req := httptest.NewRequest("GET", "/secret", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected status %d, got %d", http.StatusForbidden, rec.Code)
	}
	if handlerReached {
		t.Error("expected handler NOT to be reached due to short-circuit")
	}
	if !strings.Contains(rec.Body.String(), "forbidden by middleware") {
		t.Errorf("unexpected body: %s", rec.Body.String())
	}
}

func TestRecoverMiddleware_SafeResponse(t *testing.T) {
	r := router.New()
	r.Use(router.Recover())
	r.GET("/panic", func(c *router.Context) error {
		panic("database connection string postgres://secret:password@db leaked")
	})

	req := httptest.NewRequest("GET", "/panic", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", rec.Code)
	}
	body := rec.Body.String()
	if strings.Contains(body, "password") || strings.Contains(body, "postgres://") {
		t.Errorf("panic message leaked into response body: %s", body)
	}
	if !strings.Contains(body, "Internal Server Error") {
		t.Errorf("expected standard 500 message, got: %s", body)
	}
}

func TestRequestIDMiddleware(t *testing.T) {
	r := router.New()
	r.Use(router.RequestID())

	var capturedReqID string
	r.GET("/id", func(c *router.Context) error {
		capturedReqID = c.RequestID()
		return c.Text("ok")
	})

	// Case 1: No incoming X-Request-Id header -> auto-generated
	req1 := httptest.NewRequest("GET", "/id", nil)
	rec1 := httptest.NewRecorder()
	r.ServeHTTP(rec1, req1)

	respID1 := rec1.Header().Get("X-Request-Id")
	if respID1 == "" {
		t.Fatal("expected X-Request-Id in response header")
	}
	if capturedReqID != respID1 {
		t.Fatalf("expected context request ID %s to match header %s", capturedReqID, respID1)
	}

	// Case 2: Provided incoming X-Request-Id header -> preserved
	customID := "custom-uuid-123456"
	req2 := httptest.NewRequest("GET", "/id", nil)
	req2.Header.Set("X-Request-Id", customID)
	rec2 := httptest.NewRecorder()
	r.ServeHTTP(rec2, req2)

	if rec2.Header().Get("X-Request-Id") != customID {
		t.Fatalf("expected preserved X-Request-Id %s, got %s", customID, rec2.Header().Get("X-Request-Id"))
	}
}

func TestSecureMiddleware(t *testing.T) {
	r := router.New()
	r.Use(router.Secure())
	r.GET("/secure", func(c *router.Context) error {
		return c.Text("secure")
	})

	// Plain HTTP: should NOT include Strict-Transport-Security to preserve local dev
	reqHTTP := httptest.NewRequest("GET", "/secure", nil)
	recHTTP := httptest.NewRecorder()
	r.ServeHTTP(recHTTP, reqHTTP)

	if recHTTP.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Error("expected nosniff header")
	}
	if recHTTP.Header().Get("X-Frame-Options") != "DENY" {
		t.Error("expected DENY frame options")
	}
	if recHTTP.Header().Get("Strict-Transport-Security") != "" {
		t.Error("HSTS should not be set over plain HTTP")
	}

	// HTTPS proxy request: should include Strict-Transport-Security
	reqHTTPS := httptest.NewRequest("GET", "/secure", nil)
	reqHTTPS.Header.Set("X-Forwarded-Proto", "https")
	recHTTPS := httptest.NewRecorder()
	r.ServeHTTP(recHTTPS, reqHTTPS)

	if recHTTPS.Header().Get("Strict-Transport-Security") == "" {
		t.Error("expected HSTS header when X-Forwarded-Proto is https")
	}
}

func TestMaxBytesMiddleware(t *testing.T) {
	r := router.New()
	r.Use(router.MaxBytes(10)) // limit body to 10 bytes
	r.POST("/upload", func(c *router.Context) error {
		body, err := io.ReadAll(c.Request().Body)
		if err != nil {
			return err
		}
		return c.Text(string(body))
	})

	// Exceeding Content-Length check
	reqLarge := httptest.NewRequest("POST", "/upload", bytes.NewReader([]byte("this body is clearly longer than 10 bytes")))
	recLarge := httptest.NewRecorder()
	r.ServeHTTP(recLarge, reqLarge)

	if recLarge.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("expected 413, got %d", recLarge.Code)
	}

	// Under limit
	reqSmall := httptest.NewRequest("POST", "/upload", bytes.NewReader([]byte("small")))
	recSmall := httptest.NewRecorder()
	r.ServeHTTP(recSmall, reqSmall)

	if recSmall.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", recSmall.Code)
	}
}

func TestTimeoutMiddleware(t *testing.T) {
	r := router.New()
	r.Use(router.Timeout(50 * time.Millisecond))
	r.GET("/slow", func(c *router.Context) error {
		select {
		case <-time.After(200 * time.Millisecond):
			return c.Text("too late")
		case <-c.Request().Context().Done():
			return c.Request().Context().Err()
		}
	})

	req := httptest.NewRequest("GET", "/slow", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusGatewayTimeout {
		t.Errorf("expected 504 Gateway Timeout, got %d", rec.Code)
	}
}

func TestCORSMiddleware(t *testing.T) {
	r := router.New()
	r.Use(router.CORS("https://app.example.com"))
	r.GET("/api/data", func(c *router.Context) error {
		return c.JSON(map[string]string{"status": "ok"})
	})

	// Preflight request
	reqOptions := httptest.NewRequest("OPTIONS", "/api/data", nil)
	reqOptions.Header.Set("Origin", "https://app.example.com")
	recOptions := httptest.NewRecorder()
	r.ServeHTTP(recOptions, reqOptions)

	if recOptions.Code != http.StatusNoContent {
		t.Errorf("expected 204 No Content on preflight, got %d", recOptions.Code)
	}
	if recOptions.Header().Get("Access-Control-Allow-Origin") != "https://app.example.com" {
		t.Errorf("unexpected allow-origin: %s", recOptions.Header().Get("Access-Control-Allow-Origin"))
	}

	// Disallowed origin
	reqDisallowed := httptest.NewRequest("GET", "/api/data", nil)
	reqDisallowed.Header.Set("Origin", "https://evil.com")
	recDisallowed := httptest.NewRecorder()
	r.ServeHTTP(recDisallowed, reqDisallowed)

	if recDisallowed.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Errorf("expected empty allow-origin for unlisted origin, got %s", recDisallowed.Header().Get("Access-Control-Allow-Origin"))
	}
}
