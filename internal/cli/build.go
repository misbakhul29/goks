package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/fatih/color"
	"github.com/misbakhulmunir/goks/internal/generator"
	"github.com/spf13/cobra"
)

// BuildCmd returns the `goks build` subcommand.
func BuildCmd() *cobra.Command {
	var outDir string

	cmd := &cobra.Command{
		Use:   "build",
		Short: "Build GoKS app for production (server binary + WASM)",
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, _ := os.Getwd()
			if outDir == "" {
				outDir = filepath.Join(cwd, "dist")
			}
			if err := os.MkdirAll(outDir, 0755); err != nil {
				return err
			}

			fmt.Println(color.CyanString("\n  🔨 GoKS Production Build"))
			
			// Generate App Router
			if err := generator.GenerateRouter(cwd); err != nil {
				return err
			}

			goksBuildDir := filepath.Join(cwd, ".goks", "build")
			os.MkdirAll(goksBuildDir, 0755)

			if err := buildWASM(cwd, goksBuildDir); err != nil {
				return err
			}
			if err := buildCSS(cwd, goksBuildDir); err != nil {
				return err
			}
			if err := buildServer(cwd, outDir); err != nil {
				return err
			}

			fmt.Println()
			fmt.Println(color.GreenString("  ✅ Build complete!"))
			fmt.Printf("  Output: %s\n", color.CyanString(outDir))
			fmt.Printf("  %s  %s\n", color.HiBlackString("server →"), filepath.Join(outDir, "server"))
			fmt.Printf("  %s  %s\n", color.HiBlackString("wasm   →"), filepath.Join(goksBuildDir, "app.wasm"))
			fmt.Printf("  %s  %s\n", color.HiBlackString("css    →"), filepath.Join(goksBuildDir, "app.css"))
			fmt.Printf("\n  Deploy: %s\n", color.CyanString("./dist/server"))
			return nil
		},
	}

	cmd.Flags().StringVarP(&outDir, "out", "o", "", "Output directory (default: ./dist)")
	return cmd
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
	
	// Run go mod tidy in entryDir first
	tidyCmd := exec.Command("go", "mod", "tidy")
	tidyCmd.Dir = entryDir
	_ = tidyCmd.Run()
	
	cmd := exec.Command("go", "build", "-o", filepath.Join(outDir, "server"), ".")
	cmd.Dir = entryDir
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("server build failed: %w", err)
	}
	fmt.Printf(" %s (%v)\n", color.GreenString("done"), time.Since(start).Round(time.Millisecond))
	return nil
}
