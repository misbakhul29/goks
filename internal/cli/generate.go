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
  goks generate page about           → app/pages/about.go
  goks generate page blog/[slug]     → app/pages/blog/[slug].go`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return genPage(args[0])
		},
	}
}

func genPage(pagePath string) error {
	parts := strings.Split(pagePath, "/")
	name := parts[len(parts)-1]
	structName := title(strings.ReplaceAll(strings.ReplaceAll(name, "[", ""), "]", "")) + "Page"

	dir := filepath.Join("app", "pages", filepath.Join(parts[:len(parts)-1]...))
	path := filepath.Join(dir, name+".go")

	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	if err := writeTemplate(path, tmplGenPage, map[string]string{"StructName": structName}); err != nil {
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
			path := filepath.Join("app", "components", strings.ToLower(args[0])+".go")
			if err := writeTemplate(path, tmplGenComponent, map[string]string{"Name": args[0]}); err != nil {
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

import "github.com/misbakhulmunir/goks/pkg/orm"

// {{.Name}} is a GoKS ORM model.
type {{.Name}} struct {
	orm.Model
	// Add your fields here
}
`

var tmplGenPage = `//go:build js && wasm

package pages

import "github.com/misbakhulmunir/goks/pkg/component"

// {{.StructName}} is a GoKS page component.
type {{.StructName}} struct {
	component.ComponentBase
}

func (p *{{.StructName}}) Render() *component.Node {
	return component.H("div", component.Props{"class": "page"},
		component.H("h1", nil,
			component.Text("{{.StructName}}"),
		),
	)
}
`

var tmplGenComponent = `//go:build js && wasm

package components

import "github.com/misbakhulmunir/goks/pkg/component"

// {{.Name}} is a reusable GoKS UI component.
type {{.Name}} struct {
	component.ComponentBase
}

func (c *{{.Name}}) Render() *component.Node {
	return component.H("div", component.Props{"class": "{{.Name}}"},
		component.Text("{{.Name}} component"),
	)
}
`
