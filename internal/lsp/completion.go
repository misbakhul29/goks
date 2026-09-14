package lsp

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// GetCompletions computes completion items based on document text and cursor position.
func GetCompletions(content string, pos Position, docURI string) []CompletionItem {
	lines := strings.Split(content, "\n")
	if pos.Line >= len(lines) {
		return nil
	}

	line := lines[pos.Line]
	charIdx := pos.Character
	if charIdx > len(line) {
		charIdx = len(line)
	}
	prefix := line[:charIdx]

	// 1. Inside import block context
	if isInsideImport(lines, pos.Line) {
		return getImportCompletions()
	}

	// 2. Tag completion context: user typed `<` or `<tag`
	if tagMatch := tagContextRegex.FindStringSubmatch(prefix); len(tagMatch) > 0 {
		return getTagCompletions(docURI)
	}

	// 3. Attribute completion context: user is inside an open tag `<div ...`
	if isInsideTag(prefix) {
		return getAttributeCompletions()
	}

	// 4. Dot completion context: user typed `pkg.` (e.g. `router.`, `google.`)
	if dotMatch := dotContextRegex.FindStringSubmatch(prefix); len(dotMatch) > 0 {
		return getPackageMemberCompletions(content, dotMatch[1])
	}

	// 5. Go code symbol/package completion (e.g. typing `r := router` or `router` or `google`)
	return getGoSymbolCompletions(content, prefix)
}

var tagContextRegex = regexp.MustCompile(`<([a-zA-Z0-9_.]*)$`)
var dotContextRegex = regexp.MustCompile(`([a-zA-Z0-9_]+)\.$`)
var identifierRegex = regexp.MustCompile(`([a-zA-Z0-9_]+)$`)

type PackageMember struct {
	Name       string
	Kind       CompletionItemKind
	Detail     string
	InsertText string
	Doc        string
}

type GoKSPackage struct {
	Name        string
	ImportPath  string
	Description string
	Members     []PackageMember
}

var goksPackages = []GoKSPackage{
	{
		Name:        "router",
		ImportPath:  "github.com/misbakhul29/goks/pkg/router",
		Description: "GoKS client & server routing package (UseRouter, Push, Replace, Back, Forward, CurrentPath)",
		Members: []PackageMember{
			{
				Name:       "UseRouter()",
				Kind:       CompletionKindFunction,
				Detail:     "func() *ClientRouter",
				InsertText: "UseRouter()",
				Doc:        "UseRouter returns the client router instance for navigation (Push, Replace, Back, Forward, Path).",
			},
			{
				Name:       "Push(path)",
				Kind:       CompletionKindFunction,
				Detail:     "func(path string)",
				InsertText: "Push(\"${1:path}\")",
				Doc:        "Push navigates to the specified URL path and adds it to browser history.",
			},
			{
				Name:       "Replace(path)",
				Kind:       CompletionKindFunction,
				Detail:     "func(path string)",
				InsertText: "Replace(\"${1:path}\")",
				Doc:        "Replace navigates to the specified URL path replacing the current history entry.",
			},
			{
				Name:       "Back()",
				Kind:       CompletionKindFunction,
				Detail:     "func()",
				InsertText: "Back()",
				Doc:        "Back navigates back to the previous page in history.",
			},
			{
				Name:       "Forward()",
				Kind:       CompletionKindFunction,
				Detail:     "func()",
				InsertText: "Forward()",
				Doc:        "Forward navigates forward to the next page in history.",
			},
			{
				Name:       "Path()",
				Kind:       CompletionKindFunction,
				Detail:     "func() string",
				InsertText: "Path()",
				Doc:        "Path returns the current URL pathname.",
			},
			{
				Name:       "CurrentPath",
				Kind:       CompletionKindVariable,
				Detail:     "*component.Store[string]",
				InsertText: "CurrentPath",
				Doc:        "CurrentPath is a global reactive store holding the current URL path.",
			},
			{
				Name:       "Context",
				Kind:       CompletionKindStruct,
				Detail:     "type Context struct",
				InsertText: "Context",
				Doc:        "Context wraps http.ResponseWriter and *http.Request with a rich API in server handlers.",
			},
		},
	},
	{
		Name:        "google",
		ImportPath:  "github.com/misbakhul29/goks/pkg/font/google",
		Description: "Google Fonts loader for GoKS (à la next/font/google)",
		Members: []PackageMember{
			{
				Name:       "Options",
				Kind:       CompletionKindStruct,
				Detail:     "type Options struct",
				InsertText: "Options{\n\tVariable: \"${1:--font-name}\",\n\tSubsets:  []string{\"latin\"},\n}",
				Doc:        "Options for configuring Google Fonts (Variable, Weight, Subsets, Display).",
			},
			{
				Name:       "Classes(parts...)",
				Kind:       CompletionKindFunction,
				Detail:     "func(parts ...any) string",
				InsertText: "Classes(${1:fonts...})",
				Doc:        "Classes joins Font pointers and CSS class strings into a single class string.",
			},
			{
				Name:       "Geist(opt)",
				Kind:       CompletionKindFunction,
				Detail:     "func(opt Options) *Font",
				InsertText: "Geist(google.Options{Variable: \"${1:--font-geist-sans}\"})",
				Doc:        "Loads Geist font from Google Fonts.",
			},
			{
				Name:       "GeistMono(opt)",
				Kind:       CompletionKindFunction,
				Detail:     "func(opt Options) *Font",
				InsertText: "GeistMono(google.Options{Variable: \"${1:--font-geist-mono}\"})",
				Doc:        "Loads Geist Mono font from Google Fonts.",
			},
			{
				Name:       "SpaceGrotesk(opt)",
				Kind:       CompletionKindFunction,
				Detail:     "func(opt Options) *Font",
				InsertText: "SpaceGrotesk(google.Options{Variable: \"${1:--font-space-grotesk}\"})",
				Doc:        "Loads Space Grotesk font from Google Fonts.",
			},
			{
				Name:       "RubikGlitch(opt)",
				Kind:       CompletionKindFunction,
				Detail:     "func(opt Options) *Font",
				InsertText: "RubikGlitch(google.Options{Variable: \"${1:--font-rubik-glitch}\"})",
				Doc:        "Loads Rubik Glitch font from Google Fonts.",
			},
			{
				Name:       "PermanentMarker(opt)",
				Kind:       CompletionKindFunction,
				Detail:     "func(opt Options) *Font",
				InsertText: "PermanentMarker(google.Options{Variable: \"${1:--font-permanent-marker}\"})",
				Doc:        "Loads Permanent Marker font from Google Fonts.",
			},
			{
				Name:       "Inter(opt)",
				Kind:       CompletionKindFunction,
				Detail:     "func(opt Options) *Font",
				InsertText: "Inter(google.Options{Variable: \"${1:--font-inter}\"})",
				Doc:        "Loads Inter font from Google Fonts.",
			},
			{
				Name:       "Poppins(opt)",
				Kind:       CompletionKindFunction,
				Detail:     "func(opt Options) *Font",
				InsertText: "Poppins(google.Options{Variable: \"${1:--font-poppins}\"})",
				Doc:        "Loads Poppins font from Google Fonts.",
			},
			{
				Name:       "Montserrat(opt)",
				Kind:       CompletionKindFunction,
				Detail:     "func(opt Options) *Font",
				InsertText: "Montserrat(google.Options{Variable: \"${1:--font-montserrat}\"})",
				Doc:        "Loads Montserrat font from Google Fonts.",
			},
		},
	},
	{
		Name:        "component",
		ImportPath:  "github.com/misbakhul29/goks/pkg/component",
		Description: "Core UI component system and Virtual DOM for GoKS",
		Members: []PackageMember{
			{
				Name:       "ComponentBase",
				Kind:       CompletionKindStruct,
				Detail:     "type ComponentBase struct",
				InsertText: "ComponentBase",
				Doc:        "Base struct to embed in GoKS components to enable state management and rerendering.",
			},
			{
				Name:       "Renderable",
				Kind:       CompletionKindInterface,
				Detail:     "type Renderable interface { Render() *Node }",
				InsertText: "Renderable",
				Doc:        "Interface implemented by components that render to a virtual DOM Node tree.",
			},
			{
				Name:       "Node",
				Kind:       CompletionKindStruct,
				Detail:     "type Node struct",
				InsertText: "Node",
				Doc:        "Virtual DOM node representation.",
			},
			{
				Name:       "NewStore(initial)",
				Kind:       CompletionKindFunction,
				Detail:     "func NewStore[T any](initial T) *Store[T]",
				InsertText: "NewStore(${1:initialValue})",
				Doc:        "Creates a reactive observable state store.",
			},
			{
				Name:       "UseState(initial)",
				Kind:       CompletionKindFunction,
				Detail:     "func UseState[T any](initial T) (T, func(T))",
				InsertText: "UseState(${1:initialValue})",
				Doc:        "State hook for components.",
			},
			{
				Name:       "UseEffect(fn, deps...)",
				Kind:       CompletionKindFunction,
				Detail:     "func UseEffect(fn func() func(), deps ...any)",
				InsertText: "UseEffect(func() func() {\n\t$0\n\treturn nil\n})",
				Doc:        "Lifecycle side-effect hook for components.",
			},
			{
				Name:       "C(c)",
				Kind:       CompletionKindFunction,
				Detail:     "func C(c Renderable) *Node",
				InsertText: "C(${1:component})",
				Doc:        "Wraps a component into a Virtual DOM node.",
			},
			{
				Name:       "Text(s)",
				Kind:       CompletionKindFunction,
				Detail:     "func Text(s string) *Node",
				InsertText: "Text(${1:string})",
				Doc:        "Creates a text Virtual DOM node.",
			},
			{
				Name:       "Fragment(children...)",
				Kind:       CompletionKindFunction,
				Detail:     "func Fragment(children ...*Node) *Node",
				InsertText: "Fragment(${1:children...})",
				Doc:        "Creates a fragment node containing multiple children.",
			},
		},
	},
	{
		Name:        "metadata",
		ImportPath:  "github.com/misbakhul29/goks/pkg/metadata",
		Description: "Page & Layout metadata, SEO, and OpenGraph configuration for GoKS",
		Members: []PackageMember{
			{
				Name:       "Metadata",
				Kind:       CompletionKindStruct,
				Detail:     "type Metadata struct",
				InsertText: "Metadata{\n\tTitle:       \"${1:Title}\",\n\tDescription: \"${2:Description}\",\n}",
				Doc:        "Page and layout metadata configuration struct.",
			},
			{
				Name:       "OpenGraph",
				Kind:       CompletionKindStruct,
				Detail:     "type OpenGraph struct",
				InsertText: "OpenGraph{\n\tTitle:       \"${1:Title}\",\n\tDescription: \"${2:Description}\",\n}",
				Doc:        "OpenGraph social preview metadata configuration.",
			},
		},
	},
	{
		Name:        "html",
		ImportPath:  "github.com/misbakhul29/goks/pkg/html",
		Description: "HTML builder functions for GoKS",
		Members: []PackageMember{
			{Name: "Div", Kind: CompletionKindFunction, Detail: "func Div(children ...any) *Builder", InsertText: "Div($1)"},
			{Name: "Span", Kind: CompletionKindFunction, Detail: "func Span(children ...any) *Builder", InsertText: "Span($1)"},
			{Name: "Button", Kind: CompletionKindFunction, Detail: "func Button(children ...any) *Builder", InsertText: "Button($1)"},
			{Name: "A", Kind: CompletionKindFunction, Detail: "func A(children ...any) *Builder", InsertText: "A($1)"},
			{Name: "P", Kind: CompletionKindFunction, Detail: "func P(children ...any) *Builder", InsertText: "P($1)"},
			{Name: "H1", Kind: CompletionKindFunction, Detail: "func H1(children ...any) *Builder", InsertText: "H1($1)"},
		},
	},
	{
		Name:        "client",
		ImportPath:  "github.com/misbakhul29/goks/runtime/client",
		Description: "GoKS client-side WASM runtime and browser interop",
		Members: []PackageMember{
			{Name: "Console", Kind: CompletionKindVariable, Detail: "var Console struct { Log, Warn, Error }", InsertText: "Console.Log($1)"},
			{Name: "Fetch", Kind: CompletionKindFunction, Detail: "func Fetch(url string) (string, error)", InsertText: "Fetch(\"${1:url}\")"},
			{Name: "Mount", Kind: CompletionKindFunction, Detail: "func Mount(selector string, root Renderable) *App", InsertText: "Mount(\"${1:#app}\", ${2:root})"},
		},
	},
	{
		Name:        "orm",
		ImportPath:  "github.com/misbakhul29/goks/pkg/orm",
		Description: "Lightweight SQLite / Postgres ORM for GoKS",
		Members: []PackageMember{
			{Name: "Open", Kind: CompletionKindFunction, Detail: "func Open(driver, dsn string) (*DB, error)", InsertText: "Open(\"${1:sqlite}\", \"${2:file.db}\")"},
			{Name: "DB", Kind: CompletionKindStruct, Detail: "type DB struct", InsertText: "DB"},
		},
	},
	{
		Name:        "auth",
		ImportPath:  "github.com/misbakhul29/goks/pkg/auth",
		Description: "Authentication, session management, and JWT helpers for GoKS",
		Members: []PackageMember{
			{Name: "GenerateToken", Kind: CompletionKindFunction, Detail: "func GenerateToken(...) (string, error)", InsertText: "GenerateToken($1)"},
			{Name: "ValidateToken", Kind: CompletionKindFunction, Detail: "func ValidateToken(...) (*Claims, error)", InsertText: "ValidateToken($1)"},
		},
	},
}

func isInsideImport(lines []string, currentLine int) bool {
	inImport := false
	for i := 0; i <= currentLine && i < len(lines); i++ {
		l := strings.TrimSpace(lines[i])
		if strings.HasPrefix(l, "import (") {
			inImport = true
		} else if inImport && strings.HasPrefix(l, ")") {
			inImport = false
		}
	}
	return inImport
}

func getImportCompletions() []CompletionItem {
	var items []CompletionItem
	for _, pkg := range goksPackages {
		items = append(items, CompletionItem{
			Label:  fmt.Sprintf("%q", pkg.ImportPath),
			Kind:   CompletionKindModule,
			Detail: fmt.Sprintf("package %s — %s", pkg.Name, pkg.Description),
			Documentation: MarkupContent{
				Kind:  "markdown",
				Value: fmt.Sprintf("### `%s`\n\n%s\n\n```go\nimport %q\n```", pkg.ImportPath, pkg.Description, pkg.ImportPath),
			},
			InsertText:       fmt.Sprintf("%q", pkg.ImportPath),
			InsertTextFormat: InsertFormatPlainText,
			SortText:         "00_" + pkg.Name,
		})
	}
	return items
}

func getPackageMemberCompletions(content, pkgName string) []CompletionItem {
	var items []CompletionItem
	for _, pkg := range goksPackages {
		if strings.EqualFold(pkg.Name, pkgName) {
			autoImport := computeAutoImportEdit(content, pkg.ImportPath)
			for _, m := range pkg.Members {
				items = append(items, CompletionItem{
					Label:  m.Name,
					Kind:   m.Kind,
					Detail: fmt.Sprintf("%s (%s)", m.Detail, pkg.ImportPath),
					Documentation: MarkupContent{
						Kind:  "markdown",
						Value: fmt.Sprintf("### `%s.%s`\n`%s`\n\n%s\n\n```go\nimport %q\n```", pkg.Name, m.Name, m.Detail, m.Doc, pkg.ImportPath),
					},
					InsertText:          m.InsertText,
					InsertTextFormat:    InsertFormatSnippet,
					SortText:            "00_" + m.Name,
					AdditionalTextEdits: autoImport,
				})
			}
			break
		}
	}
	return items
}

func getGoSymbolCompletions(content, prefix string) []CompletionItem {
	var query string
	if m := identifierRegex.FindStringSubmatch(prefix); len(m) > 0 {
		query = strings.ToLower(m[1])
	}

	var items []CompletionItem

	for _, pkg := range goksPackages {
		matchPkg := query == "" || strings.HasPrefix(strings.ToLower(pkg.Name), query) || strings.Contains(strings.ToLower(pkg.Name), query)

		autoImport := computeAutoImportEdit(content, pkg.ImportPath)

		if matchPkg {
			// Suggest package name
			items = append(items, CompletionItem{
				Label:  pkg.Name,
				Kind:   CompletionKindModule,
				Detail: fmt.Sprintf("import %q", pkg.ImportPath),
				Documentation: MarkupContent{
					Kind:  "markdown",
					Value: fmt.Sprintf("### Package `%s`\n\n%s\n\n```go\nimport %q\n```", pkg.Name, pkg.Description, pkg.ImportPath),
				},
				InsertText:          pkg.Name,
				InsertTextFormat:    InsertFormatPlainText,
				SortText:            "00_" + pkg.Name,
				AdditionalTextEdits: autoImport,
			})
		}

		// Suggest members with package prefix
		for _, member := range pkg.Members {
			fullLabel := pkg.Name + "." + member.Name
			cleanName := strings.TrimSuffix(member.Name, "()")
			cleanName = strings.TrimSuffix(cleanName, "(opt)")
			cleanName = strings.TrimSuffix(cleanName, "(path)")
			cleanName = strings.TrimSuffix(cleanName, "(initial)")
			cleanName = strings.TrimSuffix(cleanName, "(parts...)")

			matchMember := matchPkg || (query != "" && (strings.Contains(strings.ToLower(fullLabel), query) || strings.HasPrefix(strings.ToLower(cleanName), query)))

			if matchMember {
				items = append(items, CompletionItem{
					Label:  fullLabel,
					Kind:   member.Kind,
					Detail: fmt.Sprintf("%s (%s)", member.Detail, pkg.ImportPath),
					Documentation: MarkupContent{
						Kind:  "markdown",
						Value: fmt.Sprintf("### `%s`\n`%s`\n\n%s\n\n```go\nimport %q\n```", fullLabel, member.Detail, member.Doc, pkg.ImportPath),
					},
					InsertText:          pkg.Name + "." + member.InsertText,
					InsertTextFormat:    InsertFormatSnippet,
					SortText:            "01_" + pkg.Name + "_" + cleanName,
					AdditionalTextEdits: autoImport,
				})
			}
		}
	}

	return items
}

func computeAutoImportEdit(content, importPath string) []TextEdit {
	if strings.Contains(content, `"`+importPath+`"`) {
		return nil // already imported
	}

	lines := strings.Split(content, "\n")

	// Check for import ( ... ) block
	importOpenLine := -1
	for i, l := range lines {
		trimmed := strings.TrimSpace(l)
		if strings.HasPrefix(trimmed, "import (") {
			importOpenLine = i
			break
		}
	}

	if importOpenLine != -1 {
		// Insert right after `import (`
		return []TextEdit{
			{
				Range: Range{
					Start: Position{Line: importOpenLine + 1, Character: 0},
					End:   Position{Line: importOpenLine + 1, Character: 0},
				},
				NewText: fmt.Sprintf("\t%q\n", importPath),
			},
		}
	}

	// Check for single import "..."
	for i, l := range lines {
		trimmed := strings.TrimSpace(l)
		if strings.HasPrefix(trimmed, "import \"") && strings.HasSuffix(trimmed, "\"") {
			oldImport := strings.TrimPrefix(trimmed, "import ")
			return []TextEdit{
				{
					Range: Range{
						Start: Position{Line: i, Character: 0},
						End:   Position{Line: i + 1, Character: 0},
					},
					NewText: fmt.Sprintf("import (\n\t%s\n\t%q\n)\n", oldImport, importPath),
				},
			}
		}
	}

	// No import block at all: insert after package statement
	for i, l := range lines {
		if strings.HasPrefix(strings.TrimSpace(l), "package ") {
			return []TextEdit{
				{
					Range: Range{
						Start: Position{Line: i + 1, Character: 0},
						End:   Position{Line: i + 1, Character: 0},
					},
					NewText: fmt.Sprintf("\nimport %q\n", importPath),
				},
			}
		}
	}

	return nil
}

func isInsideTag(prefix string) bool {
	lastOpen := strings.LastIndex(prefix, "<")
	lastClose := strings.LastIndex(prefix, ">")
	return lastOpen > lastClose
}

func getTagCompletions(docURI string) []CompletionItem {

	items := []CompletionItem{
		// Common layout & structure
		{
			Label:            "div",
			Kind:             CompletionKindSnippet,
			Detail:           "HTML <div> element",
			InsertText:       "div class=\"$1\">$0</div>",
			InsertTextFormat: InsertFormatSnippet,
			SortText:         "01_div",
		},
		{
			Label:            "html",
			Kind:             CompletionKindSnippet,
			Detail:           "Root <html> element",
			InsertText:       "html lang=\"${1:en}\">\n\t<body class=\"$2\">\n\t\t$0\n\t</body>\n</html>",
			InsertTextFormat: InsertFormatSnippet,
			SortText:         "02_html",
		},
		{
			Label:            "head",
			Kind:             CompletionKindSnippet,
			Detail:           "HTML <head> element",
			InsertText:       "head>\n\t$0\n</head>",
			InsertTextFormat: InsertFormatSnippet,
			SortText:         "03_head",
		},
		{
			Label:            "body",
			Kind:             CompletionKindSnippet,
			Detail:           "HTML <body> element",
			InsertText:       "body class=\"$1\">\n\t$0\n</body>",
			InsertTextFormat: InsertFormatSnippet,
			SortText:         "04_body",
		},
		{
			Label:            "main",
			Kind:             CompletionKindSnippet,
			Detail:           "HTML <main> element",
			InsertText:       "main class=\"$1\">\n\t$0\n</main>",
			InsertTextFormat: InsertFormatSnippet,
			SortText:         "05_main",
		},
		{
			Label:            "section",
			Kind:             CompletionKindSnippet,
			Detail:           "HTML <section> element",
			InsertText:       "section class=\"$1\">\n\t$0\n</section>",
			InsertTextFormat: InsertFormatSnippet,
			SortText:         "06_section",
		},
		{
			Label:            "nav",
			Kind:             CompletionKindSnippet,
			Detail:           "HTML <nav> element",
			InsertText:       "nav class=\"$1\">\n\t$0\n</nav>",
			InsertTextFormat: InsertFormatSnippet,
			SortText:         "07_nav",
		},
		{
			Label:            "header",
			Kind:             CompletionKindSnippet,
			Detail:           "HTML <header> element",
			InsertText:       "header class=\"$1\">\n\t$0\n</header>",
			InsertTextFormat: InsertFormatSnippet,
			SortText:         "08_header",
		},
		{
			Label:            "footer",
			Kind:             CompletionKindSnippet,
			Detail:           "HTML <footer> element",
			InsertText:       "footer class=\"$1\">\n\t$0\n</footer>",
			InsertTextFormat: InsertFormatSnippet,
			SortText:         "09_footer",
		},
		// Interactive & form elements
		{
			Label:            "button",
			Kind:             CompletionKindSnippet,
			Detail:           "HTML <button> element with onClick",
			InsertText:       "button class=\"$1\" onClick={$2}>$0</button>",
			InsertTextFormat: InsertFormatSnippet,
			SortText:         "10_button",
		},
		{
			Label:            "input",
			Kind:             CompletionKindSnippet,
			Detail:           "HTML <input /> element",
			InsertText:       "input type=\"${1:text}\" class=\"$2\" placeholder=\"$3\" />",
			InsertTextFormat: InsertFormatSnippet,
			SortText:         "11_input",
		},
		{
			Label:            "form",
			Kind:             CompletionKindSnippet,
			Detail:           "HTML <form> element with onSubmit",
			InsertText:       "form onSubmit={$1} class=\"$2\">\n\t$0\n</form>",
			InsertTextFormat: InsertFormatSnippet,
			SortText:         "12_form",
		},
		{
			Label:            "a",
			Kind:             CompletionKindSnippet,
			Detail:           "HTML <a> link element",
			InsertText:       "a href=\"${1:#}\" class=\"$2\">$0</a>",
			InsertTextFormat: InsertFormatSnippet,
			SortText:         "13_a",
		},
		{
			Label:            "img",
			Kind:             CompletionKindSnippet,
			Detail:           "HTML <img /> image element",
			InsertText:       "img src=\"$1\" alt=\"$2\" class=\"$3\" />",
			InsertTextFormat: InsertFormatSnippet,
			SortText:         "14_img",
		},
		// Text elements
		{
			Label:            "p",
			Kind:             CompletionKindSnippet,
			Detail:           "HTML <p> paragraph element",
			InsertText:       "p class=\"$1\">$0</p>",
			InsertTextFormat: InsertFormatSnippet,
			SortText:         "15_p",
		},
		{
			Label:            "span",
			Kind:             CompletionKindSnippet,
			Detail:           "HTML <span> inline element",
			InsertText:       "span class=\"$1\">$0</span>",
			InsertTextFormat: InsertFormatSnippet,
			SortText:         "16_span",
		},
		{
			Label:            "h1",
			Kind:             CompletionKindSnippet,
			Detail:           "HTML <h1> heading",
			InsertText:       "h1 class=\"$1\">$0</h1>",
			InsertTextFormat: InsertFormatSnippet,
			SortText:         "17_h1",
		},
		{
			Label:            "h2",
			Kind:             CompletionKindSnippet,
			Detail:           "HTML <h2> heading",
			InsertText:       "h2 class=\"$1\">$0</h2>",
			InsertTextFormat: InsertFormatSnippet,
			SortText:         "18_h2",
		},
		{
			Label:            "h3",
			Kind:             CompletionKindSnippet,
			Detail:           "HTML <h3> heading",
			InsertText:       "h3 class=\"$1\">$0</h3>",
			InsertTextFormat: InsertFormatSnippet,
			SortText:         "19_h3",
		},
		// Metadata & head elements
		{
			Label:            "title",
			Kind:             CompletionKindSnippet,
			Detail:           "HTML <title> element",
			InsertText:       "title>$0</title>",
			InsertTextFormat: InsertFormatSnippet,
			SortText:         "20_title",
		},
		{
			Label:            "meta",
			Kind:             CompletionKindSnippet,
			Detail:           "HTML <meta /> element",
			InsertText:       "meta name=\"$1\" content=\"$2\" />",
			InsertTextFormat: InsertFormatSnippet,
			SortText:         "21_meta",
		},
		{
			Label:            "link",
			Kind:             CompletionKindSnippet,
			Detail:           "HTML <link /> stylesheet/asset",
			InsertText:       "link rel=\"${1:stylesheet}\" href=\"$2\" />",
			InsertTextFormat: InsertFormatSnippet,
			SortText:         "22_link",
		},
		{
			Label:            "script",
			Kind:             CompletionKindSnippet,
			Detail:           "HTML <script> element",
			InsertText:       "script src=\"$1\"></script>",
			InsertTextFormat: InsertFormatSnippet,
			SortText:         "23_script",
		},
		// Lists
		{
			Label:            "ul",
			Kind:             CompletionKindSnippet,
			Detail:           "HTML <ul> unordered list",
			InsertText:       "ul class=\"$1\">\n\t<li>$0</li>\n</ul>",
			InsertTextFormat: InsertFormatSnippet,
			SortText:         "24_ul",
		},
		{
			Label:            "li",
			Kind:             CompletionKindSnippet,
			Detail:           "HTML <li> list item",
			InsertText:       "li class=\"$1\">$0</li>",
			InsertTextFormat: InsertFormatSnippet,
			SortText:         "25_li",
		},
	}

	// Also discover custom components in workspace
	compItems := discoverComponents(docURI)
	items = append(items, compItems...)

	return items
}

func getAttributeCompletions() []CompletionItem {
	return []CompletionItem{
		{
			Label:            "class",
			Kind:             CompletionKindProperty,
			Detail:           "CSS classes (Tailwind / styling)",
			InsertText:       "class=\"$1\"",
			InsertTextFormat: InsertFormatSnippet,
			SortText:         "01_class",
		},
		{
			Label:            "id",
			Kind:             CompletionKindProperty,
			Detail:           "Element ID",
			InsertText:       "id=\"$1\"",
			InsertTextFormat: InsertFormatSnippet,
			SortText:         "02_id",
		},
		{
			Label:            "onClick",
			Kind:             CompletionKindEvent,
			Detail:           "Click event handler",
			InsertText:       "onClick={$1}",
			InsertTextFormat: InsertFormatSnippet,
			SortText:         "03_onClick",
		},
		{
			Label:            "onChange",
			Kind:             CompletionKindEvent,
			Detail:           "Change event handler",
			InsertText:       "onChange={$1}",
			InsertTextFormat: InsertFormatSnippet,
			SortText:         "04_onChange",
		},
		{
			Label:            "onSubmit",
			Kind:             CompletionKindEvent,
			Detail:           "Form submit event handler",
			InsertText:       "onSubmit={$1}",
			InsertTextFormat: InsertFormatSnippet,
			SortText:         "05_onSubmit",
		},
		{
			Label:            "onInput",
			Kind:             CompletionKindEvent,
			Detail:           "Input event handler",
			InsertText:       "onInput={$1}",
			InsertTextFormat: InsertFormatSnippet,
			SortText:         "06_onInput",
		},
		{
			Label:            "href",
			Kind:             CompletionKindProperty,
			Detail:           "Link destination URL",
			InsertText:       "href=\"$1\"",
			InsertTextFormat: InsertFormatSnippet,
			SortText:         "07_href",
		},
		{
			Label:            "src",
			Kind:             CompletionKindProperty,
			Detail:           "Resource source URL",
			InsertText:       "src=\"$1\"",
			InsertTextFormat: InsertFormatSnippet,
			SortText:         "08_src",
		},
		{
			Label:            "type",
			Kind:             CompletionKindProperty,
			Detail:           "Element type (text, password, email, submit...)",
			InsertText:       "type=\"${1:text}\"",
			InsertTextFormat: InsertFormatSnippet,
			SortText:         "09_type",
		},
		{
			Label:            "placeholder",
			Kind:             CompletionKindProperty,
			Detail:           "Input placeholder text",
			InsertText:       "placeholder=\"$1\"",
			InsertTextFormat: InsertFormatSnippet,
			SortText:         "10_placeholder",
		},
		{
			Label:            "value",
			Kind:             CompletionKindProperty,
			Detail:           "Input value",
			InsertText:       "value=\"$1\"",
			InsertTextFormat: InsertFormatSnippet,
			SortText:         "11_value",
		},
		{
			Label:            "disabled",
			Kind:             CompletionKindProperty,
			Detail:           "Disabled state",
			InsertText:       "disabled",
			InsertTextFormat: InsertFormatPlainText,
			SortText:         "12_disabled",
		},
		{
			Label:            "name",
			Kind:             CompletionKindProperty,
			Detail:           "Form field name",
			InsertText:       "name=\"$1\"",
			InsertTextFormat: InsertFormatSnippet,
			SortText:         "13_name",
		},
		{
			Label:            "key",
			Kind:             CompletionKindProperty,
			Detail:           "Virtual DOM reconciliation key",
			InsertText:       "key=\"$1\"",
			InsertTextFormat: InsertFormatSnippet,
			SortText:         "14_key",
		},
	}
}

func discoverComponents(docURI string) []CompletionItem {
	var items []CompletionItem

	// Extract directory from URI (file://...)
	cleanPath := strings.TrimPrefix(docURI, "file://")
	dir := filepath.Dir(cleanPath)

	// Search up for workspace root containing go.mod
	for i := 0; i < 5; i++ {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	compDir := filepath.Join(dir, "components")
	if _, err := os.Stat(compDir); os.IsNotExist(err) {
		compDir = filepath.Join(dir, "app", "components")
	}

	_ = filepath.Walk(compDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".gox") || strings.HasSuffix(path, ".go") {
			data, err := os.ReadFile(path)
			if err != nil {
				return nil
			}
			// Find type [Name] struct
			re := regexp.MustCompile(`type\s+([A-Z][a-zA-Z0-9_]*)\s+struct`)
			matches := re.FindAllStringSubmatch(string(data), -1)
			for _, m := range matches {
				compName := m[1]
				items = append(items, CompletionItem{
					Label:            "c." + compName,
					Kind:             CompletionKindClass,
					Detail:           "Custom GoKS Component",
					InsertText:       "c." + compName + " $0/>",
					InsertTextFormat: InsertFormatSnippet,
					SortText:         "00_c_" + compName,
				})
			}
		}
		return nil
	})

	return items
}
