package studio

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
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

func TestStudio_Database_TablesAndRows(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "app.db")

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}
	defer db.Close()

	if _, err := db.Exec(`CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, email TEXT); INSERT INTO users (name, email) VALUES ('Budi', 'budi@example.com');`); err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	st := New(Config{
		AppDir:  tmpDir,
		DevMode: true,
	})

	// 1. Check tables API
	req := httptest.NewRequest("GET", "/__goks/api/db/tables", nil)
	rec := httptest.NewRecorder()
	st.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Expected 200 for tables API, got %d", rec.Code)
	}
	var tables []string
	if err := json.NewDecoder(rec.Body).Decode(&tables); err != nil {
		t.Fatalf("failed to decode tables: %v", err)
	}
	if len(tables) != 1 || tables[0] != "users" {
		t.Fatalf("Expected ['users'], got %v", tables)
	}

	// 2. Check table rows API
	req = httptest.NewRequest("GET", "/__goks/api/db/table?name=users", nil)
	rec = httptest.NewRecorder()
	st.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Expected 200 for table rows API, got %d", rec.Code)
	}
	var res map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&res); err != nil {
		t.Fatalf("failed to decode table rows: %v", err)
	}
	if int(res["total"].(float64)) != 1 {
		t.Fatalf("Expected 1 total row, got %v", res["total"])
	}
}

func TestStudio_Migrations_DatabaseFolder(t *testing.T) {
	tmpDir := t.TempDir()
	migDir := filepath.Join(tmpDir, "database", "migrations")
	if err := os.MkdirAll(migDir, 0755); err != nil {
		t.Fatalf("failed to create migrations dir: %v", err)
	}

	migFile := filepath.Join(migDir, "20260917130913_user.sql")
	if err := os.WriteFile(migFile, []byte("-- UP\nCREATE TABLE users (id INT);\n-- DOWN\nDROP TABLE users;"), 0644); err != nil {
		t.Fatalf("failed to write migration file: %v", err)
	}

	dbPath := filepath.Join(tmpDir, "database", "app.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}
	defer db.Close()

	if _, err := db.Exec(`CREATE TABLE goks_migrations (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		version TEXT NOT NULL UNIQUE,
		name TEXT NOT NULL,
		batch INTEGER NOT NULL,
		applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	);
	INSERT INTO goks_migrations (version, name, batch) VALUES ('20260917130913', 'user', 1);`); err != nil {
		t.Fatalf("failed to setup migrations table: %v", err)
	}

	st := New(Config{
		AppDir:  tmpDir,
		DevMode: true,
	})

	req := httptest.NewRequest("GET", "/__goks/api/migrations", nil)
	rec := httptest.NewRecorder()
	st.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Expected 200 for migrations API, got %d", rec.Code)
	}
	var migs []map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&migs); err != nil {
		t.Fatalf("failed to decode migrations: %v", err)
	}
	if len(migs) != 1 {
		t.Fatalf("Expected 1 migration, got %d", len(migs))
	}
	if migs[0]["name"] != "20260917130913_user.sql" {
		t.Errorf("Expected migration name 20260917130913_user.sql, got %v", migs[0]["name"])
	}
	if migs[0]["applied"] != true {
		t.Errorf("Expected migration to be applied = true, got %v", migs[0]["applied"])
	}
	if int(migs[0]["batch"].(float64)) != 1 {
		t.Errorf("Expected batch = 1, got %v", migs[0]["batch"])
	}
}
