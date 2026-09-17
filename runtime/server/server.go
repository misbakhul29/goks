// Package server provides the GoKS development and production server runtime.
package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"mime"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/misbakhul29/goks/internal/livereload"
	"github.com/misbakhul29/goks/internal/watcher"
	"github.com/misbakhul29/goks/pkg/action"
	"github.com/misbakhul29/goks/pkg/component"
	"github.com/misbakhul29/goks/pkg/font/google"
	"github.com/misbakhul29/goks/pkg/image"
	"github.com/misbakhul29/goks/pkg/metadata"
	"github.com/misbakhul29/goks/pkg/router"
	"github.com/misbakhul29/goks/pkg/rpc"
	"github.com/misbakhul29/goks/pkg/studio"
	"github.com/misbakhul29/goks/pkg/ws"
)

var ssrMutex sync.Mutex

func init() {
	// Pastikan MIME type standar selalu terdaftar, bahkan jika OS tidak memilikinya.
	mime.AddExtensionType(".css", "text/css; charset=utf-8")
	mime.AddExtensionType(".js", "application/javascript")
	mime.AddExtensionType(".wasm", "application/wasm")
	mime.AddExtensionType(".html", "text/html; charset=utf-8")
}

// Config holds the server configuration.
type Config struct {
	Host        string                  // default "0.0.0.0"
	Port        int                     // default 3000
	AppDir      string                  // path to user's app directory
	DevMode     bool                    // enable hot reload and live reload
	Standalone  bool                    // if true, serve assets from embedded FS (standalone build)
	Root        component.Renderable    // Root component for Server-Side Rendering (SSR)
	Middlewares []router.MiddlewareFunc // User-defined global middlewares
	WSHub       *ws.Hub                 // Optional WebSocket hub whose lifecycle is managed with the server

	// Standalone mode: embedded filesystems (set by generated server_main.go)
	EmbeddedAssets fs.ReadFileFS // embeds app.wasm, app.css, wasm_exec.js
	EmbeddedPublic fs.FS         // embeds public/ directory
}

// DevServer is the GoKS development server with hot reload.
type DevServer struct {
	cfg        Config
	router     *router.Router
	lr         *livereload.Server
	wasm       string // path to compiled app.wasm
	routesOnce sync.Once
	httpServer *http.Server
	serverMu   sync.Mutex
	wsHub      *ws.Hub
}

// Server is an alias to DevServer representing the GoKS HTTP runtime server.
type Server = DevServer

// New creates a new GoKS runtime server.
func New(cfg Config) *Server {
	return NewDev(cfg)
}

// NewDev creates a new development server.
func NewDev(cfg Config) *DevServer {
	if cfg.Host == "" {
		cfg.Host = "0.0.0.0"
	}
	if cfg.Port == 0 {
		cfg.Port = 3000
	}
	if cfg.AppDir == "" {
		cfg.AppDir = "."
	}

	r := router.New()
	// Standard middleware order: RequestID injects context/headers first, then Logger, then Recover wraps downstream errors
	r.Use(router.RequestID(), router.Logger(), router.Recover())
	if len(cfg.Middlewares) > 0 {
		r.Use(cfg.Middlewares...)
	}

	return &DevServer{
		cfg:    cfg,
		router: r,
		lr:     livereload.New(),
		wsHub:  cfg.WSHub,
	}
}

// SetWSHub registers or updates the WebSocket hub associated with this server.
func (s *DevServer) SetWSHub(hub *ws.Hub) {
	s.serverMu.Lock()
	defer s.serverMu.Unlock()
	s.wsHub = hub
}

// WSHub returns the registered WebSocket hub if any.
func (s *DevServer) WSHub() *ws.Hub {
	s.serverMu.Lock()
	defer s.serverMu.Unlock()
	return s.wsHub
}

// Router returns the underlying HTTP router instance.
func (s *DevServer) Router() *router.Router {
	return s.router
}

// Shutdown gracefully stops the HTTP server and any associated WebSocket hubs.
func (s *DevServer) Shutdown(ctx context.Context) error {
	s.serverMu.Lock()
	srv := s.httpServer
	hub := s.wsHub
	s.serverMu.Unlock()

	var firstErr error
	if srv != nil {
		if err := srv.Shutdown(ctx); err != nil {
			firstErr = err
		}
	}

	if hub != nil {
		if err := hub.Shutdown(ctx); err != nil && firstErr == nil {
			firstErr = err
		}
	}

	return firstErr
}

// Start launches the dev server with file watching and live reload.
func (s *DevServer) Start() error {
	// Load .env file
	if err := godotenv.Load(filepath.Join(s.cfg.AppDir, ".env")); err == nil {
		log.Println("[GoKS] Loaded .env file")
	}

	// Set up routes
	s.setupRoutes()

	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)
	srv := &http.Server{
		Addr:    addr,
		Handler: s.router,
	}

	s.serverMu.Lock()
	s.httpServer = srv
	s.serverMu.Unlock()

	// Graceful shutdown on SIGINT/SIGTERM
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-quit
		log.Println("[GoKS] Shutting down...")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	}()

	if os.Getenv("GOKS_CHILD_PORT") == "" {
		log.Printf("[GoKS] 🚀 Server running at http://localhost:%d", s.cfg.Port)
	}

	err := srv.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

// setupRoutes wires up all built-in and user routes idempotently.
func (s *DevServer) setupRoutes() {
	s.routesOnce.Do(s.initRoutes)
}

func (s *DevServer) initRoutes() {
	// Standard Health and Readiness probes (M1.3)
	s.router.GET("/_goks/healthz", func(ctx *router.Context) error {
		return ctx.Status(http.StatusOK).JSON(map[string]any{
			"status": "ok",
			"time":   time.Now().UTC().Format(time.RFC3339),
		})
	})
	s.router.GET("/_goks/ready", func(ctx *router.Context) error {
		return ctx.Status(http.StatusOK).JSON(map[string]any{
			"status": "ready",
			"time":   time.Now().UTC().Format(time.RFC3339),
		})
	})

	// Live reload WebSocket endpoint
	if s.cfg.DevMode {
		s.router.GET("/__goks_livereload", func(ctx *router.Context) error {
			s.lr.Handler()(ctx.Response(), ctx.Request())
			return nil
		})

		// GoKS Studio DevTools Dashboard & APIs
		studioHandler := studio.New(studio.Config{
			AppDir:  s.cfg.AppDir,
			Router:  s.router,
			DevMode: true,
		})
		s.router.GET("/__goks", func(ctx *router.Context) error {
			studioHandler.ServeHTTP(ctx.Response(), ctx.Request())
			return nil
		})
		s.router.GET("/__goks/*path", func(ctx *router.Context) error {
			studioHandler.ServeHTTP(ctx.Response(), ctx.Request())
			return nil
		})
	}

	// RPC Endpoint
	s.router.POST("/__goks_rpc", func(ctx *router.Context) error {
		rpc.Handler().ServeHTTP(ctx.Response(), ctx.Request())
		return nil
	})

	// Server Actions Endpoint
	s.router.POST("/__goks_action", func(ctx *router.Context) error {
		action.Handler().ServeHTTP(ctx.Response(), ctx.Request())
		return nil
	})

	// Image Optimizer Endpoint
	var embeddedPublic http.FileSystem
	if s.cfg.EmbeddedPublic != nil {
		if subFS, err := fs.Sub(s.cfg.EmbeddedPublic, "public"); err == nil {
			embeddedPublic = http.FS(subFS)
		}
	}
	imgOpt := image.NewOptimizer(image.Config{
		AppDir:         s.cfg.AppDir,
		EmbeddedPublic: embeddedPublic,
	})
	s.router.GET("/__goks_image", func(ctx *router.Context) error {
		imgOpt.ServeHTTP(ctx.Response(), ctx.Request())
		return nil
	})

	if s.cfg.Standalone && s.cfg.EmbeddedAssets != nil {
		// ── STANDALONE MODE: serve all assets from embedded FS ──────────────────
		s.setupStandaloneRoutes()
	} else {
		// ── NORMAL MODE: serve assets from disk ─────────────────────────────────
		s.setupDiskRoutes()
	}
}

// setupStandaloneRoutes wires routes that serve assets from embedded FS.
func (s *DevServer) setupStandaloneRoutes() {
	// Serve compiled WASM binary from embed
	s.router.GET("/app.wasm", func(ctx *router.Context) error {
		data, err := s.cfg.EmbeddedAssets.ReadFile("app.wasm")
		if err != nil {
			ctx.Response().WriteHeader(http.StatusNotFound)
			return nil
		}
		ctx.Response().Header().Set("Content-Type", "application/wasm")
		ctx.Response().Write(data)
		return nil
	})

	// Serve compiled CSS from embed
	s.router.GET("/app.css", func(ctx *router.Context) error {
		data, err := s.cfg.EmbeddedAssets.ReadFile("app.css")
		if err != nil {
			ctx.Response().WriteHeader(http.StatusNotFound)
			return nil
		}
		ctx.Response().Header().Set("Content-Type", "text/css; charset=utf-8")
		ctx.Response().Write(data)
		return nil
	})

	// Serve wasm_exec.js from embed
	s.router.GET("/wasm_exec.js", func(ctx *router.Context) error {
		data, err := s.cfg.EmbeddedAssets.ReadFile("wasm_exec.js")
		if err != nil {
			ctx.Response().WriteHeader(http.StatusNotFound)
			return nil
		}
		ctx.Response().Header().Set("Content-Type", "application/javascript")
		ctx.Response().Write(data)
		return nil
	})

	// Serve public static files from embedded FS
	var publicFS http.FileSystem
	if s.cfg.EmbeddedPublic != nil {
		subFS, err := fs.Sub(s.cfg.EmbeddedPublic, "public")
		if err == nil {
			publicFS = http.FS(subFS)
		}
	}

	s.router.GET("/public/*", func(ctx *router.Context) error {
		if publicFS == nil {
			ctx.Response().WriteHeader(http.StatusNotFound)
			return nil
		}
		http.StripPrefix("/public/", http.FileServer(noDirListingFS{publicFS})).ServeHTTP(
			ctx.Response(), ctx.Request(),
		)
		return nil
	})

	// Catch-all: serve static files from embedded public/ or app shell HTML
	s.router.GET("/*", func(ctx *router.Context) error {
		reqPath := ctx.Request().URL.Path
		if reqPath != "/" && !strings.Contains(reqPath, "..") && publicFS != nil {
			// Try to open as static file from embedded public/
			f, err := publicFS.Open(reqPath)
			if err == nil {
				stat, statErr := f.Stat()
				f.Close()
				if statErr == nil && !stat.IsDir() {
					http.FileServer(publicFS).ServeHTTP(ctx.Response(), ctx.Request())
					return nil
				}
			}
		}
		return s.serveShell(ctx)
	})
	s.router.GET("/", func(ctx *router.Context) error {
		return s.serveShell(ctx)
	})
}

// setupDiskRoutes wires routes that serve assets from disk (normal/dev mode).
func (s *DevServer) setupDiskRoutes() {
	// Serve compiled WASM binary
	s.router.GET("/app.wasm", func(ctx *router.Context) error {
		wasmPath := filepath.Join(s.cfg.AppDir, ".goks", "build", "app.wasm")
		if _, err := os.Stat(wasmPath); os.IsNotExist(err) {
			wasmPath = filepath.Join(s.cfg.AppDir, "app.wasm")
		}
		http.ServeFile(ctx.Response(), ctx.Request(), wasmPath)
		return nil
	})

	// Serve compiled CSS
	s.router.GET("/app.css", func(ctx *router.Context) error {
		cssPath := filepath.Join(s.cfg.AppDir, ".goks", "build", "app.css")
		if _, err := os.Stat(cssPath); os.IsNotExist(err) {
			cssPath = filepath.Join(s.cfg.AppDir, "app.css")
		}
		ctx.Response().Header().Set("Content-Type", "text/css; charset=utf-8")
		http.ServeFile(ctx.Response(), ctx.Request(), cssPath)
		return nil
	})

	// Serve wasm_exec.js (Go WASM bootstrap)
	s.router.GET("/wasm_exec.js", func(ctx *router.Context) error {
		// In production, we expect wasm_exec.js to be in .goks/build
		prodExec := filepath.Join(s.cfg.AppDir, ".goks", "build", "wasm_exec.js")
		if _, err := os.Stat(prodExec); err == nil {
			ctx.Response().Header().Set("Content-Type", "application/javascript")
			http.ServeFile(ctx.Response(), ctx.Request(), prodExec)
			return nil
		}

		// Fallback to GOROOT for development
		goRoot := os.Getenv("GOROOT")
		if goRoot == "" {
			out, err := exec.Command("go", "env", "GOROOT").Output()
			if err == nil {
				goRoot = strings.TrimSpace(string(out))
			}
		}
		wasmExec := filepath.Join(goRoot, "misc", "wasm", "wasm_exec.js")
		if _, err := os.Stat(wasmExec); os.IsNotExist(err) {
			wasmExec = filepath.Join(goRoot, "lib", "wasm", "wasm_exec.js")
		}

		ctx.Response().Header().Set("Content-Type", "application/javascript")
		http.ServeFile(ctx.Response(), ctx.Request(), wasmExec)
		return nil
	})

	// Serve public static files (both at /public/* and root / Next.js-style)
	publicDir := filepath.Join(s.cfg.AppDir, "public")
	s.router.GET("/public/*", func(ctx *router.Context) error {
		reqPath := ctx.Request().URL.Path
		if strings.HasSuffix(reqPath, ".css") {
			ctx.Response().Header().Set("Content-Type", "text/css; charset=utf-8")
		} else if strings.HasSuffix(reqPath, ".js") {
			ctx.Response().Header().Set("Content-Type", "application/javascript")
		} else if strings.HasSuffix(reqPath, ".wasm") {
			ctx.Response().Header().Set("Content-Type", "application/wasm")
		}

		http.StripPrefix("/public/", http.FileServer(noDirListingFS{http.Dir(publicDir)})).ServeHTTP(
			ctx.Response(), ctx.Request(),
		)
		return nil
	})

	// Catch-all: serve static files from public/ or app shell HTML (SPA mode)
	s.router.GET("/*", func(ctx *router.Context) error {
		reqPath := ctx.Request().URL.Path
		if reqPath != "/" && !strings.Contains(reqPath, "..") {
			staticFile := filepath.Join(publicDir, filepath.Clean(reqPath))
			if rel, err := filepath.Rel(publicDir, staticFile); err == nil && !strings.HasPrefix(rel, "..") {
				if fi, err := os.Stat(staticFile); err == nil && !fi.IsDir() {
					http.ServeFile(ctx.Response(), ctx.Request(), staticFile)
					return nil
				}
			}
		}
		return s.serveShell(ctx)
	})
	s.router.GET("/", func(ctx *router.Context) error {
		return s.serveShell(ctx)
	})
}

// noDirListingFS wraps http.FileSystem to prevent directory listings.
type noDirListingFS struct {
	fs http.FileSystem
}

func (nfs noDirListingFS) Open(path string) (http.File, error) {
	f, err := nfs.fs.Open(path)
	if err != nil {
		return nil, err
	}
	s, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}
	if s.IsDir() {
		// If requesting directory, only allow if index.html exists, otherwise return 404
		index := filepath.Join(path, "index.html")
		if _, err := nfs.fs.Open(index); err != nil {
			f.Close()
			return nil, os.ErrNotExist
		}
	}
	return f, nil
}

// serveShell renders the HTML shell that bootstraps the WASM app.
func (s *DevServer) serveShell(ctx *router.Context) error {
	lrScript := ""
	if s.cfg.DevMode {
		lrScript = livereload.Script()
	}

	// Collect PUBLIC_ env vars
	publicEnv := make(map[string]string)
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 && strings.HasPrefix(parts[0], "PUBLIC_") {
			publicEnv[parts[0]] = parts[1]
		}
	}
	envJSON, _ := json.Marshal(publicEnv)
	envScript := fmt.Sprintf(`<script>window.__GOKS_ENV = %s;</script>`, string(envJSON))

	if s.cfg.Root != nil {
		ssrMutex.Lock()

		// Temporarily set the path for SSR
		originalPath := router.CurrentPath.Get()
		reqPath := ctx.Request().URL.Path
		router.CurrentPath.Set(reqPath)

		// Set 404 status code if the route is not matched
		if matcher, ok := s.cfg.Root.(interface{ HasMatchedPage(string) bool }); ok {
			if !matcher.HasMatchedPage(reqPath) {
				ctx.Status(http.StatusNotFound)
			}
		}

		// Expand the root component tree
		renderedNode := component.Expand(component.C(s.cfg.Root), func() {}, nil)

		var meta metadata.Metadata
		if renderedNode != nil && renderedNode.Tag == "html" {
			meta = metadata.ExtractFromTree(component.C(s.cfg.Root))
		}

		// Restore path
		router.CurrentPath.Set(originalPath)
		ssrMutex.Unlock()

		if renderedNode != nil && renderedNode.Tag == "html" {
			if meta.Title == "" {
				meta.Title = "GoKS App"
			}
			fullHTML := renderDocumentHTML(renderedNode, meta, envScript, lrScript)
			return ctx.HTML(fullHTML)
		}

		// Fallback for non-html root
		ssrContent := component.RenderToString(renderedNode)
		return ctx.HTML(shellHTML(lrScript, envScript, ssrContent))
	}

	html := shellHTML(lrScript, envScript, "")
	return ctx.HTML(html)
}

// renderDocumentHTML injects metadata, assets, and WASM runtime scripts into a root <html> element tree.
func renderDocumentHTML(htmlNode *component.Node, meta metadata.Metadata, envScript, lrScript string) string {
	var headNode *component.Node
	var bodyNode *component.Node
	var otherChildren []*component.Node

	for _, ch := range htmlNode.Children {
		if ch.Tag == "head" {
			headNode = ch
		} else if ch.Tag == "body" {
			bodyNode = ch
		} else {
			otherChildren = append(otherChildren, ch)
		}
	}

	// 1. Prepare <head>
	metaHTML := metadata.RenderHTML(meta)
	fontsHTML := google.AllFontsHeadHTML()
	headAssets := fontsHTML + `<link rel="stylesheet" href="/app.css" />` + "\n" + envScript + "\n"

	if headNode == nil {
		headNode = &component.Node{
			Type: component.NodeTypeElement,
			Tag:  "head",
			Props: component.Props{
				"innerHTML": metaHTML + headAssets,
			},
		}
	} else {
		// Preserve any custom head elements defined by the user
		existingHTML := ""
		for _, ch := range headNode.Children {
			existingHTML += component.RenderToString(ch) + "\n"
		}
		headNode.Children = nil
		headNode.Props["innerHTML"] = metaHTML + headAssets + existingHTML
	}

	// 2. Prepare <body>
	needsHydration := component.NeedsHydration(htmlNode)
	wasmBootScript := ""
	if needsHydration {
		wasmBootScript = `
  <script src="/wasm_exec.js"></script>
  <script>
    const go = new Go();
    WebAssembly.instantiateStreaming(fetch("/app.wasm"), go.importObject)
      .then(result => {
        go.run(result.instance);
      })
      .catch(err => {
        console.error("Failed to load WASM:", err);
      });
  </script>
`
	}
	wasmBootScript += lrScript

	if bodyNode == nil {
		bodyNode = &component.Node{
			Type:     component.NodeTypeElement,
			Tag:      "body",
			Children: otherChildren,
		}
		otherChildren = nil
	}

	// Wrap body's component children inside <div id="__goks">
	hasAppWrapper := false
	if len(bodyNode.Children) == 1 && (bodyNode.Children[0].Props["id"] == "app" || bodyNode.Children[0].Props["id"] == "__goks") {
		hasAppWrapper = true
	}

	var innerBodyHTML string
	if hasAppWrapper {
		innerBodyHTML = component.RenderToString(bodyNode.Children[0])
	} else {
		var sb strings.Builder
		for _, ch := range bodyNode.Children {
			sb.WriteString(component.RenderToString(ch))
		}
		innerBodyHTML = `<div id="__goks">` + sb.String() + `</div>`
	}

	bodyNode.Children = nil
	bodyNode.Props["innerHTML"] = innerBodyHTML + wasmBootScript

	// Reassemble htmlNode
	htmlNode.Children = []*component.Node{headNode, bodyNode}

	return "<!DOCTYPE html>\n" + component.RenderToString(htmlNode)
}

// compileWASM compiles the user's app to WebAssembly.
func (s *DevServer) compileWASM() error {
	outDir := filepath.Join(s.cfg.AppDir, ".goks", "build")
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return err
	}
	outPath := filepath.Join(outDir, "app.wasm")

	log.Println("[GoKS] Compiling WASM...")
	cmd := exec.Command("go", "build", "-o", outPath, "./client")
	cmd.Env = append(os.Environ(), "GOOS=js", "GOARCH=wasm")
	cmd.Dir = s.cfg.AppDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	start := time.Now()
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("WASM compile error: %w", err)
	}
	s.wasm = outPath
	log.Printf("[GoKS] ✅ WASM compiled in %v → %s", time.Since(start).Round(time.Millisecond), outPath)
	return nil
}

// startWatcher sets up the file watcher for hot reload.
func (s *DevServer) startWatcher() {
	w, err := watcher.New(200*time.Millisecond, func(ev watcher.Event) {
		// Only recompile on .go file changes
		if !strings.HasSuffix(ev.Path, ".go") {
			s.lr.Reload() // Still reload for CSS/HTML changes
			return
		}

		log.Printf("[GoKS] Change detected: %s — recompiling...", ev.Path)
		if err := s.compileWASM(); err != nil {
			log.Printf("[GoKS] ❌ Compile error: %v", err)
			return
		}
		s.lr.Reload()
	})
	if err != nil {
		log.Printf("[GoKS] Watcher init error: %v", err)
		return
	}

	// Watch app source directory
	dirs := []string{
		filepath.Join(s.cfg.AppDir, "app"),
		filepath.Join(s.cfg.AppDir, "pages"),
		filepath.Join(s.cfg.AppDir, "components"),
		filepath.Join(s.cfg.AppDir, "public"),
	}
	for _, d := range dirs {
		if _, err := os.Stat(d); err == nil {
			if err := w.Watch(d); err != nil {
				log.Printf("[GoKS] Cannot watch %s: %v", d, err)
			}
		}
	}
	w.Start()
	log.Println("[GoKS] 👀 File watcher started")
}

// shellHTML returns the HTML shell that loads and runs the WASM binary.
func shellHTML(liveReloadScript, envScript, ssrContent string) string {
	fontsHTML := google.AllFontsHeadHTML()
	return `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>GoKS App</title>
  ` + fontsHTML + `<link rel="stylesheet" href="/app.css" />
  ` + envScript + `
  <style>
    #app { min-height: 100vh; }
  </style>
</head>
<body>
  <div id="app">` + ssrContent + `</div>

  <script src="/wasm_exec.js"></script>
  <script>
    const go = new Go();
    WebAssembly.instantiateStreaming(fetch("/app.wasm"), go.importObject)
      .then(result => {
        go.run(result.instance);
      })
      .catch(err => {
        console.error("Failed to load WASM:", err);
      });
  </script>
  ` + liveReloadScript + `
</body>
</html>`
}

// ExportStatic pre-renders all discoverable pages to static HTML files and copies
// static assets into exportDir, producing a 100% self-contained static site.
func (s *DevServer) ExportStatic(exportDir string) error {
	s.setupRoutes()

	cleanExport, err := filepath.Abs(filepath.Clean(exportDir))
	if err != nil {
		return fmt.Errorf("invalid export directory path: %w", err)
	}

	if err := os.MkdirAll(cleanExport, 0755); err != nil {
		return fmt.Errorf("failed to create export directory: %w", err)
	}

	// 1. Discover all routes
	var routes []string
	if r, ok := s.cfg.Root.(interface{ PageRoutes() []string }); ok {
		routes = r.PageRoutes()
	} else {
		routes = []string{"/"}
	}

	fmt.Printf("\n  📦 Exporting %d routes to %s\n", len(routes), cleanExport)

	for _, route := range routes {
		// Skip dynamic routes with unresolved parameters (e.g. /:id) for static export
		if strings.Contains(route, "/:") || strings.Contains(route, "/_") {
			fmt.Printf("  ⚠️  Skipping dynamic route %s (requires dynamic server or static params)\n", route)
			continue
		}

		req := httptest.NewRequest("GET", route, nil)
		rec := httptest.NewRecorder()
		s.router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			return fmt.Errorf("route %s returned HTTP status %d during static export", route, rec.Code)
		}

		htmlContent := rec.Body.Bytes()

		// Write to out/<route>/index.html
		var targetHTML string
		if route == "/" {
			targetHTML = filepath.Join(cleanExport, "index.html")
		} else {
			cleanRoute := strings.TrimPrefix(route, "/")
			subDir := filepath.Join(cleanExport, cleanRoute)
			// Security: verify within cleanExport
			if !strings.HasPrefix(filepath.Clean(subDir), cleanExport+string(os.PathSeparator)) {
				return fmt.Errorf("security violation: illegal path for route %s", route)
			}
			if err := os.MkdirAll(subDir, 0755); err != nil {
				return err
			}
			targetHTML = filepath.Join(subDir, "index.html")

			// Also write clean-URL route.html (e.g., out/about.html)
			cleanHTMLPath := filepath.Clean(filepath.Join(cleanExport, cleanRoute+".html"))
			if strings.HasPrefix(cleanHTMLPath, cleanExport+string(os.PathSeparator)) {
				_ = os.WriteFile(cleanHTMLPath, htmlContent, 0644)
			}
		}

		if err := os.WriteFile(targetHTML, htmlContent, 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", targetHTML, err)
		}

		relTarget, _ := filepath.Rel(cleanExport, targetHTML)
		fmt.Printf("  ✓ %-20s → %s\n", route, relTarget)
	}

	// 2. Export 404 page
	req404 := httptest.NewRequest("GET", "/__goks_export_404_test__", nil)
	rec404 := httptest.NewRecorder()
	s.router.ServeHTTP(rec404, req404)
	_ = os.WriteFile(filepath.Join(cleanExport, "404.html"), rec404.Body.Bytes(), 0644)
	fmt.Printf("  ✓ %-20s → 404.html\n", "404 (Not Found)")

	// 3. Copy static build assets (.goks/build/app.css, app.wasm, wasm_exec.js)
	buildDir := filepath.Join(s.cfg.AppDir, ".goks", "build")
	assets := []string{"app.css", "app.wasm", "wasm_exec.js"}
	for _, asset := range assets {
		src := filepath.Join(buildDir, asset)
		if _, err := os.Stat(src); err == nil {
			dst := filepath.Join(cleanExport, asset)
			if err := copyFile(src, dst); err != nil {
				return fmt.Errorf("failed to copy %s: %w", asset, err)
			}
			fmt.Printf("  ✓ Asset: %s\n", asset)
		}
	}

	// 4. Copy public/ directory if exists
	publicDir := filepath.Join(s.cfg.AppDir, "public")
	if _, err := os.Stat(publicDir); err == nil {
		err := filepath.Walk(publicDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			rel, err := filepath.Rel(publicDir, path)
			if err != nil || rel == "." {
				return nil
			}
			// Security check: ensure target path is strictly within cleanExport
			target := filepath.Clean(filepath.Join(cleanExport, rel))
			if !strings.HasPrefix(target, cleanExport+string(os.PathSeparator)) {
				return fmt.Errorf("security violation: path traversal in public directory: %s", rel)
			}

			if info.IsDir() {
				return os.MkdirAll(target, 0755)
			}
			return copyFile(path, target)
		})
		if err != nil {
			return fmt.Errorf("failed to copy public directory: %w", err)
		}
		fmt.Printf("  ✓ Copied public/ directory\n")
	}

	fmt.Printf("\n  ✨ Static export completed successfully!\n")
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
