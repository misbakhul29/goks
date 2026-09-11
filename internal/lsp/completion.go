package lsp

import (
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

	// 1. Tag completion context: user typed `<` or `<tag`
	if tagMatch := tagContextRegex.FindStringSubmatch(prefix); len(tagMatch) > 0 {
		return getTagCompletions(docURI)
	}

	// 2. Attribute completion context: user is inside an open tag `<div ...`
	if isInsideTag(prefix) {
		return getAttributeCompletions()
	}

	return nil
}

var tagContextRegex = regexp.MustCompile(`<([a-zA-Z0-9_.]*)$`)

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
