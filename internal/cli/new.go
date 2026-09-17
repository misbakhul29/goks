package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/misbakhul29/goks/internal/version"
)

var moduleFlag string

// NewCmd returns the `goks new` subcommand.
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "new <app-name>",
		Short: "Create a new GoKS application",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return scaffoldApp(args[0], moduleFlag)
		},
	}
	cmd.Flags().StringVarP(&moduleFlag, "module", "m", "", "Go module path for the new application")
	return cmd
}

func scaffoldApp(name string, modPath string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("app name cannot be empty")
	}
	if strings.Contains(name, "..") || strings.ContainsAny(name, `/\:*?"<>|`) {
		return fmt.Errorf("invalid app name '%s': must not contain path separators or traversal characters", name)
	}

	slug := strings.ToLower(strings.ReplaceAll(name, " ", "-"))
	if slug == "" || slug == "." {
		return fmt.Errorf("invalid app name '%s'", name)
	}

	appDir := filepath.Join(".", slug)
	if _, err := os.Stat(appDir); err == nil {
		return fmt.Errorf("directory '%s' already exists", slug)
	}

	fmt.Println(color.CyanString("\n  🚀 Creating new GoKS app:"), color.WhiteString(slug))

	dirs := []string{
		"app", "app/components",
		"components", "database", "database/models", "database/repositories",
		"database/migrations", "database/seeds",
		"services", "middleware", "api", "public",
	}
	for _, d := range dirs {
		path := filepath.Join(appDir, d)
		if err := os.MkdirAll(path, 0755); err != nil {
			return fmt.Errorf("mkdir %s: %w", path, err)
		}
		color.HiBlack("  create  %s/", filepath.Join(slug, d))
	}

	if modPath == "" {
		modPath = "github.com/user/" + slug
	}

	data := map[string]string{
		"AppName": name,
		"Module":  modPath,
		"Version": getGoKSVersion(),
	}

	files := map[string]string{
		"go.mod":                             tmplGoMod,
		"app/layout.gox":                     tmplLayout,
		"app/page.gox":                       tmplPage,
		"app/components/hero.gox":            tmplHeroComponent,
		"components/button.gox":              tmplExampleComponent,
		"database/models/user.go":            tmplExampleModel,
		"services/user_service.go":           tmplExampleService,
		"database/repositories/user_repo.go": tmplExampleRepository,
		"api/routes.go":                      tmplExampleAPI,
		"config/goks.config.go":              tmplConfig,
		"middleware/logger.go":               tmplMiddleware,
		".gitignore":                         tmplGitignore,
		"README.md":                          tmplReadme,
		"public/global.css":                  tmplGlobalCss,
		"public/favicon.ico":                 "", // empty placeholder
	}

	for relPath, tmplStr := range files {
		absPath := filepath.Join(appDir, relPath)
		if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
			return err
		}
		if tmplStr == "" {
			if err := os.WriteFile(absPath, nil, 0644); err != nil {
				return err
			}
		} else {
			if err := renderTemplate(absPath, relPath, tmplStr, data); err != nil {
				return err
			}
		}
		color.HiBlack("  create  %s", filepath.Join(slug, relPath))
	}

	fmt.Println()
	fmt.Println(color.GreenString("  ✅ Done!"))
	fmt.Println()
	fmt.Println("  Next steps:")
	fmt.Println(color.CyanString("    cd " + slug))
	fmt.Println(color.CyanString("    go mod tidy"))
	fmt.Println(color.CyanString("    goks dev"))
	fmt.Println()
	return nil
}

func renderTemplate(dst, name, tmplStr string, data any) error {
	t, err := template.New(name).Parse(tmplStr)
	if err != nil {
		return err
	}
	f, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer f.Close()
	return t.Execute(f, data)
}

func getGoKSVersion() string {
	return version.Current()
}
