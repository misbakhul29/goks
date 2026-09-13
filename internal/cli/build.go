package cli

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/misbakhul29/goks/internal/compiler"
	"github.com/misbakhul29/goks/internal/generator"
	"github.com/spf13/cobra"
)

// BuildCmd returns the `goks build` subcommand.
func BuildCmd() *cobra.Command {
	var standalone bool
	var compilerType string

	cmd := &cobra.Command{
		Use:   "build",
		Short: "Build GoKS app for production (server binary + WASM)",
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, _ := os.Getwd()

			if _, err := os.Stat(filepath.Join(cwd, "app")); err != nil {
				return fmt.Errorf("folder 'app/' tidak ditemukan di %s — pastikan kamu berada di dalam folder project GoKS sebelum menjalankan 'goks build'", cwd)
			}

			if standalone {
				fmt.Println(color.CyanString("\n  📦 GoKS Standalone Build"))
			} else {
				fmt.Println(color.CyanString("\n  🔨 GoKS Production Build"))
			}

			// Prepare workspace & transpile .gox files into .goks/workspace
			if err := compiler.PrepareWorkspace(cwd); err != nil {
				return err
			}

			// Generate App Router for Production
			if err := generator.GenerateRouter(cwd, true); err != nil {
				return err
			}

			goksBuildDir := filepath.Join(cwd, ".goks", "build")
			entryDir := filepath.Join(cwd, ".goks", "entry")
			os.MkdirAll(goksBuildDir, 0755)

			// Ensure dependencies are downloaded before compiling anything
			tidyCmd := exec.Command("go", "mod", "tidy")
			tidyCmd.Dir = entryDir
			_ = tidyCmd.Run()

			if err := buildWASM(cwd, goksBuildDir, compilerType); err != nil {
				return err
			}
			if err := buildCSS(cwd, goksBuildDir); err != nil {
				return err
			}

			// Copy wasm_exec.js into .goks/build for production
			wasmExec, err := locateWasmExec(compilerType)
			if err != nil {
				return err
			}
			copyFile(wasmExec, filepath.Join(goksBuildDir, "wasm_exec.js"))

			if standalone {
				// Build standalone binary with embedded assets
				if err := buildStandalone(cwd, goksBuildDir, wasmExec); err != nil {
					return err
				}
			} else {
				if err := buildServer(cwd, goksBuildDir); err != nil {
					return err
				}
			}

			fmt.Println()
			if standalone {
				fmt.Println(color.GreenString("  ✅ Standalone build complete!"))
				fmt.Printf("  %s  %s\n", color.HiBlackString("server    →"), filepath.Join(cwd, ".goks", "standalone", "server"))
				fmt.Printf("\n  Run anywhere: %s\n", color.CyanString(".goks/standalone/server"))
				fmt.Printf("  With port:    %s\n", color.CyanString("PORT=8080 .goks/standalone/server"))
			} else {
				fmt.Println(color.GreenString("  ✅ Build complete!"))
				fmt.Printf("  %s  %s\n", color.HiBlackString("server    →"), filepath.Join(goksBuildDir, "server"))
				fmt.Printf("  %s  %s\n", color.HiBlackString("wasm      →"), filepath.Join(goksBuildDir, "app.wasm"))
				fmt.Printf("  %s  %s\n", color.HiBlackString("css       →"), filepath.Join(goksBuildDir, "app.css"))
				fmt.Printf("\n  Deploy: %s\n", color.CyanString("goks start [port]"))
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&standalone, "standalone", false, "Build a self-contained binary with all assets embedded (like Next.js output: standalone)")
	cmd.Flags().StringVar(&compilerType, "compiler", "go", "WASM compiler to use: 'go' (default) or 'tinygo' (ultra-compact <300KB WASM)")
	return cmd
}

func locateWasmExec(compilerType string) (string, error) {
	if compilerType == "tinygo" {
		out, err := exec.Command("tinygo", "env", "TINYGOROOT").Output()
		if err == nil {
			tinyRoot := strings.TrimSpace(string(out))
			candidate := filepath.Join(tinyRoot, "targets", "wasm_exec.js")
			if _, err := os.Stat(candidate); err == nil {
				return candidate, nil
			}
		}
	}

	goRoot := os.Getenv("GOROOT")
	if goRoot == "" {
		if out, err := exec.Command("go", "env", "GOROOT").Output(); err == nil {
			goRoot = strings.TrimSpace(string(out))
		}
	}
	wasmExec := filepath.Join(goRoot, "misc", "wasm", "wasm_exec.js")
	if _, err := os.Stat(wasmExec); os.IsNotExist(err) {
		wasmExec = filepath.Join(goRoot, "lib", "wasm", "wasm_exec.js")
	}
	if _, err := os.Stat(wasmExec); err == nil {
		return wasmExec, nil
	}
	return "", fmt.Errorf("wasm_exec.js not found in GOROOT: %s", goRoot)
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

func buildWASM(cwd, outDir, compilerType string) error {
	compilerLabel := "Go"
	if compilerType == "tinygo" {
		compilerLabel = "TinyGo"
	}
	fmt.Printf("  [1/3] Compiling WASM frontend (%s)...", compilerLabel)
	start := time.Now()
	entryDir := filepath.Join(cwd, ".goks", "entry")
	outPath := filepath.Join(outDir, "app.wasm")

	var cmd *exec.Cmd
	if compilerType == "tinygo" {
		if _, err := exec.LookPath("tinygo"); err != nil {
			return fmt.Errorf("tinygo not found in PATH. Install TinyGo from https://tinygo.org/getting-started/install/ or omit --compiler=tinygo")
		}
		cmd = exec.Command("tinygo", "build", "-o", outPath, "-target=wasm", "-no-debug", ".")
	} else {
		cmd = exec.Command("go", "build", "-o", outPath, ".")
		cmd.Env = append(os.Environ(), "GOOS=js", "GOARCH=wasm")
	}

	cmd.Dir = entryDir
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("WASM build failed: %w", err)
	}

	sizeStr := ""
	if fi, err := os.Stat(outPath); err == nil {
		sizeKB := float64(fi.Size()) / 1024.0
		if sizeKB > 1024 {
			sizeStr = fmt.Sprintf(" — %.1f MB", sizeKB/1024.0)
		} else {
			sizeStr = fmt.Sprintf(" — %.0f KB", sizeKB)
		}
	}

	fmt.Printf(" %s (%v%s)\n", color.GreenString("done"), time.Since(start).Round(time.Millisecond), sizeStr)
	return nil
}

func buildCSS(cwd, buildDir string) error {
	fmt.Print("  [2/3] Compiling Tailwind CSS...")
	start := time.Now()

	twCLI, err := ensureTailwindCLI()
	if err != nil {
		fmt.Printf(" %s\n", color.YellowString("skipped (download failed: %v)", err))
		return nil
	}

	cmd := exec.Command(twCLI, "-i", "public/global.css", "-o", filepath.Join(buildDir, "app.css"), "--minify")
	cmd.Dir = cwd
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Printf(" %s\n", color.YellowString("skipped (tailwind build failed)"))
		return nil // We don't fail the build if tailwind is not installed globally
	}
	fmt.Printf(" %s (%v)\n", color.GreenString("done"), time.Since(start).Round(time.Millisecond))
	return nil
}

func buildServer(cwd, outDir string) error {
	fmt.Print("  [3/3] Building server binary...")
	start := time.Now()
	entryDir := filepath.Join(cwd, ".goks", "entry")
	cmd := exec.Command("go", "build", "-o", filepath.Join(outDir, "server"), ".")
	cmd.Dir = entryDir
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("server build failed: %w", err)
	}
	fmt.Printf(" %s (%v)\n", color.GreenString("done"), time.Since(start).Round(time.Millisecond))
	return nil
}

// buildStandalone creates a self-contained server binary with all assets embedded.
// It copies compiled assets into the entry dir so Go's //go:embed can pick them up,
// regenerates server_main.go with embed directives, compiles the binary, then
// cleans up the temporary copies.
func buildStandalone(cwd, buildDir, wasmExecPath string) error {
	entryDir := filepath.Join(cwd, ".goks", "entry")
	standaloneDir := filepath.Join(cwd, ".goks", "standalone")

	if err := os.MkdirAll(standaloneDir, 0755); err != nil {
		return fmt.Errorf("failed to create standalone dir: %w", err)
	}

	// ── Step 1: Copy assets into .goks/entry/ so //go:embed works ────────────
	fmt.Print("  [3/4] Preparing embedded assets...")
	assetsToCopy := map[string]string{
		filepath.Join(buildDir, "app.wasm"): filepath.Join(entryDir, "app.wasm"),
		filepath.Join(buildDir, "app.css"):  filepath.Join(entryDir, "app.css"),
		wasmExecPath:                        filepath.Join(entryDir, "wasm_exec.js"),
	}
	for src, dst := range assetsToCopy {
		if err := copyFile(src, dst); err != nil {
			fmt.Printf(" %s\n", color.YellowString("warning: could not copy %s: %v", filepath.Base(src), err))
		}
	}

	publicSrc := filepath.Join(cwd, "public")
	publicDst := filepath.Join(entryDir, "public")

	defer func() {
		for _, dst := range assetsToCopy {
			os.Remove(dst)
		}
		os.RemoveAll(publicDst)
	}()

	// Copy public/ directory into entry dir so it can be embedded
	hasPublic := false
	if _, err := os.Stat(publicSrc); err == nil {
		hasPublic = true
		if err := copyDir(publicSrc, publicDst); err != nil {
			fmt.Printf(" %s\n", color.YellowString("warning: could not copy public/: %v", err))
			hasPublic = false
		}
	}
	_ = hasPublic
	fmt.Printf(" %s\n", color.GreenString("done"))

	// ── Step 2: Regenerate server_main.go with //go:embed directives ─────────
	fmt.Print("  [4/4] Building standalone binary...")
	start := time.Now()

	if err := generator.GenerateStandaloneRouter(cwd); err != nil {
		return fmt.Errorf("failed to generate standalone router: %w", err)
	}

	// Ensure entry module is up to date
	tidyCmd := exec.Command("go", "mod", "tidy")
	tidyCmd.Dir = entryDir
	_ = tidyCmd.Run()

	// Build the standalone server binary
	cmd := exec.Command("go", "build", "-o", filepath.Join(standaloneDir, "server"), ".")
	cmd.Dir = entryDir
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	buildErr := cmd.Run()

	if buildErr != nil {
		return fmt.Errorf("standalone build failed: %w", buildErr)
	}
	fmt.Printf(" %s (%v)\n", color.GreenString("done"), time.Since(start).Round(time.Millisecond))
	return nil
}

// copyDir recursively copies a directory tree from src to dst.
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}
		return copyFile(path, target)
	})
}
