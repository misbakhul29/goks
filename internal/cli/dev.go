package cli

import (
	"fmt"
	"html"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/fatih/color"
	"github.com/misbakhul29/goks/internal/compiler"
	"github.com/misbakhul29/goks/internal/generator"
	"github.com/misbakhul29/goks/internal/livereload"
	"github.com/misbakhul29/goks/internal/watcher"
	"github.com/spf13/cobra"
)

func getFreePort(startPort int) int {
	// A simple heuristic for now
	return startPort + 1
}

// DevCmd returns the `goks dev` subcommand.
func DevCmd() *cobra.Command {
	var port int
	var appDir string

	cmd := &cobra.Command{
		Use:   "dev",
		Short: "Start the GoKS development server with hot reload",
		RunE: func(cmd *cobra.Command, args []string) error {
			if appDir == "" {
				appDir, _ = os.Getwd()
			}
			if _, err := os.Stat(filepath.Join(appDir, "go.mod")); err != nil {
				return fmt.Errorf("no go.mod found in %s — is this a GoKS project?", appDir)
			}

			fmt.Println(color.CyanString("\n  ⚡ GoKS Dev Server"))
			fmt.Printf("  %s %s\n\n", color.HiBlackString("app dir:"), appDir)

			childPort := getFreePort(port)
			var buildErrorMutex sync.RWMutex
			var buildError string

			lr := livereload.New()

			// 1. Generate & Compile Initial
			fmt.Print(color.HiBlackString("  [1/2] Compiling WASM & Server... "))
			start := time.Now()
			out, err := compileWasmAndServer(appDir)
			if err != nil {
				buildError = string(out)
				fmt.Printf("%s\n", color.RedString("error"))
				fmt.Println(color.RedString(buildError))
			} else {
				fmt.Printf("%s (%v)\n", color.GreenString("done"), time.Since(start).Round(time.Millisecond))
			}

			// 2. Start Tailwind compiler in background
			fmt.Print(color.HiBlackString("  [2/2] Compiling Tailwind CSS... "))
			start = time.Now()
			go func() {
				twCLI, err := ensureTailwindCLI()
				if err != nil {
					fmt.Println(color.RedString("Failed to download Tailwind CLI: %v", err))
					return
				}
				buildDir := filepath.Join(appDir, ".goks", "build")
				os.MkdirAll(buildDir, 0755)

				// Lakukan build awal agar file app.css dijamin ada sebelum server menyala
				initial := exec.Command(twCLI, "-i", "public/global.css", "-o", ".goks/build/app.css")
				initial.Dir = appDir
				_ = initial.Run()

				tw := exec.Command(twCLI, "-i", "public/global.css", "-o", ".goks/build/app.css", "--watch")
				tw.Dir = appDir
				_ = tw.Run()
			}()
			fmt.Printf("%s (%v)\n\n", color.GreenString("done"), time.Since(start).Round(time.Millisecond))

			// 3. Start the GoKS Child Server
			var childCmd *exec.Cmd
			startChild := func() {
				if childCmd != nil && childCmd.Process != nil {
					childCmd.Process.Kill()
					childCmd.Wait()
				}
				
				buildErrorMutex.RLock()
				hasErr := buildError != ""
				buildErrorMutex.RUnlock()
				if hasErr {
					return
				}

				childCmd = exec.Command("./goks-server")
				childCmd.Dir = filepath.Join(appDir, ".goks", "entry")
				childCmd.Env = append(os.Environ(), "GOKS_CHILD_PORT="+strconv.Itoa(childPort))
				childCmd.Stdout = os.Stdout
				childCmd.Stderr = os.Stderr
				_ = childCmd.Start()
			}
			startChild()

			// 4. File Watcher to trigger rebuilds and restarts
			w, err := watcher.New(500*time.Millisecond, func(ev watcher.Event) {
				fmt.Printf("\n%s %s\n", color.YellowString("↻ Change detected:"), ev.Path)
				start = time.Now()
				out, err := compileWasmAndServer(appDir)
				
				buildErrorMutex.Lock()
				if err != nil {
					buildError = string(out)
					fmt.Printf("%s (%v)\n", color.RedString("  ❌ Compile Error"), time.Since(start).Round(time.Millisecond))
					fmt.Println(color.RedString(buildError))
				} else {
					buildError = ""
					fmt.Printf("%s (%v)\n", color.GreenString("  ✅ Recompiled"), time.Since(start).Round(time.Millisecond))
				}
				buildErrorMutex.Unlock()
				
				if err == nil {
					fmt.Println(color.CyanString("  🚀 Restarting server..."))
				}
				startChild()
				lr.Reload()
			})
			if err == nil {
				w.Watch(filepath.Join(appDir, "app"))
				w.Start()
			}

			// 5. Start Reverse Proxy
			fmt.Println(color.CyanString("  🚀 Starting server at http://localhost:%d", port))
			childURL, _ := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", childPort))
			proxy := httputil.NewSingleHostReverseProxy(childURL)
			
			http.HandleFunc("/__goks_livereload", lr.Handler())
			http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				buildErrorMutex.RLock()
				errStr := buildError
				buildErrorMutex.RUnlock()

				if errStr != "" {
					w.Header().Set("Content-Type", "text/html; charset=utf-8")
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte(`
					<!DOCTYPE html>
					<html>
					<head>
						<title>GoKS Compilation Error</title>
						<style>
							body { font-family: monospace; background: #1e1e1e; color: #d4d4d4; padding: 2rem; }
							h1 { color: #f48771; }
							pre { background: #2d2d2d; padding: 1rem; border-radius: 4px; overflow-x: auto; }
						</style>
					</head>
					<body>
						<h1>Build Error</h1>
						<pre>` + html.EscapeString(errStr) + `</pre>
						` + livereload.Script() + `
					</body>
					</html>
					`))
					return
				}
				proxy.ServeHTTP(w, r)
			})

			server := &http.Server{Addr: fmt.Sprintf(":%d", port)}
			go func() {
				if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					log.Fatalf("Listen error: %s\n", err)
				}
			}()

			// Block until signal received
			c := make(chan os.Signal, 1)
			signal.Notify(c, os.Interrupt, syscall.SIGTERM)
			<-c

			if childCmd != nil && childCmd.Process != nil {
				childCmd.Process.Kill()
			}
			server.Close()

			return nil
		},
	}

	cmd.Flags().IntVarP(&port, "port", "p", 3000, "Port to listen on")
	cmd.Flags().StringVarP(&appDir, "dir", "d", "", "App directory (default: current dir)")
	return cmd
}

func compileWasmAndServer(appDir string) ([]byte, error) {
	// Transpile any .gox files to .go files
	_ = compiler.TranspileDir(appDir)

	// Generate router logic & go.mod for Development
	_ = generator.GenerateRouter(appDir, false)
	
	entryDir := filepath.Join(appDir, ".goks", "entry")
	
	// Run go mod tidy in entryDir
	tidyCmd := exec.Command("go", "mod", "tidy")
	tidyCmd.Dir = entryDir
	_ = tidyCmd.Run()

	// Compile WASM
	outDir := filepath.Join(appDir, ".goks", "build")
	os.MkdirAll(outDir, 0755)
	
	cmdWasm := exec.Command("go", "build", "-o", filepath.Join(outDir, "app.wasm"), ".")
	cmdWasm.Env = append(os.Environ(), "GOOS=js", "GOARCH=wasm")
	cmdWasm.Dir = entryDir
	if out, err := cmdWasm.CombinedOutput(); err != nil {
		return out, err
	}
	
	// Compile Server
	cmdServer := exec.Command("go", "build", "-o", "goks-server", ".")
	cmdServer.Dir = entryDir
	if out, err := cmdServer.CombinedOutput(); err != nil {
		return out, err
	}

	return nil, nil
}
