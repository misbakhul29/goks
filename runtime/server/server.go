// Package server provides the GoKS development and production server runtime.
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"mime"
	"net/http"
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
	"github.com/misbakhul29/goks/pkg/component"
	"github.com/misbakhul29/goks/pkg/font/google"
	"github.com/misbakhul29/goks/pkg/metadata"
	"github.com/misbakhul29/goks/pkg/router"
	"github.com/misbakhul29/goks/pkg/rpc"
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

	// Standalone mode: embedded filesystems (set by generated server_main.go)
	EmbeddedAssets fs.ReadFileFS // embeds app.wasm, app.css, wasm_exec.js
	EmbeddedPublic fs.FS        // embeds public/ directory
}

// DevServer is the GoKS development server with hot reload.
type DevServer struct {
	cfg    Config
	router *router.Router
	lr     *livereload.Server
	wasm   string // path to compiled app.wasm
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
	r.Use(router.Logger(), router.Recover())
	if len(cfg.Middlewares) > 0 {
		r.Use(cfg.Middlewares...)
	}

	return &DevServer{
		cfg:    cfg,
		router: r,
		lr:     livereload.New(),
	}
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

	// Graceful shutdown on SIGINT/SIGTERM
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-quit
		log.Println("[GoKS] Shutting down...")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv.Shutdown(ctx)
	}()

	if os.Getenv("GOKS_CHILD_PORT") == "" {
		log.Printf("[GoKS] 🚀 Server running at http://localhost:%d", s.cfg.Port)
	}

	return srv.ListenAndServe()
}

// setupRoutes wires up all built-in and user routes.
func (s *DevServer) setupRoutes() {
	// Live reload WebSocket endpoint
	if s.cfg.DevMode {
		s.router.GET("/__goks_livereload", func(ctx *router.Context) error {
			s.lr.Handler()(ctx.Response(), ctx.Request())
			return nil
		})
	}

	// RPC Endpoint
	s.router.POST("/__goks_rpc", func(ctx *router.Context) error {
		rpc.Handler().ServeHTTP(ctx.Response(), ctx.Request())
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
		http.StripPrefix("/public/", http.FileServer(publicFS)).ServeHTTP(
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

		http.StripPrefix("/public/", http.FileServer(http.Dir(publicDir))).ServeHTTP(
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

		// Restore path
		router.CurrentPath.Set(originalPath)
		ssrMutex.Unlock()


		if renderedNode != nil && renderedNode.Tag == "html" {
			meta := metadata.ExtractFromTree(component.C(s.cfg.Root))
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
	wasmBootScript := `
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
  ` + lrScript

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
