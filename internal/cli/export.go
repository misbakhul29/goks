package cli

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/fatih/color"
	"github.com/misbakhul29/goks/internal/compiler"
	"github.com/misbakhul29/goks/internal/generator"
	"github.com/spf13/cobra"
)

// ExportCmd returns the `goks export` subcommand for Static Site Generation (SSG).
func ExportCmd() *cobra.Command {
	var appDir string
	var outDir string
	var compilerType string
	var serve bool
	var port int

	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export the application as a 100% static website (SSG)",
		Long:  "Crawls and pre-renders all discoverable pages to static HTML, bundles assets, and outputs a deployable static site to out/ (perfect for GitHub Pages, Cloudflare Pages, Netlify, Vercel Static, S3).",
		RunE: func(cmd *cobra.Command, args []string) error {
			if appDir == "" {
				appDir, _ = os.Getwd()
			}
			if _, err := os.Stat(filepath.Join(appDir, "go.mod")); err != nil {
				return fmt.Errorf("no go.mod found in %s — is this a GoKS project?", appDir)
			}
			if _, err := os.Stat(filepath.Join(appDir, "app")); err != nil {
				return fmt.Errorf("folder 'app/' not found in %s — make sure you are in a GoKS project root", appDir)
			}

			compilerType = strings.ToLower(strings.TrimSpace(compilerType))
			if compilerType != "go" && compilerType != "tinygo" {
				return fmt.Errorf("invalid --compiler flag %q: supported values are 'go' (default) and 'tinygo'", compilerType)
			}

			// Security validation for outDir: prevent overwriting root or system directories
			absOut, err := filepath.Abs(filepath.Clean(outDir))
			if err != nil {
				return fmt.Errorf("invalid output directory path: %w", err)
			}
			cleanSlash := filepath.ToSlash(absOut)
			if cleanSlash == "/" || cleanSlash == "/bin" || cleanSlash == "/etc" || cleanSlash == "/usr" || cleanSlash == "/var" {
				return fmt.Errorf("security violation: cannot export into protected system directory %s", absOut)
			}

			// Collision protection: ensure outDir is not the app source or .goks directory
			absApp, _ := filepath.Abs(appDir)
			if absOut == absApp {
				return fmt.Errorf("safety violation: export directory cannot be the project root directory")
			}
			if absOut == filepath.Join(absApp, "app") {
				return fmt.Errorf("safety violation: export directory cannot be the 'app/' source directory")
			}
			if absOut == filepath.Join(absApp, ".goks") {
				return fmt.Errorf("safety violation: export directory cannot be the '.goks/' workspace directory")
			}

			fmt.Println(color.CyanString("\n  ⚡ GoKS Static Site Export (SSG)"))
			fmt.Printf("  %s %s\n", color.HiBlackString("app dir:"), appDir)
			fmt.Printf("  %s  %s\n\n", color.HiBlackString("output: "), absOut)

			// 1. Prepare workspace & transpile .gox files
			if err := compiler.PrepareWorkspace(appDir); err != nil {
				return err
			}

			// 2. Generate Router for Production
			if err := generator.GenerateRouter(appDir, true); err != nil {
				return err
			}

			entryDir := filepath.Join(appDir, ".goks", "entry")
			goksBuildDir := filepath.Join(appDir, ".goks", "build")
			_ = os.MkdirAll(goksBuildDir, 0755)

			// 3. Ensure dependencies are tidy
			tidyCmd := exec.Command("go", "mod", "tidy")
			tidyCmd.Dir = entryDir
			_ = tidyCmd.Run()

			// 4. Build WASM & Tailwind CSS
			if err := buildWASM(appDir, goksBuildDir, compilerType); err != nil {
				return err
			}
			if err := buildCSS(appDir, goksBuildDir); err != nil {
				return err
			}

			// 5. Copy wasm_exec.js
			wasmExec, err := locateWasmExec(compilerType)
			if err != nil {
				return err
			}
			if err := copyFile(wasmExec, filepath.Join(goksBuildDir, "wasm_exec.js")); err != nil {
				return fmt.Errorf("failed to copy wasm_exec.js: %w", err)
			}

			// 6. Compile export server binary
			fmt.Print("  [2/3] Compiling static generator...")
			start := time.Now()
			exportServerBin := filepath.Join(goksBuildDir, "goks-export-server")
			cmdServer := exec.Command("go", "build", "-o", exportServerBin, ".")
			cmdServer.Dir = entryDir
			if out, err := cmdServer.CombinedOutput(); err != nil {
				return fmt.Errorf("failed to build export generator: %s (%w)", string(out), err)
			}
			fmt.Printf(" %s (%v)\n", color.GreenString("done"), time.Since(start).Round(time.Millisecond))

			// 7. Run export generator
			fmt.Println("  [3/3] Pre-rendering pages to static HTML...")
			cmdExport := exec.Command(exportServerBin)
			cmdExport.Dir = entryDir
			cmdExport.Env = append(os.Environ(), "GOKS_EXPORT_DIR="+absOut)
			cmdExport.Stdout = os.Stdout
			cmdExport.Stderr = os.Stderr
			if err := cmdExport.Run(); err != nil {
				return fmt.Errorf("export execution failed: %w", err)
			}

			// Copy CSS and WASM assets
			_ = copyFile(filepath.Join(goksBuildDir, "app.css"), filepath.Join(absOut, "app.css"))
			_ = copyFile(filepath.Join(goksBuildDir, "app.wasm"), filepath.Join(absOut, "app.wasm"))
			_ = copyFile(filepath.Join(goksBuildDir, "wasm_exec.js"), filepath.Join(absOut, "wasm_exec.js"))

			// Copy public static files
			publicDir := filepath.Join(appDir, "public")
			if _, err := os.Stat(publicDir); err == nil {
				_ = filepath.Walk(publicDir, func(path string, info os.FileInfo, err error) error {
					if err != nil || info.IsDir() {
						return nil
					}
					rel, _ := filepath.Rel(publicDir, path)
					dest := filepath.Join(absOut, rel)
					_ = os.MkdirAll(filepath.Dir(dest), 0755)
					_ = copyFile(path, dest)
					return nil
				})
			}

			fmt.Println(color.GreenString("\n  🎉 Export successful! Ready for static deployment."))
			fmt.Printf("  %s %s\n", color.HiBlackString("Folder:"), absOut)
			fmt.Printf("  %s %s\n", color.HiBlackString("Deploy:"), "Upload this folder to Cloudflare Pages, Netlify, Vercel, or GitHub Pages.")

			// 8. If --serve is specified, start preview server
			if serve {
				return serveStaticExport(absOut, port)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&outDir, "out", "o", "out", "Output directory for static HTML & assets")
	cmd.Flags().StringVarP(&appDir, "dir", "d", "", "Application directory (default: current dir)")
	cmd.Flags().StringVar(&compilerType, "compiler", "go", "WASM compiler to use: 'go' (default) or 'tinygo'")
	cmd.Flags().BoolVarP(&serve, "serve", "s", false, "Preview static site immediately with a local HTTP server")
	cmd.Flags().IntVarP(&port, "port", "p", 3000, "Port for preview server (used with --serve)")

	return cmd
}

// serveStaticExport starts a secure local static HTTP file server for previewing.
func serveStaticExport(dir string, port int) error {
	addr := fmt.Sprintf(":%d", port)
	cleanDir := filepath.Clean(dir)
	fileServer := http.FileServer(http.Dir(cleanDir))

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Clean URL support: if /about requested, try /about.html or /about/index.html
		urlPath := filepath.Clean(r.URL.Path)
		if urlPath != "/" && !strings.Contains(urlPath, "..") && !strings.Contains(urlPath, ".") {
			candidateHTML := filepath.Clean(filepath.Join(cleanDir, urlPath+".html"))
			if strings.HasPrefix(candidateHTML, cleanDir+string(os.PathSeparator)) {
				if _, err := os.Stat(candidateHTML); err == nil {
					http.ServeFile(w, r, candidateHTML)
					return
				}
			}
		}
		fileServer.ServeHTTP(w, r)
	})

	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println(color.YellowString("\n  Stopping preview server..."))
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
		os.Exit(0)
	}()

	fmt.Println(color.CyanString(fmt.Sprintf("\n  🚀 Preview server running at http://localhost:%d", port)))
	fmt.Println(color.HiBlackString("  Press Ctrl+C to stop.\n"))

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}
