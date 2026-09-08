package cli

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"unicode"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

func PageCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "page <route>",
		Short: "Generate a new page component in the app/ directory",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return generatePage(args[0])
		},
	}
}

var tmplNewPage = `package {{.PackageName}}

import (
	"github.com/misbakhul29/goks/pkg/component"
	"github.com/misbakhul29/goks/pkg/html"
)

// Page is the page component for the "{{.Route}}" route.
type Page struct {
	component.ComponentBase
}

func (p *Page) Render() *component.Node {
	return html.Div(
		html.H1("{{.PageTitle}}").Class("text-3xl font-bold mb-4"),
		html.P("This is the {{.PageTitle}} page."),
	).Class("p-8")
}
`

func generatePage(route string) error {
	// Clean the route and build the target directory inside app/
	route = strings.TrimPrefix(filepath.ToSlash(filepath.Clean(route)), "/")
	
	appDir := "app"
	targetDir := filepath.Join(appDir, route)
	
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	targetFile := filepath.Join(targetDir, "page.go")
	if _, err := os.Stat(targetFile); err == nil {
		return fmt.Errorf("file already exists: %s", targetFile)
	}

	// Determine package name from the last part of the route
	packageName := filepath.Base(targetDir)
	// Go package names shouldn't contain hyphens or spaces, generally
	packageName = strings.ReplaceAll(packageName, "-", "")
	packageName = strings.ReplaceAll(packageName, " ", "")
	
	// Create a readable title for the component
	pageTitle := filepath.Base(route)
	pageTitle = strings.Title(strings.ReplaceAll(pageTitle, "-", " "))
	if pageTitle == "" || pageTitle == "." || route == "" {
		pageTitle = "Home"
		packageName = "app"
	}

	data := map[string]string{
		"PackageName": strings.ToLower(packageName),
		"Route":       route,
		"PageTitle":   capitalize(pageTitle),
	}

	t, err := template.New("page").Parse(tmplNewPage)
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return err
	}

	if err := os.WriteFile(targetFile, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	fmt.Println(color.CyanString("\n  📄 Created page:"), color.WhiteString(route))
	fmt.Println(color.HiBlackString("  create  %s", targetFile))
	fmt.Println()

	return nil
}

func capitalize(s string) string {
	if s == "" {
		return ""
	}
	runes := []rune(s)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}
