package lsp

import (
	"fmt"
	"regexp"
	"strings"
)

var elementDocs = map[string]string{
	"html":    "The root element of an HTML document. All other elements must be descendants of this element.",
	"head":    "Contains machine-readable information (metadata) about the document, like its title, scripts, and style sheets.",
	"body":    "Represents the content of an HTML document. There can be only one `<body>` element in a document.",
	"title":   "Defines the document's title that is shown in a browser's title bar or a page's tab.",
	"meta":    "Represents metadata that cannot be represented by other HTML meta-related elements, like `<base>`, `<link>`, `<script>`, `<style>` or `<title>`.",
	"link":    "Specifies relationships between the current document and an external resource, such as stylesheets or favicons.",
	"script":  "Used to embed executable code or data; typically used to embed or refer to JavaScript code.",
	"style":   "Contains style information for a document, or part of a document.",
	"div":     "The generic container for flow content. It has no effect on the content or layout until styled in some way using CSS.",
	"main":    "Represents the dominant content of the `<body>` of a document.",
	"section": "Represents a generic standalone section of a document, which doesn't have a more specific semantic element to represent it.",
	"nav":     "Represents a section of a page whose purpose is to provide navigation links.",
	"header":  "Represents introductory content, typically a group of introductory or navigational aids.",
	"footer":  "Represents a footer for its nearest ancestor sectioning content or sectioning root element.",
	"article": "Represents a self-contained composition in a document, page, application, or site, which is intended to be independently distributable.",
	"aside":   "Represents a portion of a document whose content is only indirectly related to the document's main content.",
	"h1":      "Represents the highest level section heading on a page.",
	"h2":      "Represents a level 2 section heading.",
	"h3":      "Represents a level 3 section heading.",
	"h4":      "Represents a level 4 section heading.",
	"h5":      "Represents a level 5 section heading.",
	"h6":      "Represents a level 6 section heading.",
	"p":       "Represents a paragraph of text.",
	"span":    "A generic inline container for phrasing content, which does not inherently represent anything.",
	"button":  "An interactive element activated by a user with a mouse, keyboard, finger, voice command, or other assistive technology.",
	"input":   "Used to create interactive controls for web-based forms in order to accept data from the user.",
	"form":    "Represents a document section containing interactive controls for submitting information.",
	"label":   "Represents a caption for an item in a user interface.",
	"a":       "Together with its `href` attribute, creates a hyperlink to web pages, files, email addresses, or locations in the same page.",
	"img":     "Embeds an image into the document.",
	"ul":      "Represents an unordered list of items, typically rendered as a bulleted list.",
	"ol":      "Represents an ordered list of items, typically rendered as a numbered list.",
	"li":      "Used to represent an item in a list.",
	"table":   "Represents tabular data — that is, information presented in a two-dimensional table comprised of rows and columns.",
}

var attrDocs = map[string]string{
	"class":       "A space-separated list of the case-sensitive classes of the element. Used for CSS styling (e.g. Tailwind CSS).",
	"id":          "Defines a unique identifier (ID) which must be unique in the whole document.",
	"style":       "Contains CSS styling declarations to be applied to the element.",
	"key":         "GoKS Virtual DOM reconciliation key. Used for stable list rendering and minimal DOM patching.",
	"onClick":     "GoKS event listener invoked when the user clicks on the element.",
	"onChange":    "GoKS event listener invoked when the value of an input element changes.",
	"onSubmit":    "GoKS event listener invoked when a `<form>` is submitted.",
	"onInput":     "GoKS event listener invoked synchronously when the value of an `<input>` or `<textarea>` element is changed.",
	"href":        "The URL that the hyperlink points to.",
	"src":         "The URL of the image, video, audio, or script to be embedded.",
	"type":        "The type of control to render (for `<input>`: text, password, checkbox, submit, etc.).",
	"placeholder": "A hint to the user of what can be entered in the control.",
	"value":       "The initial or current value of the control.",
	"disabled":    "A boolean attribute indicating that the user cannot interact with the control.",
	"name":        "The name of the control, submitted along with the form data.",
}

// GetHover returns Markdown hover documentation for the token under the cursor.
func GetHover(content string, pos Position) *Hover {
	lines := strings.Split(content, "\n")
	if pos.Line >= len(lines) {
		return nil
	}

	line := lines[pos.Line]
	word, startIdx, endIdx := getWordAndRangeAtPosition(line, pos.Character)
	if word == "" {
		return nil
	}

	// Check if cursor is inside an HTML tag (< ... >)
	prefix := line[:startIdx]
	lastOpen := strings.LastIndex(prefix, "<")
	lastClose := strings.LastIndex(prefix, ">")

	insideTag := lastOpen != -1 && (lastClose == -1 || lastOpen > lastClose)
	if !insideTag {
		// Cursor is outside a tag (in plain text content or Go code)
		return nil
	}

	// Inside a tag: determine if word is the tag name (right after < or </)
	trimmedAfterOpen := strings.TrimLeft(prefix[lastOpen+1:], "/")
	trimmedAfterOpen = strings.TrimSpace(trimmedAfterOpen)
	isTagName := trimmedAfterOpen == "" || trimmedAfterOpen == word

	if isTagName {
		lowerWord := strings.ToLower(word)
		if doc, ok := elementDocs[lowerWord]; ok {
			return &Hover{
				Contents: MarkupContent{
					Kind:  "markdown",
					Value: fmt.Sprintf("### `<%s>` (HTML Element)\n\n%s", lowerWord, doc),
				},
			}
		}

		if strings.HasPrefix(word, "c.") || (len(word) > 1 && strings.ToUpper(word[:1]) == word[:1]) {
			return &Hover{
				Contents: MarkupContent{
					Kind:  "markdown",
					Value: fmt.Sprintf("### `<%s>` (GoKS Component)\n\nCustom reusable component rendered via `component.C(...)`.", word),
				},
			}
		}
		return nil
	}

	// Word is an attribute name inside the tag
	if doc, ok := attrDocs[word]; ok {
		return &Hover{
			Contents: MarkupContent{
				Kind:  "markdown",
				Value: fmt.Sprintf("### `%s` (Attribute)\n\n%s", word, doc),
			},
		}
	}

	_ = endIdx
	return nil
}

var wordRegex = regexp.MustCompile(`[a-zA-Z0-9_.:-]+`)

func getWordAndRangeAtPosition(line string, charIdx int) (string, int, int) {
	matches := wordRegex.FindAllStringIndex(line, -1)
	for _, m := range matches {
		if charIdx >= m[0] && charIdx <= m[1] {
			return line[m[0]:m[1]], m[0], m[1]
		}
	}
	return "", 0, 0
}
