package lsp

import (
	"encoding/xml"
	"fmt"
	"go/parser"
	"go/scanner"
	"go/token"
	"io"
	"regexp"
	"strings"

	"github.com/misbakhul29/goks/internal/compiler"
)

// ValidateDocument parses a .gox document and returns any diagnostics (syntax/JSX errors).
func ValidateDocument(uri, content string) []Diagnostic {
	var diagnostics []Diagnostic

	lines := strings.Split(content, "\n")

	// 1. Validate JSX syntax inside all `return (` ... `)` blocks
	jsxDiags := validateJSXBlocks(content, lines)
	diagnostics = append(diagnostics, jsxDiags...)

	// 2. If no fatal JSX errors, transpile to Go and check with Go's standard AST parser
	if len(jsxDiags) == 0 {
		transpiled, err := compiler.Transpile(content)
		if err != nil {
			diagnostics = append(diagnostics, Diagnostic{
				Range: Range{
					Start: Position{Line: 0, Character: 0},
					End:   Position{Line: 0, Character: 1},
				},
				Severity: SeverityError,
				Source:   "gox-compiler",
				Message:  fmt.Sprintf("Transpile error: %v", err),
			})
		} else {
			goDiags := validateGoSyntax(transpiled, lines)
			diagnostics = append(diagnostics, goDiags...)
		}
	}

	return diagnostics
}

type tagInfo struct {
	name      string
	line      int
	character int
}

func validateJSXBlocks(content string, lines []string) []Diagnostic {
	var diagnostics []Diagnostic

	idx := 0
	for {
		start := strings.Index(content[idx:], "return (")
		if start == -1 {
			break
		}
		start += idx

		// Find matching parenthesis
		openCount := 1
		end := start + 8
		for end < len(content) {
			if content[end] == '(' {
				openCount++
			} else if content[end] == ')' {
				openCount--
				if openCount == 0 {
					break
				}
			}
			end++
		}

		if openCount != 0 || end >= len(content) {
			line, col := offsetToLineCol(content, start)
			diagnostics = append(diagnostics, Diagnostic{
				Range: Range{
					Start: Position{Line: line, Character: col},
					End:   Position{Line: line, Character: col + 8},
				},
				Severity: SeverityError,
				Source:   "gox",
				Message:  "Unmatched parenthesis for 'return (' block",
			})
			break
		}

		jsxContent := content[start+8 : end]
		jsxOffset := start + 8

		blockDiags := validateJSXContent(jsxContent, jsxOffset, content)
		diagnostics = append(diagnostics, blockDiags...)

		idx = end + 1
	}

	return diagnostics
}

var selfClosingTags = map[string]bool{
	"area": true, "base": true, "br": true, "col": true, "embed": true,
	"hr": true, "img": true, "input": true, "link": true, "meta": true,
	"param": true, "source": true, "track": true, "wbr": true,
}

func validateJSXContent(jsx string, baseOffset int, fullContent string) []Diagnostic {
	var diagnostics []Diagnostic

	// Check for unclosed attribute strings or expressions
	if diag := checkUnclosedAttributes(jsx, baseOffset, fullContent); diag != nil {
		diagnostics = append(diagnostics, *diag)
	}

	// Sanitize naked & so text like "A & B" doesn't fail XML entity parser
	sanitizedJSX := sanitizeNakedAmpersands(jsx)

	// Use XML decoder with strict checking for tag balance
	d := xml.NewDecoder(strings.NewReader(sanitizedJSX))
	d.Strict = true
	d.Entity = xml.HTMLEntity

	var tagStack []tagInfo
	hasParseError := false

	for {
		tokenOffset := int(d.InputOffset())
		t, err := d.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			hasParseError = true
			line, col := offsetToLineCol(fullContent, baseOffset+tokenOffset)
			diagnostics = append(diagnostics, Diagnostic{
				Range: Range{
					Start: Position{Line: line, Character: col},
					End:   Position{Line: line, Character: col + 1},
				},
				Severity: SeverityError,
				Source:   "gox-xml",
				Message:  cleanXMLError(err.Error()),
			})
			break
		}

		switch tok := t.(type) {
		case xml.StartElement:
			tagName := tok.Name.Local
			if tok.Name.Space != "" {
				tagName = tok.Name.Space + "." + tok.Name.Local
			}

			if !selfClosingTags[strings.ToLower(tagName)] {
				line, col := offsetToLineCol(fullContent, baseOffset+tokenOffset)
				tagStack = append(tagStack, tagInfo{
					name:      tagName,
					line:      line,
					character: col,
				})
			}

		case xml.EndElement:
			tagName := tok.Name.Local
			if tok.Name.Space != "" {
				tagName = tok.Name.Space + "." + tok.Name.Local
			}

			if selfClosingTags[strings.ToLower(tagName)] {
				continue // Do not pop from stack for void/self-closing tags (e.g. <br />, <img />)
			}

			if len(tagStack) == 0 {
				line, col := offsetToLineCol(fullContent, baseOffset+tokenOffset)
				diagnostics = append(diagnostics, Diagnostic{
					Range: Range{
						Start: Position{Line: line, Character: col},
						End:   Position{Line: line, Character: col + len(tagName) + 3},
					},
					Severity: SeverityError,
					Source:   "gox",
					Message:  fmt.Sprintf("Unexpected closing tag </%s> without matching opening tag", tagName),
				})
			} else {
				last := tagStack[len(tagStack)-1]
				tagStack = tagStack[:len(tagStack)-1]
				if last.name != tagName {
					line, col := offsetToLineCol(fullContent, baseOffset+tokenOffset)
					diagnostics = append(diagnostics, Diagnostic{
						Range: Range{
							Start: Position{Line: line, Character: col},
							End:   Position{Line: line, Character: col + len(tagName) + 3},
						},
						Severity: SeverityError,
						Source:   "gox",
						Message:  fmt.Sprintf("Mismatched closing tag: expected </%s>, got </%s>", last.name, tagName),
					})
				}
			}
		}
	}

	// Any unclosed tags remaining on the stack (only if parsed cleanly to EOF)
	if !hasParseError {
		for _, unclosed := range tagStack {
			diagnostics = append(diagnostics, Diagnostic{
				Range: Range{
					Start: Position{Line: unclosed.line, Character: unclosed.character},
					End:   Position{Line: unclosed.line, Character: unclosed.character + len(unclosed.name) + 2},
				},
				Severity: SeverityError,
				Source:   "gox",
				Message:  fmt.Sprintf("Unclosed tag <%s>", unclosed.name),
			})
		}
	}

	return diagnostics
}

func sanitizeNakedAmpersands(jsx string) string {
	var sb strings.Builder
	sb.Grow(len(jsx))

	for i := 0; i < len(jsx); i++ {
		if jsx[i] == '&' {
			rest := jsx[i+1:]
			isEntity := false
			for _, ent := range []string{"amp;", "lt;", "gt;", "quot;", "apos;"} {
				if strings.HasPrefix(rest, ent) {
					isEntity = true
					break
				}
			}
			if !isEntity && strings.HasPrefix(rest, "#") {
				semicolon := strings.Index(rest, ";")
				if semicolon > 1 && semicolon < 10 {
					isEntity = true
				}
			}

			if isEntity {
				sb.WriteByte('&')
			} else {
				sb.WriteByte(' ') // Replace naked & with space to keep exact byte length
			}
		} else {
			sb.WriteByte(jsx[i])
		}
	}
	return sb.String()
}

func checkUnclosedAttributes(jsx string, baseOffset int, fullContent string) *Diagnostic {
	lines := strings.Split(jsx, "\n")
	for i, line := range lines {
		// Look for unclosed quote in attr
		inQuote := false
		quoteChar := rune(0)
		for _, r := range line {
			if r == '"' || r == '\'' {
				if !inQuote {
					inQuote = true
					quoteChar = r
				} else if r == quoteChar {
					inQuote = false
				}
			}
		}
		if inQuote {
			lineNum, _ := offsetToLineCol(fullContent, baseOffset)
			targetLine := lineNum + i
			return &Diagnostic{
				Range: Range{
					Start: Position{Line: targetLine, Character: 0},
					End:   Position{Line: targetLine, Character: len(line)},
				},
				Severity: SeverityError,
				Source:   "gox",
				Message:  "Unclosed string literal in tag attribute",
			}
		}
	}
	return nil
}

func validateGoSyntax(transpiled string, origLines []string) []Diagnostic {
	var diagnostics []Diagnostic

	fset := token.NewFileSet()
	_, err := parser.ParseFile(fset, "", transpiled, parser.AllErrors)
	if err == nil {
		return nil
	}

	if errorList, ok := err.(scanner.ErrorList); ok {
		for _, e := range errorList {
			lineIdx := e.Pos.Line - 1
			if lineIdx < 0 {
				lineIdx = 0
			}
			colIdx := e.Pos.Column - 1
			if colIdx < 0 {
				colIdx = 0
			}

			lineLen := 1
			if lineIdx < len(origLines) {
				lineLen = len(origLines[lineIdx])
			}

			diagnostics = append(diagnostics, Diagnostic{
				Range: Range{
					Start: Position{Line: lineIdx, Character: colIdx},
					End:   Position{Line: lineIdx, Character: lineLen},
				},
				Severity: SeverityError,
				Source:   "go",
				Message:  e.Msg,
			})
		}
	}

	return diagnostics
}

func offsetToLineCol(content string, offset int) (int, int) {
	if offset > len(content) {
		offset = len(content)
	}
	sub := content[:offset]
	lines := strings.Split(sub, "\n")
	line := len(lines) - 1
	col := len(lines[line])
	return line, col
}

var xmlPrefixRegex = regexp.MustCompile(`^XML syntax error on line \d+:\s*`)

func cleanXMLError(errMsg string) string {
	cleaned := xmlPrefixRegex.ReplaceAllString(errMsg, "")
	cleaned = strings.TrimPrefix(cleaned, "XML syntax error: ")
	cleaned = strings.TrimSpace(cleaned)
	if len(cleaned) == 0 {
		return "Malformed JSX/HTML syntax"
	}
	return strings.ToUpper(cleaned[:1]) + cleaned[1:]
}
