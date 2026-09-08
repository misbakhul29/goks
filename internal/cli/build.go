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
	cmd := &cobra.Command{
		Use:   "build",
		Short: "Build GoKS app for production (server binary + WASM)",
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, _ := os.Getwd()

			fmt.Println(color.CyanString("\n  🔨 GoKS Production Build"))
			
			// Transpile .gox files to .go files before generating router
			if err := compiler.TranspileDir(cwd); err != nil {
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

			if err := buildWASM(cwd, goksBuildDir); err != nil {
				return err
			}
			if err := buildCSS(cwd, goksBuildDir); err != nil {
				return err
			}
			if err := buildServer(cwd, goksBuildDir); err != nil {
				return err
			}

			// Copy wasm_exec.js into .goks/build for production
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
			copyFile(wasmExec, filepath.Join(goksBuildDir, "wasm_exec.js"))

			fmt.Println()
			fmt.Println(color.GreenString("  ✅ Build complete!"))
			fmt.Printf("  %s  %s\n", color.HiBlackString("server    →"), filepath.Join(goksBuildDir, "server"))
			fmt.Printf("  %s  %s\n", color.HiBlackString("wasm      →"), filepath.Join(goksBuildDir, "app.wasm"))
			fmt.Printf("  %s  %s\n", color.HiBlackString("css       →"), filepath.Join(goksBuildDir, "app.css"))
			fmt.Printf("\n  Deploy: %s\n", color.CyanString("goks start [port]"))
			return nil
		},
	}

	return cmd
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

func buildWASM(cwd, outDir string) error {
	fmt.Print("  [1/3] Compiling WASM frontend...")
	start := time.Now()
	entryDir := filepath.Join(cwd, ".goks", "entry")
	cmd := exec.Command("go", "build", "-o", filepath.Join(outDir, "app.wasm"), ".")
	cmd.Dir = entryDir
	cmd.Env = append(os.Environ(), "GOOS=js", "GOARCH=wasm")
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("WASM build failed: %w", err)
	}
	fmt.Printf(" %s (%v)\n", color.GreenString("done"), time.Since(start).Round(time.Millisecond))
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
