package html

import (
	"github.com/misbakhul29/goks/pkg/component"
)

// SafeHTML represents pre-sanitized, trusted raw HTML content.
// CAUTION: Passing unescaped user input can lead to Cross-Site Scripting (XSS).
// Always sanitize user-provided HTML before wrapping it with SafeHTML.
type SafeHTML string

// Raw creates a container component with raw unescaped HTML content.
// WARNING: Only use with trusted, sanitized markup to prevent XSS.
func Raw(trustedHTML string) *component.Node {
	return &component.Node{
		Type: component.NodeTypeElement,
		Tag:  "span",
		Props: component.Props{
			"innerHTML": trustedHTML,
		},
	}
}

// Element is the core builder function that creates a component Node
// and appends children appropriately.
func Element(tag string, children ...any) *component.Node {
	n := &component.Node{
		Type:  component.NodeTypeElement,
		Tag:   tag,
		Props: component.Props{},
	}

	for _, child := range children {
		switch v := child.(type) {
		case SafeHTML:
			// Inject trusted raw HTML
			n.Props["innerHTML"] = string(v)
		case string:
			// Automatically wrap strings as Text nodes (HTML-escaped during rendering)
			n.Children = append(n.Children, component.Text(v))
		case *component.Node:
			if v != nil {
				if v.Type == component.NodeTypeFragment {
					n.Children = append(n.Children, v.Children...)
				} else {
					n.Children = append(n.Children, v)
				}
			}
		case component.Renderable:
			if v != nil {
				n.Children = append(n.Children, component.C(v))
			}
		case []any:
			// Flatten slice of any
			for _, sub := range v {
				if subStr, ok := sub.(string); ok {
					n.Children = append(n.Children, component.Text(subStr))
				} else if subNode, ok := sub.(*component.Node); ok && subNode != nil {
					n.Children = append(n.Children, subNode)
				} else if subRend, ok := sub.(component.Renderable); ok && subRend != nil {
					n.Children = append(n.Children, component.C(subRend))
				}
			}
		case []*component.Node:
			for _, sub := range v {
				if sub != nil {
					n.Children = append(n.Children, sub)
				}
			}
		}
	}
	return n
}
