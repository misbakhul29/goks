package html

import (
	"github.com/misbakhul29/goks/pkg/component"
)

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
		case string:
			// Automatically wrap strings as Text nodes
			n.Children = append(n.Children, component.Text(v))
		case *component.Node:
			if v != nil {
				n.Children = append(n.Children, v)
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
