package generator

import (
	"bytes"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"text/template"
)

type RouteNode struct {
	Path       string // e.g., "/" or "/about"
	PkgName    string // e.g., "pkg_about"
	ImportPath string // e.g., "github.com/user/test/app/about"
	HasPage    bool
	HasLayout  bool
	Children   []*RouteNode
}

// GetModuleName extracts the module name from go.mod
func GetModuleName(appDir string) (string, error) {
	b, err := os.ReadFile(filepath.Join(appDir, "go.mod"))
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module ")), nil
		}
	}
	return "", fmt.Errorf("module directive not found in go.mod")
}

// GenerateRouter scans the app directory and generates app/routes.gen.go
func GenerateRouter(appDir string, isProd bool) error {
	moduleName, err := GetModuleName(appDir)
	if err != nil {
		return err
	}

	appRoot := filepath.Join(appDir, "app")
	if _, err := os.Stat(appRoot); os.IsNotExist(err) {
		return nil // No app directory, nothing to generate
	}

	rootNode := &RouteNode{
		Path:       "/",
		PkgName:    "app", // root is always app
		ImportPath: "",    // no import needed for root
	}

	err = filepath.Walk(appRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			return nil
		}

		// Calculate relative route path
		rel, err := filepath.Rel(appRoot, path)
		if err != nil {
			return err
		}
		
		if rel == "." {
			checkFiles(path, rootNode)
			return nil
		}

		routePath := "/" + strings.ReplaceAll(rel, string(filepath.Separator), "/")
		// Convert _slug to :slug for dynamic routes
		parts := strings.Split(routePath, "/")
		for i, p := range parts {
			if strings.HasPrefix(p, "_") {
				parts[i] = ":" + p[1:]
			}
		}
		routePath = strings.Join(parts, "/")

		// Calculate package alias
		pkgAlias := "pkg_" + strings.ReplaceAll(rel, string(filepath.Separator), "_")
		importPath := moduleName + "/app/" + filepath.ToSlash(rel)

		node := &RouteNode{
			Path:       routePath,
			PkgName:    pkgAlias,
			ImportPath: importPath,
		}
		checkFiles(path, node)

		if node.HasPage || node.HasLayout {
			insertNode(rootNode, node)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to scan app directory: %w", err)
	}

	entryDir := filepath.Join(appDir, ".goks", "entry")
	if err := os.MkdirAll(entryDir, 0755); err != nil {
		return err
	}

	if err := writeRouterFile(entryDir, rootNode, moduleName); err != nil {
		return err
	}

	if err := writeClientMain(entryDir, moduleName); err != nil {
		return err
	}

	if err := writeServerMain(entryDir, appDir, moduleName, isProd); err != nil {
		return err
	}

	if err := writeEntryGoMod(entryDir, appDir, moduleName); err != nil {
		return err
	}

	return nil
}

func getGoKSVersion() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		if info.Main.Version != "" && info.Main.Version != "(devel)" {
			return info.Main.Version
		}
	}
	return "v0.1.0"
}

func writeEntryGoMod(entryDir, appDir, moduleName string) error {
	content := fmt.Sprintf(`module goks_entry

go 1.22

require %s v0.0.0
require github.com/misbakhul29/goks %s

replace %s => ../workspace
`, moduleName, getGoKSVersion(), moduleName)
	
	// If the user's go.mod has a replace for goks, we should copy it
	if b, err := os.ReadFile(filepath.Join(appDir, "go.mod")); err == nil {
		lines := strings.Split(string(b), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "replace ") {
				// parse replace path
				parts := strings.Split(line, "=>")
				if len(parts) == 2 {
					targetPath := strings.TrimSpace(parts[1])
					if strings.HasPrefix(targetPath, ".") {
						// Adjust relative path by 2 levels up
						targetPath = filepath.Join("../../", targetPath)
						content += fmt.Sprintf("replace %s => %s\n", strings.TrimSpace(strings.TrimPrefix(parts[0], "replace ")), targetPath)
						continue
					}
				}
				content += line + "\n"
			}
		}
	}

	return os.WriteFile(filepath.Join(entryDir, "go.mod"), []byte(content), 0644)
}

func checkFiles(dir string, node *RouteNode) {
	if _, err := os.Stat(filepath.Join(dir, "page.go")); err == nil {
		node.HasPage = true
	} else if _, err := os.Stat(filepath.Join(dir, "page.gox")); err == nil {
		node.HasPage = true
	}

	if _, err := os.Stat(filepath.Join(dir, "layout.go")); err == nil {
		node.HasLayout = true
	} else if _, err := os.Stat(filepath.Join(dir, "layout.gox")); err == nil {
		node.HasLayout = true
	}
}

// insertNode inserts a node into the tree based on path hierarchy
func insertNode(root, newNode *RouteNode) {
	// Find the deepest layout parent for this node
	// For simplicity in this iteration, we just flat-append everything to the root
	// unless it's a nested layout. Proper nested layout tree requires matching prefixes.
	parent := findDeepestParent(root, newNode.Path)
	parent.Children = append(parent.Children, newNode)
}

func findDeepestParent(current *RouteNode, targetPath string) *RouteNode {
	for _, child := range current.Children {
		if child.HasLayout {
			// e.g. target = "/dashboard/settings", child = "/dashboard"
			prefix := child.Path
			if prefix != "/" && strings.HasPrefix(targetPath, prefix) && len(targetPath) > len(prefix) && targetPath[len(prefix)] == '/' {
				return findDeepestParent(child, targetPath)
			}
		}
	}
	return current
}

const routerTemplate = `// Code generated by GoKS. DO NOT EDIT.
package main

import (
	"github.com/misbakhul29/goks/pkg/router"
	"github.com/misbakhul29/goks/pkg/component"
	pkg_root "{{.ModuleName}}/app"
	{{range .Imports}}
	{{.Alias}} "{{.Path}}"
	{{end}}
)

// AppRouter is the generated file-based router.
type AppRouter struct {
	component.ComponentBase
	{{range .Fields}}
	{{.Name}} {{.Type}}
	{{end}}
}

func (r *AppRouter) Render() *component.Node {
	return {{.RenderCode}}
}
`

type TmplData struct {
	ModuleName string
	Imports    []Import
	Fields     []Field
	RenderCode string
}

type Field struct {
	Name string
	Type string
}

type Import struct {
	Alias string
	Path  string
}

func writeRouterFile(entryDir string, rootNode *RouteNode, moduleName string) error {
	var imports []Import
	collectImports(rootNode, &imports)

	var fields []Field
	collectFields(rootNode, &fields)

	renderCode := generateRenderCode(rootNode, true)

	data := TmplData{
		ModuleName: moduleName,
		Imports:    imports,
		Fields:     fields,
		RenderCode: renderCode,
	}

	tmpl, err := template.New("router").Parse(routerTemplate)
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return err
	}

	var gettersBuf strings.Builder
	generateGetters(rootNode, &gettersBuf)
	buf.WriteString(gettersBuf.String())

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		// fallback to unformatted if error
		formatted = buf.Bytes()
	}

	outFile := filepath.Join(entryDir, "router.go")
	return os.WriteFile(outFile, formatted, 0644)
}

func collectImports(node *RouteNode, imports *[]Import) {
	if node.ImportPath != "" && node.PkgName != "app" && (node.HasPage || node.HasLayout) {
		*imports = append(*imports, Import{Alias: node.PkgName, Path: node.ImportPath})
	}
	for _, child := range node.Children {
		collectImports(child, imports)
	}
}

func collectFields(node *RouteNode, fields *[]Field) {
	pkg := node.PkgName
	if node.PkgName == "app" {
		pkg = "pkg_root"
	}

	if node.HasPage {
		*fields = append(*fields, Field{
			Name: "page_" + pkg,
			Type: "*" + pkg + ".Page",
		})
	}
	if node.HasLayout {
		*fields = append(*fields, Field{
			Name: "layout_" + pkg,
			Type: "*" + pkg + ".Layout",
		})
	}

	for _, child := range node.Children {
		collectFields(child, fields)
	}
}

func generateRenderCode(node *RouteNode, isRoot bool) string {
	pkg := node.PkgName
	if isRoot {
		pkg = "pkg_root" 
	}

	var childrenCode string
	
	// If it has a Page, add it to children
	var items []string
	if node.HasPage {
		items = append(items, fmt.Sprintf(`component.C(&router.PageRoute{Path: "%s", Exact: true, Component: r.get_page_%s()})`, node.Path, pkg))
	}
	
	// Add all sub-routes
	for _, child := range node.Children {
		items = append(items, generateChildCode(child))
	}

	if len(items) > 0 {
		childrenCode = "component.Fragment(\n\t\t\t" + strings.Join(items, ",\n\t\t\t") + ",\n\t\t)"
	} else {
		childrenCode = "component.Text(\"\")"
	}

	if node.HasLayout {
		return fmt.Sprintf(`component.C(r.get_layout_%s(%s))`, pkg, childrenCode)
	}

	// If no layout but is root, just return the list
	return childrenCode
}

func generateGetters(node *RouteNode, out *strings.Builder) {
	pkg := node.PkgName
	if node.PkgName == "app" {
		pkg = "pkg_root"
	}

	if node.HasPage {
		out.WriteString(fmt.Sprintf(`
func (r *AppRouter) get_page_%s() *%s.Page {
	if r.page_%s == nil {
		r.page_%s = &%s.Page{}
	}
	return r.page_%s
}
`, pkg, pkg, pkg, pkg, pkg, pkg))
	}

	if node.HasLayout {
		out.WriteString(fmt.Sprintf(`
func (r *AppRouter) get_layout_%s(children *component.Node) *%s.Layout {
	if r.layout_%s == nil {
		r.layout_%s = &%s.Layout{}
	}
	r.layout_%s.Children = children
	return r.layout_%s
}
`, pkg, pkg, pkg, pkg, pkg, pkg, pkg))
	}

	for _, child := range node.Children {
		generateGetters(child, out)
	}
}
func writeClientMain(entryDir, moduleName string) error {
	content := `//go:build js && wasm
// Code generated by GoKS. DO NOT EDIT.

package main

import (
	"github.com/misbakhul29/goks/runtime/client"
	"github.com/misbakhul29/goks/pkg/router"
)

func main() {
	router.InitClientRouter()
	client.Mount("#app", &AppRouter{})
	select {}
}
`
	return os.WriteFile(filepath.Join(entryDir, "client_main.go"), []byte(content), 0644)
}

func writeServerMain(entryDir, appDir, moduleName string, isProd bool) error {
	hasConfig := false
	if b, err := os.ReadFile(filepath.Join(appDir, "config", "goks.config.go")); err == nil {
		if strings.Contains(string(b), "func ServerConfig") {
			hasConfig = true
		}
	}

	imports := `	"log"
	"os"
	"strconv"
	"github.com/misbakhul29/goks/runtime/server"`
	if hasConfig {
		imports += fmt.Sprintf("\n\t\"%s/config\"", moduleName)
	}

	appDirStr := `"../.."`
	devModeStr := `true`
	if isProd {
		appDirStr = `"."`
		devModeStr = `false`
	}

	configInit := fmt.Sprintf(`	port := 3000
	if envPort := os.Getenv("PORT"); envPort != "" {
		if p, err := strconv.Atoi(envPort); err == nil {
			port = p
		}
	}
	if envPort := os.Getenv("GOKS_CHILD_PORT"); envPort != "" {
		if p, err := strconv.Atoi(envPort); err == nil {
			port = p
		}
	}
	cfg := server.Config{
		Port:    port,
		AppDir:  %s,
		DevMode: %s,
		Root:    &AppRouter{},
	}`, appDirStr, devModeStr)
	if hasConfig {
		configInit = fmt.Sprintf(`	port := 3000
	if envPort := os.Getenv("PORT"); envPort != "" {
		if p, err := strconv.Atoi(envPort); err == nil {
			port = p
		}
	}
	if envPort := os.Getenv("GOKS_CHILD_PORT"); envPort != "" {
		if p, err := strconv.Atoi(envPort); err == nil {
			port = p
		}
	}
	cfg := config.ServerConfig()
	cfg.Port = port
	cfg.AppDir = %s
	cfg.DevMode = %s
	cfg.Root = &AppRouter{}`, appDirStr, devModeStr)
	}

	content := fmt.Sprintf(`//go:build !js || !wasm
// Code generated by GoKS. DO NOT EDIT.

package main

import (
%s
)

func main() {
%s
	srv := server.NewDev(cfg)
	log.Fatal(srv.Start())
}
`, imports, configInit)
	return os.WriteFile(filepath.Join(entryDir, "server_main.go"), []byte(content), 0644)
}
func generateChildCode(child *RouteNode) string {
	if child.HasLayout {
		return fmt.Sprintf(`component.C(&router.PageRoute{Path: "%s", Exact: false, Component: %s})`, child.Path, generateRenderCode(child, false))
	}
	
	// Leaf node (Page only)
	return fmt.Sprintf(`component.C(&router.PageRoute{Path: "%s", Exact: true, Component: r.get_page_%s()})`, child.Path, child.PkgName)
}
