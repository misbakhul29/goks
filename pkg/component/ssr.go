package component

import (
	"fmt"
	"html"
	"sort"
	"strings"
)

// RenderToString converts a virtual DOM node into an HTML string.
// This runs on the backend server for Server-Side Rendering (SSR).
func RenderToString(node *Node) string {
	if node == nil {
		return ""
	}

	switch node.Type {
	case NodeTypeText:
		// Escape HTML special characters to prevent XSS.
		return html.EscapeString(node.Text)
	case NodeTypeFragment:
		var sb strings.Builder
		for _, child := range node.Children {
			sb.WriteString(RenderToString(child))
		}
		return sb.String()
	case NodeTypeComponent:
		if node.Component != nil {
			return RenderToString(node.Component.Render())
		}
		return ""
	default: // NodeTypeElement
		var sb strings.Builder
		sb.WriteString("<")
		sb.WriteString(node.Tag)

		// Sort attribute keys for deterministic SSR output across all platforms
		keys := make([]string, 0, len(node.Props))
		for k := range node.Props {
			if strings.HasPrefix(k, "on") || k == "innerHTML" {
				continue // Skip event listeners and innerHTML on element tag
			}
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {
			v := node.Props[k]
			attrName := k
			if attrName == "className" {
				attrName = "class"
			}
			// Escape attribute values to prevent XSS injection.
			escapedVal := html.EscapeString(fmt.Sprintf("%v", v))
			sb.WriteString(fmt.Sprintf(" %s=\"%s\"", attrName, escapedVal))
		}
		sb.WriteString(">")

		// Handle self-closing tags
		switch node.Tag {
		case "area", "base", "br", "col", "embed", "hr", "img", "input", "link", "meta", "param", "source", "track", "wbr":
			return sb.String()
		}

		// Handle innerHTML
		if html, ok := node.Props["innerHTML"]; ok {
			sb.WriteString(fmt.Sprintf("%v", html))
		} else {
			for _, child := range node.Children {
				sb.WriteString(RenderToString(child))
			}
		}

		sb.WriteString("</")
		sb.WriteString(node.Tag)
		sb.WriteString(">")
		return sb.String()
	}
}
