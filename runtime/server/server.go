// Package server provides the GoKS development and production server runtime.
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"mime"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/misbakhul29/goks/internal/livereload"
	"github.com/misbakhul29/goks/internal/watcher"
	"github.com/misbakhul29/goks/pkg/component"
	"github.com/misbakhul29/goks/pkg/router"
	"github.com/misbakhul29/goks/pkg/rpc"
)

func init() {
	// Pastikan MIME type standar selalu terdaftar, bahkan jika OS tidak memilikinya.
	mime.AddExtensionType(".css", "text/css; charset=utf-8")
	mime.AddExtensionType(".js", "application/javascript")
	mime.AddExtensionType(".wasm", "application/wasm")
	mime.AddExtensionType(".html", "text/html; charset=utf-8")
}

// Config holds the server configuration.
type Config struct {
	Host    string               // default "0.0.0.0"
	Port        int                  // default 3000
	AppDir      string               // path to user's app directory
	DevMode     bool                 // enable hot reload and live reload
	Root        component.Renderable // Root component for Server-Side Rendering (SSR)
	Middlewares []router.MiddlewareFunc  // User-defined global middlewares
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

	log.Printf("[GoKS] 🚀 Dev server running at http://localhost:%d", s.cfg.Port)
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

	// Serve public static files
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

	// Catch-all: serve the app shell HTML (SPA mode)
	s.router.GET("/*", func(ctx *router.Context) error {
		return s.serveShell(ctx)
	})
	s.router.GET("/", func(ctx *router.Context) error {
		return s.serveShell(ctx)
	})
}

// serveShell renders the HTML shell that bootstraps the WASM app.
func (s *DevServer) serveShell(ctx *router.Context) error {
	var ssrContent string
	if s.cfg.Root != nil {
		node := s.cfg.Root.Render()
		ssrContent = component.RenderToString(node)
	}

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

	html := shellHTML(lrScript, envScript, ssrContent)
	return ctx.HTML(html)
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
	return `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>GoKS App</title>
  <link rel="stylesheet" href="/app.css" />
  ` + envScript + `
  <style>
    #app { min-height: 100vh; }
    #goks-loading {
      position: fixed; inset: 0;
      display: flex; align-items: center; justify-content: center;
      background: #0f0f0f; color: #fff; font-size: 1rem;
      gap: 12px; z-index: 9999;
    }
    .spinner {
      width: 20px; height: 20px;
      border: 2px solid rgba(255,255,255,0.2);
      border-top-color: #fff;
      border-radius: 50%;
      animation: spin 0.6s linear infinite;
    }
    @keyframes spin { to { transform: rotate(360deg); } }
  </style>
</head>
<body>
  <div id="goks-loading">
    <div class="spinner"></div>
    <span>Loading GoKS app...</span>
  </div>
  <div id="app">` + ssrContent + `</div>

  <script src="/wasm_exec.js"></script>
  <script>
    const go = new Go();
    WebAssembly.instantiateStreaming(fetch("/app.wasm"), go.importObject)
      .then(result => {
        document.getElementById("goks-loading").style.display = "none";
        go.run(result.instance);
      })
      .catch(err => {
        document.getElementById("goks-loading").innerHTML =
          '<span style="color:#ff6b6b">❌ Failed to load WASM: ' + err.message + '</span>';
      });
  </script>
  ` + liveReloadScript + `
</body>
</html>`
}
