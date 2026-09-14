package generator

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"github.com/misbakhul29/goks/internal/version"
)

type RouteNode struct {
	Path        string // e.g., "/" or "/about"
	PkgName     string // e.g., "pkg_about"
	ImportPath  string // e.g., "github.com/user/test/app/about"
	HasPage     bool
	HasLayout   bool
	HasNotFound bool
	Children    []*RouteNode
}

// APIRoute describes a file-based API route discovered in the app directory.
type APIRoute struct {
	Path       string   // e.g. "/api/users/:id"
	PkgAlias   string   // e.g. "api_users_id"
	ImportPath string   // e.g. "module/app/api/users/_id"
	Methods    []string // e.g. ["GET", "POST", "DELETE"]
}

var validHTTPMethods = map[string]bool{
	"GET":     true,
	"POST":    true,
	"PUT":     true,
	"DELETE":  true,
	"PATCH":   true,
	"HEAD":    true,
	"OPTIONS": true,
}

var httpMethodOrder = []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"}

func parseRouteMethods(filePath string) ([]string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	fset := token.NewFileSet()
	fileNode, err := parser.ParseFile(fset, filePath, data, parser.ParseComments)
	found := make(map[string]bool)
	if err == nil {
		for _, decl := range fileNode.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv != nil {
				continue
			}
			if validHTTPMethods[fn.Name.Name] {
				found[fn.Name.Name] = true
			}
		}
	} else {
		// Fallback regex scanner if AST parse encounters non-standard syntax
		for m := range validHTTPMethods {
			re := regexp.MustCompile(`(?m)^func\s+` + m + `\s*\(`)
			if re.Match(data) {
				found[m] = true
			}
		}
	}

	var methods []string
	for _, m := range httpMethodOrder {
		if found[m] {
			methods = append(methods, m)
		}
	}
	return methods, nil
}

func cleanRoutePath(rel string) string {
	if rel == "." || rel == "" {
		return "/"
	}
	parts := strings.Split(filepath.ToSlash(rel), "/")
	for i, p := range parts {
		if strings.HasPrefix(p, "[...") && strings.HasSuffix(p, "]") {
			parts[i] = "*" + strings.TrimSuffix(strings.TrimPrefix(p, "[..."), "]")
		} else if strings.HasPrefix(p, "[") && strings.HasSuffix(p, "]") {
			parts[i] = ":" + strings.TrimSuffix(strings.TrimPrefix(p, "["), "]")
		} else if strings.HasPrefix(p, "_") {
			parts[i] = ":" + p[1:]
		}
	}
	res := "/" + strings.Join(parts, "/")
	for strings.Contains(res, "//") {
		res = strings.ReplaceAll(res, "//", "/")
	}
	return res
}

func normalizeImportRel(rel string) string {
	if rel == "." || rel == "" {
		return ""
	}
	parts := strings.Split(filepath.ToSlash(rel), "/")
	for i, p := range parts {
		if strings.HasPrefix(p, "[...") && strings.HasSuffix(p, "]") {
			parts[i] = "_" + strings.TrimSuffix(strings.TrimPrefix(p, "[..."), "]")
		} else if strings.HasPrefix(p, "[") && strings.HasSuffix(p, "]") {
			parts[i] = "_" + strings.TrimSuffix(strings.TrimPrefix(p, "["), "]")
		}
	}
	return strings.Join(parts, "/")
}

func sanitizeIdentifier(s string) string {
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			b.WriteRune(r)
		} else {
			b.WriteRune('_')
		}
	}
	res := b.String()
	for strings.Contains(res, "__") {
		res = strings.ReplaceAll(res, "__", "_")
	}
	return strings.Trim(res, "_")
}

func sanitizePkgAlias(rel string, aliasCounts map[string]int) string {
	if rel == "." || rel == "" {
		return "api_root"
	}
	clean := sanitizeIdentifier(rel)
	if !strings.HasPrefix(clean, "api_") {
		clean = "api_" + clean
	}
	if count, exists := aliasCounts[clean]; exists {
		aliasCounts[clean] = count + 1
		return fmt.Sprintf("%s_%d", clean, count+1)
	}
	aliasCounts[clean] = 1
	return clean
}

func findRouteFile(dir string) string {
	for _, name := range []string{"route.go", "route.gox"} {
		p := filepath.Join(dir, name)
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
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

// GenerateRouter scans the app directory and generates the router files.
func GenerateRouter(appDir string, isProd bool) error {
	return generateRouterInternal(appDir, isProd, false)
}

// GenerateStandaloneRouter generates router files optimized for standalone build.
// The server_main.go will embed all assets (WASM, CSS, wasm_exec.js, public/) via //go:embed.
func GenerateStandaloneRouter(appDir string) error {
	return generateRouterInternal(appDir, true, true)
}

func generateRouterInternal(appDir string, isProd bool, standalone bool) error {
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

	var apiRoutes []APIRoute
	aliasCounts := make(map[string]int)

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
			if routeFile := findRouteFile(path); routeFile != "" {
				if methods, err := parseRouteMethods(routeFile); err == nil && len(methods) > 0 {
					apiRoutes = append(apiRoutes, APIRoute{
						Path:       "/",
						PkgAlias:   sanitizePkgAlias(".", aliasCounts),
						ImportPath: moduleName + "/app",
						Methods:    methods,
					})
				}
			}
			return nil
		}

		routePath := cleanRoutePath(rel)
		normalizedRel := normalizeImportRel(rel)
		pkgAlias := "pkg_" + sanitizeIdentifier(rel)
		importPath := moduleName + "/app/" + normalizedRel

		node := &RouteNode{
			Path:       routePath,
			PkgName:    pkgAlias,
			ImportPath: importPath,
		}
		checkFiles(path, node)

		if node.HasPage || node.HasLayout {
			insertNode(rootNode, node)
		}

		if routeFile := findRouteFile(path); routeFile != "" {
			if methods, err := parseRouteMethods(routeFile); err == nil && len(methods) > 0 {
				apiRoutes = append(apiRoutes, APIRoute{
					Path:       routePath,
					PkgAlias:   sanitizePkgAlias(rel, aliasCounts),
					ImportPath: importPath,
					Methods:    methods,
				})
			}
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

	if standalone {
		if err := writeStandaloneServerMain(entryDir, appDir, moduleName, apiRoutes); err != nil {
			return err
		}
	} else {
		if err := writeServerMain(entryDir, appDir, moduleName, isProd, apiRoutes); err != nil {
			return err
		}
	}

	if err := writeEntryGoMod(entryDir, appDir, moduleName); err != nil {
		return err
	}

	return nil
}

func getGoKSVersion() string {
	return version.Current()
}

func writeEntryGoMod(entryDir, appDir, moduleName string) error {
	content := fmt.Sprintf(`module goks_entry

go 1.26.6

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

	if _, err := os.Stat(filepath.Join(dir, "not-found.go")); err == nil {
		node.HasNotFound = true
	} else if _, err := os.Stat(filepath.Join(dir, "not-found.gox")); err == nil {
		node.HasNotFound = true
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

var registeredPageRoutes = []string{
	{{range .PageRoutes}}"{{.}}",
	{{end}}
}

// AppRouter is the generated file-based router.
type AppRouter struct {
	component.ComponentBase
	{{range .Fields}}
	{{.Name}} {{.Type}}
	{{end}}
	not_found component.Renderable
}

func (r *AppRouter) HasMatchedPage(path string) bool {
	return router.MatchAnyRoute(registeredPageRoutes, path)
}

func (r *AppRouter) PageRoutes() []string {
	return registeredPageRoutes
}

func (r *AppRouter) Render() *component.Node {
	current := router.CurrentPath.Get()
	var children *component.Node
	if !r.HasMatchedPage(current) {
		children = component.C(r.get_not_found())
	} else {
		children = {{.ChildrenCode}}
	}
	{{if .HasRootLayout}}
	return component.C(r.get_layout_pkg_root(children))
	{{else}}
	return children
	{{end}}
}
`

type TmplData struct {
	ModuleName    string
	Imports       []Import
	Fields        []Field
	ChildrenCode  string
	HasRootLayout bool
	PageRoutes    []string
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

	var pageRoutes []string
	collectPageRoutes(rootNode, &pageRoutes)

	childrenCode := generateChildrenCode(rootNode, true)

	data := TmplData{
		ModuleName:    moduleName,
		Imports:       imports,
		Fields:        fields,
		ChildrenCode:  childrenCode,
		HasRootLayout: rootNode.HasLayout,
		PageRoutes:    pageRoutes,
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

func collectPageRoutes(node *RouteNode, routes *[]string) {
	if node.HasPage {
		*routes = append(*routes, node.Path)
	}
	for _, child := range node.Children {
		collectPageRoutes(child, routes)
	}
}

func generateChildrenCode(node *RouteNode, isRoot bool) string {
	pkg := node.PkgName
	if isRoot {
		pkg = "pkg_root"
	}

	var items []string
	if node.HasPage {
		items = append(items, fmt.Sprintf(`component.C(&router.PageRoute{Path: "%s", Exact: true, Component: r.get_page_%s()})`, node.Path, pkg))
	}

	for _, child := range node.Children {
		items = append(items, generateChildCode(child))
	}

	if len(items) > 0 {
		return "component.Fragment(\n\t\t\t" + strings.Join(items, ",\n\t\t\t") + ",\n\t\t)"
	}
	return "component.Text(\"\")"
}

func generateGetters(node *RouteNode, out *strings.Builder) {
	pkg := node.PkgName
	if node.PkgName == "app" {
		pkg = "pkg_root"
	}

	if node.Path == "/" {
		out.WriteString("\nfunc (r *AppRouter) get_not_found() component.Renderable {\n\tif r.not_found == nil {\n")
		if node.HasNotFound {
			out.WriteString("\t\tr.not_found = &pkg_root.NotFound{}\n")
		} else {
			out.WriteString("\t\tr.not_found = &router.DefaultNotFound{}\n")
		}
		out.WriteString("\t}\n\treturn r.not_found\n}\n")
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
	client.Mount("__goks", &AppRouter{})
	select {}
}
`
	return os.WriteFile(filepath.Join(entryDir, "client_main.go"), []byte(content), 0644)
}

func buildAPIRouteCode(apiRoutes []APIRoute) (string, string) {
	var importsB strings.Builder
	var regB strings.Builder
	for _, route := range apiRoutes {
		importsB.WriteString(fmt.Sprintf("\n\t%s \"%s\"", route.PkgAlias, route.ImportPath))
		for _, method := range route.Methods {
			regB.WriteString(fmt.Sprintf("\n\tsrv.Router().Handle(\"%s\", \"%s\", %s.%s)",
				method, route.Path, route.PkgAlias, method))
		}
	}
	return importsB.String(), regB.String()
}

func writeServerMain(entryDir, appDir, moduleName string, isProd bool, apiRoutes []APIRoute) error {
	hasConfig := false
	if b, err := os.ReadFile(filepath.Join(appDir, "config", "goks.config.go")); err == nil {
		if strings.Contains(string(b), "func ServerConfig") {
			hasConfig = true
		}
	}

	apiImports, apiRegistrations := buildAPIRouteCode(apiRoutes)

	imports := `	"log"
	"os"
	"strconv"
	"github.com/misbakhul29/goks/runtime/server"`
	if hasConfig {
		imports += fmt.Sprintf("\n\t\"%s/config\"", moduleName)
	}
	imports += apiImports

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
		configInit = fmt.Sprintf(`	cfg := config.ServerConfig()
	if cfg.Port == 0 {
		cfg.Port = 3000
	}
	if envPort := os.Getenv("PORT"); envPort != "" {
		if p, err := strconv.Atoi(envPort); err == nil {
			cfg.Port = p
		}
	}
	if envPort := os.Getenv("GOKS_CHILD_PORT"); envPort != "" {
		if p, err := strconv.Atoi(envPort); err == nil {
			cfg.Port = p
		}
	}
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
	srv := server.NewDev(cfg)%s
	if exportDir := os.Getenv("GOKS_EXPORT_DIR"); exportDir != "" {
		if err := srv.ExportStatic(exportDir); err != nil {
			log.Fatalf("[GoKS] Static export failed: %%v", err)
		}
		return
	}
	log.Fatal(srv.Start())
}
`, imports, configInit, apiRegistrations)
	return os.WriteFile(filepath.Join(entryDir, "server_main.go"), []byte(content), 0644)
}

// writeStandaloneServerMain generates server_main.go with //go:embed directives
// for all assets (WASM, CSS, wasm_exec.js, public/) so the binary is self-contained.
func writeStandaloneServerMain(entryDir, appDir, moduleName string, apiRoutes []APIRoute) error {
	hasConfig := false
	if b, err := os.ReadFile(filepath.Join(appDir, "config", "goks.config.go")); err == nil {
		if strings.Contains(string(b), "func ServerConfig") {
			hasConfig = true
		}
	}

	apiImports, apiRegistrations := buildAPIRouteCode(apiRoutes)

	// Check if public/ directory exists so we only embed it when present
	hasPublic := false
	publicDir := filepath.Join(entryDir, "public")
	if _, err := os.Stat(publicDir); err == nil {
		hasPublic = true
	}

	var importsB strings.Builder
	importsB.WriteString(`	"embed"
	"log"
	"os"
	"strconv"
	"github.com/misbakhul29/goks/runtime/server"`)
	if hasConfig {
		importsB.WriteString(fmt.Sprintf("\n\t\"%s/config\"", moduleName))
	}
	importsB.WriteString(apiImports)

	// Build embed directives
	var embedB strings.Builder
	embedB.WriteString("//go:embed app.wasm app.css wasm_exec.js\n")
	embedB.WriteString("var embeddedAssets embed.FS\n\n")
	if hasPublic {
		embedB.WriteString("//go:embed public\n")
		embedB.WriteString("var embeddedPublic embed.FS\n\n")
	}

	// Build config init
	var configB strings.Builder
	if hasConfig {
		configB.WriteString(`	cfg := config.ServerConfig()
	if cfg.Port == 0 {
		cfg.Port = 3000
	}
	if envPort := os.Getenv("PORT"); envPort != "" {
		if p, err := strconv.Atoi(envPort); err == nil {
			cfg.Port = p
		}
	}
	cfg.AppDir = "."
	cfg.DevMode = false
	cfg.Standalone = true
	cfg.EmbeddedAssets = embeddedAssets
	cfg.Root = &AppRouter{}`)
		if hasPublic {
			configB.WriteString("\n\tcfg.EmbeddedPublic = embeddedPublic")
		}
	} else {
		configB.WriteString(`	port := 3000
	if envPort := os.Getenv("PORT"); envPort != "" {
		if p, err := strconv.Atoi(envPort); err == nil {
			port = p
		}
	}
	cfg := server.Config{
		Port:           port,
		AppDir:         ".",
		DevMode:        false,
		Standalone:     true,
		EmbeddedAssets: embeddedAssets,
		Root:           &AppRouter{},`)
		if hasPublic {
			configB.WriteString("\n\t\tEmbeddedPublic: embeddedPublic,")
		}
		configB.WriteString("\n\t}")
	}

	content := fmt.Sprintf(`//go:build !js || !wasm
// Code generated by GoKS. DO NOT EDIT.

package main

import (
%s
)

%s
func main() {
%s
	srv := server.NewDev(cfg)%s
	log.Fatal(srv.Start())
}
`, importsB.String(), embedB.String(), configB.String(), apiRegistrations)

	return os.WriteFile(filepath.Join(entryDir, "server_main.go"), []byte(content), 0644)
}

func generateChildCode(child *RouteNode) string {
	if child.HasLayout {
		childChildren := generateChildrenCode(child, false)
		return fmt.Sprintf(`component.C(&router.PageRoute{Path: "%s", Exact: false, Component: r.get_layout_%s(%s)})`, child.Path, child.PkgName, childChildren)
	}

	// Leaf node (Page only)
	return fmt.Sprintf(`component.C(&router.PageRoute{Path: "%s", Exact: true, Component: r.get_page_%s()})`, child.Path, child.PkgName)
}
