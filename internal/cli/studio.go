package cli

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/fatih/color"
	"github.com/misbakhul29/goks/pkg/studio"
	"github.com/spf13/cobra"
)

// StudioCmd returns the `goks studio` subcommand.
func StudioCmd() *cobra.Command {
	var port int
	var appDir string
	var noOpen bool

	cmd := &cobra.Command{
		Use:   "studio",
		Short: "Open the embedded GoKS Studio DevTools dashboard",
		Long:  "Launch an interactive dashboard to inspect routes, database tables, server actions, RPC endpoints, and migrations.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if appDir == "" {
				appDir, _ = os.Getwd()
			}

			if _, err := os.Stat(filepath.Join(appDir, "go.mod")); err != nil {
				return fmt.Errorf("no go.mod found in %s — is this a GoKS project?", appDir)
			}

			if port == 0 {
				port = 4983
			}

			// Verify if port is available; if not, pick an ephemeral port
			l, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
			if err != nil {
				l, err = net.Listen("tcp", "127.0.0.1:0")
				if err != nil {
					return fmt.Errorf("failed to bind port: %w", err)
				}
				port = l.Addr().(*net.TCPAddr).Port
			}
			_ = l.Close()

			studioHandler := studio.New(studio.Config{
				AppDir:  appDir,
				DevMode: true,
			})

			mux := http.NewServeMux()
			mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/" {
					http.Redirect(w, r, "/__goks", http.StatusTemporaryRedirect)
					return
				}
				studioHandler.ServeHTTP(w, r)
			})

			server := &http.Server{
				Addr:    fmt.Sprintf("127.0.0.1:%d", port),
				Handler: mux,
			}

			studioURL := fmt.Sprintf("http://127.0.0.1:%d/__goks", port)

			fmt.Println(color.CyanString("\n  ⚡ GoKS Studio"))
			fmt.Printf("  %s %s\n", color.HiBlackString("App dir:"), appDir)
			fmt.Printf("  %s %s\n\n", color.GreenString("Dashboard:"), color.CyanString(studioURL))

			if !noOpen {
				go openBrowser(studioURL)
			}

			// Handle graceful shutdown
			stop := make(chan os.Signal, 1)
			signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

			go func() {
				<-stop
				fmt.Println(color.YellowString("\n  Shutting down GoKS Studio..."))
				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()
				_ = server.Shutdown(ctx)
			}()

			fmt.Println(color.HiBlackString("  Press Ctrl+C to exit.\n"))
			if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				return err
			}
			return nil
		},
	}

	cmd.Flags().IntVarP(&port, "port", "p", 4983, "Port to run GoKS Studio on")
	cmd.Flags().StringVarP(&appDir, "dir", "d", "", "Path to GoKS application directory (defaults to current directory)")
	cmd.Flags().BoolVar(&noOpen, "no-open", false, "Do not automatically open the browser")

	return cmd
}

func openBrowser(targetURL string) {
	time.Sleep(200 * time.Millisecond)
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", targetURL)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", targetURL)
	default:
		cmd = exec.Command("xdg-open", targetURL)
	}
	_ = cmd.Start()
}
