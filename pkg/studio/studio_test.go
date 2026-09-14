package studio

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/misbakhul29/goks/internal/version"
	"github.com/misbakhul29/goks/pkg/action"
	"github.com/misbakhul29/goks/pkg/router"
	"github.com/misbakhul29/goks/pkg/rpc"
)

func TestStudio_DevModeSecurityGuard(t *testing.T) {
	// In production (DevMode = false), all Studio endpoints MUST yield 404
	prodStudio := New(Config{
		DevMode: false,
	})

	paths := []string{
		"/__goks",
		"/__goks/",
		"/__goks/api/overview",
		"/__goks/api/routes",
		"/__goks/api/db/tables",
	}

	for _, p := range paths {
		req := httptest.NewRequest("GET", p, nil)
		rec := httptest.NewRecorder()
		prodStudio.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("Expected 404 in production mode for %s, got %d", p, rec.Code)
		}
	}
}

func TestStudio_OverviewAndRoutes(t *testing.T) {
	r := router.New()
	r.GET("/", func(c *router.Context) error { return nil })
	r.GET("/about", func(c *router.Context) error { return nil })
	r.GET("/api/users", func(c *router.Context) error { return nil })

	st := New(Config{
		DevMode: true,
		Router:  r,
	})

	// 1. Test Dashboard HTML
	req := httptest.NewRequest("GET", "/__goks", nil)
	rec := httptest.NewRecorder()
	st.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK for /__goks dashboard, got %d", rec.Code)
	}
	body := rec.Body.String()
	if body == "" || rec.Header().Get("Content-Type") != "text/html; charset=utf-8" {
		t.Errorf("Expected HTML response for /__goks")
	}

	// 2. Test Overview API
	req = httptest.NewRequest("GET", "/__goks/api/overview", nil)
	rec = httptest.NewRecorder()
	st.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK for /__goks/api/overview, got %d", rec.Code)
	}
	var overview map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&overview); err != nil {
		t.Fatalf("Failed to parse overview JSON: %v", err)
	}
	if overview["framework"] != "GoKS" {
		t.Errorf("Expected framework GoKS, got %v", overview["framework"])
	}
	if overview["version"] != version.Current() {
		t.Errorf("Expected version %s, got %v", version.Current(), overview["version"])
	}

	// 3. Test Routes API
	req = httptest.NewRequest("GET", "/__goks/api/routes", nil)
	rec = httptest.NewRecorder()
	st.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK for /__goks/api/routes, got %d", rec.Code)
	}
	var routes []map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&routes); err != nil {
		t.Fatalf("Failed to parse routes JSON: %v", err)
	}
	if len(routes) != 3 {
		t.Fatalf("Expected 3 routes, got %d", len(routes))
	}
}

func TestStudio_ActionsAndRPC(t *testing.T) {
	action.Register("studioTestAction", func(ctx *action.Context) (any, error) {
		return "ok", nil
	})
	rpc.Register("studioTestRPC", func(arg string) (string, error) {
		return "pong", nil
	})

	st := New(Config{
		DevMode: true,
	})

	// Test actions API
	req := httptest.NewRequest("GET", "/__goks/api/actions", nil)
	rec := httptest.NewRecorder()
	st.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Expected 200 for actions API, got %d", rec.Code)
	}
	var actions []map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&actions); err != nil {
		t.Fatalf("Failed to decode actions: %v", err)
	}
	foundAction := false
	for _, a := range actions {
		if a["name"] == "studioTestAction" {
			foundAction = true
			break
		}
	}
	if !foundAction {
		t.Errorf("Expected to find studioTestAction in actions list")
	}

	// Test RPC API
	req = httptest.NewRequest("GET", "/__goks/api/rpc", nil)
	rec = httptest.NewRecorder()
	st.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Expected 200 for rpc API, got %d", rec.Code)
	}
	var rpcList []map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&rpcList); err != nil {
		t.Fatalf("Failed to decode rpc: %v", err)
	}
	foundRPC := false
	for _, m := range rpcList {
		if m["method"] == "studioTestRPC" {
			foundRPC = true
			break
		}
	}
	if !foundRPC {
		t.Errorf("Expected to find studioTestRPC in RPC list")
	}
}

func TestStudio_DBSecurity_SQLInjection(t *testing.T) {
	st := New(Config{
		DevMode: true,
	})

	maliciousNames := []string{
		"users; DROP TABLE users;--",
		"users WHERE 1=1",
		"../users",
		"users/../../etc/passwd",
		"users' OR '1'='1",
	}

	for _, name := range maliciousNames {
		req := httptest.NewRequest("GET", "/__goks/api/db/table?name="+url.QueryEscape(name), nil)
		rec := httptest.NewRecorder()
		st.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 Bad Request for malicious table name %q, got %d", name, rec.Code)
		}
	}
}
