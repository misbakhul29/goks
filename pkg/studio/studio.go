// Package studio provides the embedded GoKS Studio DevTools dashboard.
// It is exclusively available in development mode (DevMode: true) to inspect
// routes, database tables, server actions, RPC methods, migrations, and runtime telemetry.
package studio

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/misbakhul29/goks/internal/version"
	"github.com/misbakhul29/goks/pkg/action"
	"github.com/misbakhul29/goks/pkg/router"
	"github.com/misbakhul29/goks/pkg/rpc"
)

var validTableName = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

var startTime = time.Now()

// Config configures GoKS Studio.
type Config struct {
	AppDir  string
	Router  *router.Router
	DevMode bool
	DBPath  string // optional custom db path; defaults to app.db in AppDir
}

// Studio provides the DevTools HTTP handler.
type Studio struct {
	cfg Config
	mu  sync.RWMutex
}

// New creates a new GoKS Studio instance.
func New(cfg Config) *Studio {
	if cfg.AppDir == "" {
		cfg.AppDir = "."
	}
	if cfg.DBPath == "" {
		cfg.DBPath = filepath.Join(cfg.AppDir, "app.db")
	}
	return &Studio{cfg: cfg}
}

// ServeHTTP handles requests to /__goks and /__goks/* endpoints.
func (s *Studio) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// SECURITY GUARD: Never expose Studio DevTools in production
	if !s.cfg.DevMode {
		http.NotFound(w, r)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/__goks")
	if path == "" || path == "/" {
		s.serveDashboardHTML(w, r)
		return
	}

	switch {
	case path == "/api/overview":
		s.handleOverview(w, r)
	case path == "/api/routes":
		s.handleRoutes(w, r)
	case path == "/api/actions":
		s.handleActions(w, r)
	case path == "/api/rpc":
		s.handleRPC(w, r)
	case path == "/api/db/tables":
		s.handleDBTables(w, r)
	case path == "/api/db/table":
		s.handleDBTableRows(w, r)
	case path == "/api/migrations":
		s.handleMigrations(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (s *Studio) handleOverview(w http.ResponseWriter, r *http.Request) {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	routesCount := 0
	if s.cfg.Router != nil {
		routesCount = len(s.cfg.Router.Routes())
	}

	actions := action.RegisteredActions()
	rpcMethods := rpc.RegisteredMethods()

	tables := s.getSQLiteTables()

	data := map[string]any{
		"framework":     "GoKS",
		"version":       version.Current(),
		"go_version":    runtime.Version(),
		"os":            runtime.GOOS,
		"arch":          runtime.GOARCH,
		"goroutines":    runtime.NumGoroutine(),
		"alloc_bytes":   mem.Alloc,
		"sys_bytes":     mem.Sys,
		"gc_cycles":     mem.NumGC,
		"uptime_sec":    int(time.Since(startTime).Seconds()),
		"routes_count":  routesCount,
		"actions_count": len(actions),
		"rpc_count":     len(rpcMethods),
		"tables_count":  len(tables),
		"dev_mode":      s.cfg.DevMode,
		"app_dir":       s.cfg.AppDir,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(data)
}

func (s *Studio) handleRoutes(w http.ResponseWriter, r *http.Request) {
	var routes []router.RouteInfo
	if s.cfg.Router != nil {
		routes = s.cfg.Router.Routes()
	}

	type RouteDetail struct {
		Method  string `json:"method"`
		Pattern string `json:"pattern"`
		Type    string `json:"type"` // "Page" or "API Route" or "System"
	}

	details := make([]RouteDetail, 0, len(routes))
	for _, rt := range routes {
		t := "Page"
		if strings.HasPrefix(rt.Pattern, "/api") {
			t = "API Route"
		} else if strings.HasPrefix(rt.Pattern, "/__goks") {
			t = "System / DevTool"
		}
		details = append(details, RouteDetail{
			Method:  rt.Method,
			Pattern: rt.Pattern,
			Type:    t,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(details)
}

func (s *Studio) handleActions(w http.ResponseWriter, r *http.Request) {
	actions := action.RegisteredActions()
	type ActionInfo struct {
		Name     string `json:"name"`
		Endpoint string `json:"endpoint"`
	}
	res := make([]ActionInfo, 0, len(actions))
	for _, a := range actions {
		res = append(res, ActionInfo{
			Name:     a,
			Endpoint: action.URL(a),
		})
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(res)
}

func (s *Studio) handleRPC(w http.ResponseWriter, r *http.Request) {
	methods := rpc.RegisteredMethods()
	type RPCInfo struct {
		Method   string `json:"method"`
		Endpoint string `json:"endpoint"`
	}
	res := make([]RPCInfo, 0, len(methods))
	for _, m := range methods {
		res = append(res, RPCInfo{
			Method:   m,
			Endpoint: "/__goks_rpc?method=" + m,
		})
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(res)
}

func (s *Studio) getSQLiteTables() []string {
	if _, err := os.Stat(s.cfg.DBPath); os.IsNotExist(err) {
		return nil
	}
	db, err := sql.Open("sqlite3", s.cfg.DBPath)
	if err != nil {
		return nil
	}
	defer db.Close()

	rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name")
	if err != nil {
		return nil
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err == nil {
			tables = append(tables, name)
		}
	}
	return tables
}

func (s *Studio) handleDBTables(w http.ResponseWriter, r *http.Request) {
	tables := s.getSQLiteTables()
	if tables == nil {
		tables = []string{}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(tables)
}

func (s *Studio) handleDBTableRows(w http.ResponseWriter, r *http.Request) {
	tableName := strings.TrimSpace(r.URL.Query().Get("name"))
	// SECURITY GUARD: Strictly validate table name to prevent SQL injection
	if !validTableName.MatchString(tableName) {
		http.Error(w, "invalid or illegal table name", http.StatusBadRequest)
		return
	}

	limit := 50
	if lStr := r.URL.Query().Get("limit"); lStr != "" {
		if l, err := strconv.Atoi(lStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	offset := 0
	if oStr := r.URL.Query().Get("offset"); oStr != "" {
		if o, err := strconv.Atoi(oStr); err == nil && o >= 0 {
			offset = o
		}
	}

	if _, err := os.Stat(s.cfg.DBPath); os.IsNotExist(err) {
		http.Error(w, "database file not found", http.StatusNotFound)
		return
	}

	db, err := sql.Open("sqlite3", s.cfg.DBPath)
	if err != nil {
		http.Error(w, "failed to open database: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer db.Close()

	// 1. Total row count
	var total int
	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM "%s"`, tableName)
	if err := db.QueryRow(countQuery).Scan(&total); err != nil {
		http.Error(w, "failed to query table count: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 2. Fetch rows (Read-only, parameterized LIMIT/OFFSET)
	selectQuery := fmt.Sprintf(`SELECT * FROM "%s" LIMIT %d OFFSET %d`, tableName, limit, offset)
	rows, err := db.Query(selectQuery)
	if err != nil {
		http.Error(w, "failed to query rows: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		http.Error(w, "failed to inspect columns: "+err.Error(), http.StatusInternalServerError)
		return
	}

	results := make([]map[string]any, 0)
	for rows.Next() {
		values := make([]any, len(columns))
		valuePtrs := make([]any, len(columns))
		for i := range columns {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			continue
		}

		rowMap := make(map[string]any)
		for i, col := range columns {
			val := values[i]
			if b, ok := val.([]byte); ok {
				rowMap[col] = string(b)
			} else {
				rowMap[col] = val
			}
		}
		results = append(results, rowMap)
	}

	data := map[string]any{
		"table":   tableName,
		"columns": columns,
		"rows":    results,
		"total":   total,
		"limit":   limit,
		"offset":  offset,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(data)
}

func (s *Studio) handleMigrations(w http.ResponseWriter, r *http.Request) {
	migDir := filepath.Join(s.cfg.AppDir, "migrations")
	type MigrationItem struct {
		Name    string `json:"name"`
		Applied bool   `json:"applied"`
		Batch   int    `json:"batch,omitempty"`
	}

	var items []MigrationItem
	if entries, err := os.ReadDir(migDir); err == nil {
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
				items = append(items, MigrationItem{
					Name:    e.Name(),
					Applied: false,
				})
			}
		}
	}

	// If database has tracking table goks_migrations, correlate applied status
	if _, err := os.Stat(s.cfg.DBPath); err == nil {
		if db, err := sql.Open("sqlite3", s.cfg.DBPath); err == nil {
			defer db.Close()
			rows, err := db.Query("SELECT migration, batch FROM goks_migrations")
			if err == nil {
				defer rows.Close()
				appliedMap := make(map[string]int)
				for rows.Next() {
					var name string
					var batch int
					if err := rows.Scan(&name, &batch); err == nil {
						appliedMap[name] = batch
					}
				}
				for i := range items {
					if b, ok := appliedMap[items[i].Name]; ok {
						items[i].Applied = true
						items[i].Batch = b
					}
				}
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(items)
}

func (s *Studio) serveDashboardHTML(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(dashboardHTML))
}

const dashboardHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>GoKS Studio | Developer Tools</title>
  <link rel="preconnect" href="https://fonts.googleapis.com">
  <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
  <link href="https://fonts.googleapis.com/css2?family=Plus+Jakarta+Sans:wght@400;500;600;700&family=JetBrains+Mono:wght@400;500;600&display=swap" rel="stylesheet">
  <style>
    :root {
      --bg: #09090b;
      --card: #121215;
      --card-hover: #18181b;
      --border: #27272a;
      --border-accent: #3f3f46;
      --text: #f4f4f5;
      --text-muted: #a1a1aa;
      --primary: #6366f1;
      --primary-hover: #4f46e5;
      --primary-light: rgba(99, 102, 241, 0.15);
      --cyan: #06b6d4;
      --emerald: #10b981;
      --amber: #f59e0b;
      --rose: #f43f5e;
      --font-sans: 'Plus Jakarta Sans', -apple-system, sans-serif;
      --font-mono: 'JetBrains Mono', monospace;
    }
    * { box-sizing: border-box; margin: 0; padding: 0; }
    body {
      background: var(--bg);
      color: var(--text);
      font-family: var(--font-sans);
      min-height: 100vh;
      display: flex;
      flex-direction: column;
    }
    header {
      background: rgba(18, 18, 21, 0.8);
      backdrop-filter: blur(12px);
      border-bottom: 1px solid var(--border);
      padding: 0.85rem 1.75rem;
      display: flex;
      align-items: center;
      justify-content: space-between;
      position: sticky;
      top: 0;
      z-index: 50;
    }
    .brand {
      display: flex;
      align-items: center;
      gap: 0.75rem;
      text-decoration: none;
      color: inherit;
    }
    .brand-logo {
      width: 32px;
      height: 32px;
      border-radius: 8px;
      background: linear-gradient(135deg, var(--cyan), var(--primary));
      display: flex;
      align-items: center;
      justify-content: center;
      font-weight: 700;
      font-size: 1rem;
      color: white;
      box-shadow: 0 0 20px rgba(99, 102, 241, 0.35);
    }
    .brand-text { font-size: 1.15rem; font-weight: 700; letter-spacing: -0.02em; }
    .brand-badge {
      font-size: 0.7rem;
      background: var(--primary-light);
      color: #818cf8;
      border: 1px solid rgba(99, 102, 241, 0.3);
      padding: 0.15rem 0.5rem;
      border-radius: 9999px;
      font-weight: 600;
    }
    .header-actions { display: flex; align-items: center; gap: 1rem; }
    .btn {
      background: var(--card);
      border: 1px solid var(--border);
      color: var(--text);
      padding: 0.45rem 0.9rem;
      border-radius: 6px;
      font-size: 0.85rem;
      font-weight: 500;
      cursor: pointer;
      display: inline-flex;
      align-items: center;
      gap: 0.4rem;
      transition: all 0.15s ease;
      text-decoration: none;
    }
    .btn:hover { background: var(--card-hover); border-color: var(--border-accent); }
    .btn-primary {
      background: var(--primary);
      border-color: var(--primary);
      color: white;
    }
    .btn-primary:hover { background: var(--primary-hover); }
    .container {
      max-width: 1300px;
      margin: 0 auto;
      padding: 2rem 1.75rem;
      width: 100%;
      flex: 1;
    }
    .tabs {
      display: flex;
      gap: 0.5rem;
      border-bottom: 1px solid var(--border);
      margin-bottom: 2rem;
      overflow-x: auto;
    }
    .tab {
      padding: 0.65rem 1.25rem;
      font-size: 0.9rem;
      font-weight: 500;
      color: var(--text-muted);
      cursor: pointer;
      border-bottom: 2px solid transparent;
      transition: all 0.2s;
      white-space: nowrap;
      display: flex;
      align-items: center;
      gap: 0.5rem;
    }
    .tab:hover { color: var(--text); }
    .tab.active {
      color: #818cf8;
      border-bottom-color: var(--primary);
      font-weight: 600;
    }
    .tab-badge {
      font-size: 0.7rem;
      padding: 0.1rem 0.45rem;
      border-radius: 9999px;
      background: var(--card);
      border: 1px solid var(--border);
    }
    .tab-content { display: none; }
    .tab-content.active { display: block; }
    .grid { display: grid; gap: 1.25rem; }
    .grid-4 { grid-template-columns: repeat(auto-fit, minmax(260px, 1fr)); }
    .grid-2 { grid-template-columns: repeat(auto-fit, minmax(450px, 1fr)); }
    .card {
      background: var(--card);
      border: 1px solid var(--border);
      border-radius: 12px;
      padding: 1.5rem;
      transition: border-color 0.2s;
    }
    .card:hover { border-color: var(--border-accent); }
    .card-title {
      font-size: 0.85rem;
      text-transform: uppercase;
      letter-spacing: 0.05em;
      color: var(--text-muted);
      margin-bottom: 0.5rem;
      font-weight: 600;
    }
    .card-value {
      font-size: 1.75rem;
      font-weight: 700;
      letter-spacing: -0.03em;
      color: var(--text);
    }
    .card-desc {
      font-size: 0.8rem;
      color: var(--text-muted);
      margin-top: 0.35rem;
    }
    .table-container {
      background: var(--card);
      border: 1px solid var(--border);
      border-radius: 12px;
      overflow: hidden;
    }
    table {
      width: 100%;
      border-collapse: collapse;
      text-align: left;
      font-size: 0.875rem;
    }
    th {
      background: rgba(255, 255, 255, 0.02);
      border-bottom: 1px solid var(--border);
      padding: 0.85rem 1.25rem;
      font-size: 0.75rem;
      text-transform: uppercase;
      letter-spacing: 0.05em;
      color: var(--text-muted);
      font-weight: 600;
    }
    td {
      padding: 0.9rem 1.25rem;
      border-bottom: 1px solid var(--border);
      vertical-align: middle;
    }
    tr:last-child td { border-bottom: none; }
    tr:hover td { background: rgba(255, 255, 255, 0.015); }
    .badge {
      font-size: 0.7rem;
      font-weight: 600;
      padding: 0.2rem 0.55rem;
      border-radius: 6px;
      display: inline-block;
      font-family: var(--font-mono);
    }
    .badge-get { background: rgba(16, 185, 129, 0.15); color: #34d399; border: 1px solid rgba(16, 185, 129, 0.3); }
    .badge-post { background: rgba(6, 182, 212, 0.15); color: #22d3ee; border: 1px solid rgba(6, 182, 212, 0.3); }
    .badge-put { background: rgba(245, 158, 11, 0.15); color: #fbbf24; border: 1px solid rgba(245, 158, 11, 0.3); }
    .badge-delete { background: rgba(244, 63, 94, 0.15); color: #fb7185; border: 1px solid rgba(244, 63, 94, 0.3); }
    .badge-patch { background: rgba(168, 85, 247, 0.15); color: #c084fc; border: 1px solid rgba(168, 85, 247, 0.3); }
    .badge-page { background: rgba(99, 102, 241, 0.15); color: #a5b4fc; border: 1px solid rgba(99, 102, 241, 0.3); }
    .badge-applied { background: rgba(16, 185, 129, 0.15); color: #34d399; }
    .badge-pending { background: rgba(245, 158, 11, 0.15); color: #fbbf24; }
    .code {
      font-family: var(--font-mono);
      font-size: 0.85rem;
      color: #38bdf8;
    }
    .empty-state {
      padding: 3rem 1rem;
      text-align: center;
      color: var(--text-muted);
    }
    .db-layout {
      display: grid;
      grid-template-columns: 260px 1fr;
      gap: 1.5rem;
    }
    .db-sidebar {
      background: var(--card);
      border: 1px solid var(--border);
      border-radius: 12px;
      padding: 1rem;
      height: fit-content;
    }
    .table-list-item {
      padding: 0.6rem 0.85rem;
      border-radius: 6px;
      cursor: pointer;
      display: flex;
      align-items: center;
      justify-content: space-between;
      color: var(--text-muted);
      font-size: 0.875rem;
      transition: all 0.15s;
      margin-bottom: 0.25rem;
    }
    .table-list-item:hover { background: var(--card-hover); color: var(--text); }
    .table-list-item.active {
      background: var(--primary-light);
      color: #818cf8;
      font-weight: 600;
    }
    footer {
      border-top: 1px solid var(--border);
      padding: 1.5rem;
      text-align: center;
      font-size: 0.8rem;
      color: var(--text-muted);
    }
  </style>
</head>
<body>
  <header>
    <div class="brand">
      <div class="brand-logo">⚡</div>
      <div class="brand-text">GoKS Studio</div>
      <span class="brand-badge">DevTools</span>
    </div>
    <div class="header-actions">
      <button class="btn" onclick="refreshData()">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21.5 2v6h-6M21.34 15.57a10 10 0 1 1-.57-8.38l5.67-5.67"/></svg>
        Refresh
      </button>
      <a href="/" target="_blank" class="btn btn-primary">
        Visit App →
      </a>
    </div>
  </header>

  <div class="container">
    <div class="tabs">
      <div class="tab active" onclick="switchTab('overview')">
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect width="7" height="9" x="3" y="3" rx="1"/><rect width="7" height="5" x="14" y="3" rx="1"/><rect width="7" height="9" x="14" y="12" rx="1"/><rect width="7" height="5" x="3" y="16" rx="1"/></svg>
        Overview
      </div>
      <div class="tab" onclick="switchTab('routes')">
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M4 15s1-1 4-1 5 2 8 2 4-1 4-1V3s-1 1-4 1-5-2-8-2-4 1-4 1z"/><line x1="4" x2="4" y1="22" y2="15"/></svg>
        Routes
        <span class="tab-badge" id="badge-routes">0</span>
      </div>
      <div class="tab" onclick="switchTab('database')">
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><ellipse cx="12" cy="5" rx="9" ry="3"/><path d="M3 5v14c0 1.66 4 3 9 3s9-1.34 9-3V5"/><path d="M3 12c0 1.66 4 3 9 3s9-1.34 9-3"/></svg>
        Database
        <span class="tab-badge" id="badge-tables">0</span>
      </div>
      <div class="tab" onclick="switchTab('actions')">
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"/></svg>
        Actions & RPC
      </div>
      <div class="tab" onclick="switchTab('migrations')">
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 2v20M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"/></svg>
        Migrations
      </div>
    </div>

    <!-- OVERVIEW TAB -->
    <div id="tab-overview" class="tab-content active">
      <div class="grid grid-4" style="margin-bottom: 1.5rem;">
        <div class="card">
          <div class="card-title">Discovered Routes</div>
          <div class="card-value" id="stat-routes">-</div>
          <div class="card-desc">SSR Pages and REST APIs</div>
        </div>
        <div class="card">
          <div class="card-title">Server Actions & RPC</div>
          <div class="card-value" id="stat-actions">-</div>
          <div class="card-desc">Type-safe endpoints</div>
        </div>
        <div class="card">
          <div class="card-title">Active Goroutines</div>
          <div class="card-value" id="stat-goroutines">-</div>
          <div class="card-desc">Go concurrency runtime</div>
        </div>
        <div class="card">
          <div class="card-title">Memory Allocation</div>
          <div class="card-value" id="stat-memory">-</div>
          <div class="card-desc">Heap in use</div>
        </div>
      </div>

      <div class="grid grid-2">
        <div class="card">
          <h3 style="margin-bottom: 1rem; font-size: 1.05rem;">Runtime & Framework</h3>
          <table style="font-size: 0.85rem;">
            <tr><td style="color:var(--text-muted); width: 140px;">GoKS Version</td><td id="sys-version" class="code">-</td></tr>
            <tr><td style="color:var(--text-muted);">Go Runtime</td><td id="sys-go" class="code">-</td></tr>
            <tr><td style="color:var(--text-muted);">Architecture</td><td id="sys-os" class="code">-</td></tr>
            <tr><td style="color:var(--text-muted);">Server Uptime</td><td id="sys-uptime">-</td></tr>
            <tr><td style="color:var(--text-muted);">Environment</td><td><span class="badge badge-get">DEVELOPMENT</span></td></tr>
          </table>
        </div>
        <div class="card">
          <h3 style="margin-bottom: 1rem; font-size: 1.05rem;">Quick Navigation</h3>
          <p style="font-size: 0.85rem; color: var(--text-muted); margin-bottom: 1rem;">
            GoKS Studio gives you real-time visibility into your fullstack application. All changes in your <code>app/</code> folder are compiled instantly and reflected here.
          </p>
          <div style="display:flex; gap:0.5rem; flex-wrap:wrap;">
            <button class="btn" onclick="switchTab('routes')">View Routes →</button>
            <button class="btn" onclick="switchTab('database')">Explore DB →</button>
            <button class="btn" onclick="switchTab('migrations')">View Migrations →</button>
          </div>
        </div>
      </div>
    </div>

    <!-- ROUTES TAB -->
    <div id="tab-routes" class="tab-content">
      <div class="table-container">
        <table>
          <thead>
            <tr>
              <th style="width: 100px;">Method</th>
              <th>Pattern / URL</th>
              <th style="width: 140px;">Type</th>
            </tr>
          </thead>
          <tbody id="routes-tbody">
            <tr><td colspan="3" class="empty-state">Loading routes...</td></tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- DATABASE TAB -->
    <div id="tab-database" class="tab-content">
      <div class="db-layout">
        <div class="db-sidebar">
          <h4 style="font-size: 0.75rem; text-transform: uppercase; color: var(--text-muted); margin-bottom: 0.75rem; letter-spacing: 0.05em;">Tables</h4>
          <div id="db-tables-list">
            <div style="color: var(--text-muted); font-size: 0.85rem;">Scanning SQLite database...</div>
          </div>
        </div>
        <div>
          <div style="display:flex; justify-content:space-between; align-items:center; margin-bottom: 1rem;">
            <h3 id="current-table-title" style="font-size: 1.1rem;">Select a table</h3>
            <span id="current-table-stats" class="badge badge-page" style="display:none;">0 rows</span>
          </div>
          <div class="table-container">
            <div id="db-table-content" class="empty-state">
              Select a table from the sidebar to inspect its records and schema.
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- ACTIONS & RPC TAB -->
    <div id="tab-actions" class="tab-content">
      <div class="grid grid-2">
        <div>
          <h3 style="margin-bottom: 1rem; font-size: 1.05rem;">Server Actions (pkg/action)</h3>
          <div class="table-container">
            <table>
              <thead><tr><th>Name</th><th>Action Endpoint</th></tr></thead>
              <tbody id="actions-tbody"><tr><td colspan="2" class="empty-state">No server actions registered</td></tr></tbody>
            </table>
          </div>
        </div>
        <div>
          <h3 style="margin-bottom: 1rem; font-size: 1.05rem;">RPC Methods (pkg/rpc)</h3>
          <div class="table-container">
            <table>
              <thead><tr><th>Method</th><th>RPC Endpoint</th></tr></thead>
              <tbody id="rpc-tbody"><tr><td colspan="2" class="empty-state">No RPC methods registered</td></tr></tbody>
            </table>
          </div>
        </div>
      </div>
    </div>

    <!-- MIGRATIONS TAB -->
    <div id="tab-migrations" class="tab-content">
      <div class="table-container">
        <table>
          <thead>
            <tr>
              <th>Migration File</th>
              <th style="width: 100px;">Batch</th>
              <th style="width: 120px;">Status</th>
            </tr>
          </thead>
          <tbody id="migrations-tbody">
            <tr><td colspan="3" class="empty-state">Loading migrations...</td></tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>

  <footer>
    GoKS Framework DevTools • High-Performance Fullstack Go & WebAssembly
  </footer>

  <script>
    function switchTab(name) {
      document.querySelectorAll('.tab').forEach(t => t.classList.remove('active'));
      document.querySelectorAll('.tab-content').forEach(c => c.classList.remove('active'));
      
      const tabBtn = Array.from(document.querySelectorAll('.tab')).find(t => t.textContent.toLowerCase().includes(name));
      if (tabBtn) tabBtn.classList.add('active');
      const content = document.getElementById('tab-' + name);
      if (content) content.classList.add('active');

      if (name === 'routes') loadRoutes();
      if (name === 'database') loadDatabase();
      if (name === 'actions') loadActions();
      if (name === 'migrations') loadMigrations();
    }

    function formatBytes(bytes) {
      if (!bytes) return '0 B';
      const k = 1024;
      const sizes = ['B', 'KB', 'MB', 'GB'];
      const i = Math.floor(Math.log(bytes) / Math.log(k));
      return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
    }

    function formatUptime(seconds) {
      const h = Math.floor(seconds / 3600);
      const m = Math.floor((seconds % 3600) / 60);
      const s = seconds % 60;
      if (h > 0) return h + 'h ' + m + 'm';
      if (m > 0) return m + 'm ' + s + 's';
      return s + 's';
    }

    async function loadOverview() {
      try {
        const res = await fetch('/__goks/api/overview');
        const data = await res.json();
        document.getElementById('stat-routes').textContent = data.routes_count;
        document.getElementById('stat-actions').textContent = (data.actions_count + data.rpc_count);
        document.getElementById('stat-goroutines').textContent = data.goroutines;
        document.getElementById('stat-memory').textContent = formatBytes(data.alloc_bytes);

        document.getElementById('sys-version').textContent = data.version;
        document.getElementById('sys-go').textContent = data.go_version;
        document.getElementById('sys-os').textContent = data.os + ' / ' + data.arch;
        document.getElementById('sys-uptime').textContent = formatUptime(data.uptime_sec);

        document.getElementById('badge-routes').textContent = data.routes_count;
        document.getElementById('badge-tables').textContent = data.tables_count;
      } catch (e) {
        console.error("Failed to load overview:", e);
      }
    }

    async function loadRoutes() {
      try {
        const res = await fetch('/__goks/api/routes');
        const routes = await res.json();
        const tbody = document.getElementById('routes-tbody');
        if (!routes || routes.length === 0) {
          tbody.innerHTML = '<tr><td colspan="3" class="empty-state">No routes discovered</td></tr>';
          return;
        }
        tbody.innerHTML = routes.map(r => {
          let badgeClass = 'badge-get';
          const m = r.method.toUpperCase();
          if (m === 'POST') badgeClass = 'badge-post';
          else if (m === 'PUT') badgeClass = 'badge-put';
          else if (m === 'DELETE') badgeClass = 'badge-delete';
          else if (m === 'PATCH') badgeClass = 'badge-patch';

          const typeBadge = r.type === 'API Route' ? 'badge-patch' : (r.type === 'Page' ? 'badge-page' : 'badge-pending');
          return '<tr>' +
            '<td><span class="badge ' + badgeClass + '">' + m + '</span></td>' +
            '<td><span class="code">' + r.pattern + '</span></td>' +
            '<td><span class="badge ' + typeBadge + '">' + r.type + '</span></td>' +
          '</tr>';
        }).join('');
      } catch (e) {
        console.error("Failed to load routes:", e);
      }
    }

    async function loadActions() {
      try {
        const [resAct, resRPC] = await Promise.all([
          fetch('/__goks/api/actions'),
          fetch('/__goks/api/rpc')
        ]);
        const actions = await resAct.json();
        const rpcList = await resRPC.json();

        const actTbody = document.getElementById('actions-tbody');
        if (!actions || actions.length === 0) {
          actTbody.innerHTML = '<tr><td colspan="2" class="empty-state">No server actions registered</td></tr>';
        } else {
          actTbody.innerHTML = actions.map(function(a) {
            return '<tr><td><strong>' + a.name + '</strong></td><td><span class="code">' + a.endpoint + '</span></td></tr>';
          }).join('');
        }

        const rpcTbody = document.getElementById('rpc-tbody');
        if (!rpcList || rpcList.length === 0) {
          rpcTbody.innerHTML = '<tr><td colspan="2" class="empty-state">No RPC methods registered</td></tr>';
        } else {
          rpcTbody.innerHTML = rpcList.map(function(m) {
            return '<tr><td><strong>' + m.method + '</strong></td><td><span class="code">' + m.endpoint + '</span></td></tr>';
          }).join('');
        }
      } catch (e) {
        console.error("Failed to load actions/rpc:", e);
      }
    }

    async function loadDatabase() {
      try {
        const res = await fetch('/__goks/api/db/tables');
        const tables = await res.json();
        const list = document.getElementById('db-tables-list');
        if (!tables || tables.length === 0) {
          list.innerHTML = '<div style="color:var(--text-muted); font-size:0.85rem;">No tables found in database.</div>';
          document.getElementById('db-table-content').innerHTML = '<div class="empty-state">No database found or no tables created yet.<br><small style="color:var(--text-muted);">Run migrations via <code>goks db migrate</code></small></div>';
          return;
        }

        list.innerHTML = tables.map(function(t) {
          return '<div class="table-list-item" onclick="viewTable(\'' + t + '\')"><span>' + t + '</span><span style="font-size:0.75rem; color:var(--text-muted);">→</span></div>';
        }).join('');

        if (tables.length > 0) {
          viewTable(tables[0]);
        }
      } catch (e) {
        console.error("Failed to load database tables:", e);
      }
    }

    async function viewTable(tableName) {
      document.querySelectorAll('.table-list-item').forEach(el => {
        el.classList.toggle('active', el.textContent.includes(tableName));
      });
      document.getElementById('current-table-title').textContent = tableName;
      const statsBadge = document.getElementById('current-table-stats');

      try {
        const res = await fetch('/__goks/api/db/table?name=' + encodeURIComponent(tableName));
        if (!res.ok) {
          document.getElementById('db-table-content').innerHTML = '<div class="empty-state">Failed to load table</div>';
          return;
        }
        const data = await res.json();
        statsBadge.style.display = 'inline-block';
        statsBadge.textContent = data.total + ' total rows';

        if (!data.rows || data.rows.length === 0) {
          document.getElementById('db-table-content').innerHTML = '<div class="empty-state">Table is empty</div>';
          return;
        }

        const cols = data.columns;
        let html = '<table><thead><tr>';
        cols.forEach(c => html += '<th>' + c + '</th>');
        html += '</tr></thead><tbody>';

        data.rows.forEach(r => {
          html += '<tr>';
          cols.forEach(c => {
            const val = r[c] !== null && r[c] !== undefined ? String(r[c]) : '<span style="color:var(--text-muted);">NULL</span>';
            html += '<td>' + val + '</td>';
          });
          html += '</tr>';
        });
        html += '</tbody></table>';
        document.getElementById('db-table-content').innerHTML = html;
      } catch (e) {
        console.error("Failed to view table:", e);
      }
    }

    async function loadMigrations() {
      try {
        const res = await fetch('/__goks/api/migrations');
        const migs = await res.json();
        const tbody = document.getElementById('migrations-tbody');
        if (!migs || migs.length === 0) {
          tbody.innerHTML = '<tr><td colspan="3" class="empty-state">No migrations found in migrations/</td></tr>';
          return;
        }
        tbody.innerHTML = migs.map(function(m) {
          return '<tr>' +
            '<td><span class="code">' + m.name + '</span></td>' +
            '<td>' + (m.batch ? m.batch : '-') + '</td>' +
            '<td><span class="badge ' + (m.applied ? 'badge-applied' : 'badge-pending') + '">' + (m.applied ? 'APPLIED' : 'PENDING') + '</span></td>' +
          '</tr>';
        }).join('');
      } catch (e) {
        console.error("Failed to load migrations:", e);
      }
    }

    function refreshData() {
      loadOverview();
      const active = document.querySelector('.tab-content.active');
      if (active) {
        const id = active.id.replace('tab-', '');
        if (id === 'routes') loadRoutes();
        else if (id === 'database') loadDatabase();
        else if (id === 'actions') loadActions();
        else if (id === 'migrations') loadMigrations();
      }
    }

    // Initialize
    loadOverview();
  </script>
</body>
</html>
`
