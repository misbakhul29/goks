package lsp

import (
	"strings"
)

// FormatDocument formats a .gox document by formatting Go code and indenting JSX blocks cleanly.
func FormatDocument(content string) []TextEdit {
	lines := strings.Split(content, "\n")
	formattedLines := make([]string, len(lines))
	copy(formattedLines, lines)

	idx := 0
	for {
		start := strings.Index(content[idx:], "return (")
		if start == -1 {
			break
		}
		start += idx

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
			break
		}

		startLine, _ := offsetToLineCol(content, start)
		endLine, _ := offsetToLineCol(content, end)

		// Base indentation for JSX content is return line indentation + 1 tab
		baseIndent := getLineIndentation(lines[startLine]) + "\t"

		jsxContent := content[start+8 : end]
		formattedJSX := formatJSXBlock(jsxContent, baseIndent)

		// Replace the lines between startLine and endLine
		formattedBlock := lines[startLine][:strings.Index(lines[startLine], "return (")+8] + "\n" +
			formattedJSX + "\n" +
			getLineIndentation(lines[startLine]) + ")"

		newLines := strings.Split(formattedBlock, "\n")
		// Splice into formattedLines
		prefix := formattedLines[:startLine]
		suffix := formattedLines[endLine+1:]
		formattedLines = append(append(prefix, newLines...), suffix...)

		// Rebuild content to keep tracking
		content = strings.Join(formattedLines, "\n")
		idx = start + len(formattedBlock)
	}

	resultText := strings.Join(formattedLines, "\n")
	if resultText == content {
		return nil
	}

	return []TextEdit{
		{
			Range: Range{
				Start: Position{Line: 0, Character: 0},
				End:   Position{Line: len(lines), Character: 0},
			},
			NewText: resultText,
		},
	}
}

func getLineIndentation(line string) string {
	var indent strings.Builder
	for _, r := range line {
		if r == '\t' || r == ' ' {
			indent.WriteRune(r)
		} else {
			break
		}
	}
	return indent.String()
}

func formatJSXBlock(jsx, baseIndent string) string {
	rawLines := strings.Split(jsx, "\n")
	var cleaned []string
	for _, l := range rawLines {
		trimmed := strings.TrimSpace(l)
		if trimmed != "" {
			cleaned = append(cleaned, trimmed)
		}
	}

	var sb strings.Builder
	depth := 0

	for _, line := range cleaned {
		isClosing := strings.HasPrefix(line, "</")
		isSelfClosing := strings.HasSuffix(line, "/>")
		isOpening := strings.HasPrefix(line, "<") && !isClosing && !isSelfClosing

		if isClosing && depth > 0 {
			depth--
		}

		for i := 0; i < depth; i++ {
			sb.WriteString("\t")
		}
		sb.WriteString(baseIndent)
		sb.WriteString(line)
		sb.WriteString("\n")

		if isOpening {
			depth++
		}
	}

	return strings.TrimRight(sb.String(), "\n")
}
