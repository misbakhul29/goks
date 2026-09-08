package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// GenerateCmd returns the `goks generate` subcommand with sub-sub-commands.
func GenerateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "generate",
		Short:   "Generate GoKS boilerplate (model, page, component)",
		Aliases: []string{"g", "gen"},
	}
	cmd.AddCommand(genModelCmd(), genPageCmd(), genComponentCmd())
	return cmd
}

// -----------------------------------------------------------------------
// generate model
// -----------------------------------------------------------------------

func genModelCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "model <Name>",
		Short: "Generate a GoKS model",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := filepath.Join("models", strings.ToLower(args[0])+".go")
			if err := writeTemplate(path, tmplGenModel, map[string]string{"Name": args[0]}); err != nil {
				return err
			}
			fmt.Printf("  %s %s\n", color.GreenString("create"), path)
			return nil
		},
	}
}

// -----------------------------------------------------------------------
// generate page
// -----------------------------------------------------------------------

func genPageCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "page <path>",
		Short: "Generate a GoKS page component",
		Long: `Generate a GoKS page component.

Examples:
  goks generate page about           → app/about/page.gox
  goks generate page blog/_slug      → app/blog/_slug/page.gox`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return genPage(args[0])
		},
	}
}

func genPage(pagePath string) error {
	route := strings.TrimPrefix(filepath.ToSlash(filepath.Clean(pagePath)), "/")
	dir := filepath.Join("app", route)
	path := filepath.Join(dir, "page.gox")

	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("file already exists: %s", path)
	}

	packageName := filepath.Base(dir)
	packageName = strings.ReplaceAll(packageName, "-", "")
	packageName = strings.ReplaceAll(packageName, " ", "")
	if packageName == "" || packageName == "." || route == "" {
		packageName = "app"
	}

	data := map[string]string{
		"PackageName": strings.ToLower(packageName),
		"Route":       route,
		"PageTitle":   title(filepath.Base(route)),
	}

	if err := writeTemplate(path, tmplGenPage, data); err != nil {
		return err
	}
	fmt.Printf("  %s %s\n", color.GreenString("create"), path)
	return nil
}

// -----------------------------------------------------------------------
// generate component
// -----------------------------------------------------------------------

func genComponentCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "component <Name>",
		Short:   "Generate a reusable GoKS UI component",
		Aliases: []string{"comp", "c"},
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := filepath.Join("app", "components")
			if _, err := os.Stat(dir); os.IsNotExist(err) {
				dir = "components"
			}
			path := filepath.Join(dir, strings.ToLower(args[0])+".gox")
			if err := os.MkdirAll(dir, 0755); err != nil {
				return err
			}
			if err := writeTemplate(path, tmplGenComponent, map[string]string{"Name": title(args[0])}); err != nil {
				return err
			}
			fmt.Printf("  %s %s\n", color.GreenString("create"), path)
			return nil
		},
	}
}

// -----------------------------------------------------------------------
// helpers
// -----------------------------------------------------------------------

func writeTemplate(path, tmplStr string, data any) error {
	// reuse renderTemplate from new.go (same package)
	return renderTemplate(path, filepath.Base(path), tmplStr, data)
}

func title(s string) string {
	if s == "" {
		return ""
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// -----------------------------------------------------------------------
// Generator templates
// -----------------------------------------------------------------------

var tmplGenModel = `package models

import "github.com/misbakhul29/goks/pkg/orm"

// {{.Name}} is a GoKS ORM model.
type {{.Name}} struct {
	orm.Model
	// Add your fields here
}
`

var tmplGenPage = `package {{.PackageName}}

import (
	"github.com/misbakhul29/goks/pkg/component"
	"github.com/misbakhul29/goks/pkg/html"
)

// Page is the page component for the "{{.Route}}" route.
type Page struct {
	component.ComponentBase
}

func (p *Page) Render() *component.Node {
	return (
		<div class="p-8">
			<h1 class="text-3xl font-bold mb-4">{{.PageTitle}}</h1>
			<p class="text-slate-600 dark:text-slate-400">This is the {{.PageTitle}} page.</p>
		</div>
	)
}
`

var tmplGenComponent = `package components

import (
	"github.com/misbakhul29/goks/pkg/component"
	"github.com/misbakhul29/goks/pkg/html"
)

// {{.Name}} is a reusable GoKS UI component.
type {{.Name}} struct {
	component.ComponentBase
}

func (c *{{.Name}}) Render() *component.Node {
	return (
		<div class="p-4 rounded-lg border border-slate-200 dark:border-slate-800">
			<span class="text-slate-900 dark:text-slate-100 font-medium">{{.Name}} Component</span>
		</div>
	)
}
`
