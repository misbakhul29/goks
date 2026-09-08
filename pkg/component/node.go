// Package component provides the core UI component system for GoKS.
// Components are written in Go and compiled to WebAssembly to run in the browser.
package component

import "reflect"

// NodeType represents the type of a virtual DOM node.
type NodeType int

const (
	NodeTypeElement  NodeType = iota // HTML element, e.g. <div>
	NodeTypeText                     // Text node
	NodeTypeFragment                 // Fragment (multiple root nodes)
	NodeTypeComponent               // A GoKS component node
)

// Props is a map of properties/attributes for a node or component.
type Props map[string]any

// Node represents a virtual DOM node.
type Node struct {
	Type       NodeType
	Tag        string    // HTML tag name (for NodeTypeElement)
	Text       string    // Text content (for NodeTypeText)
	Props      Props     // Attributes, event handlers, etc.
	Children   []*Node   // Child nodes
	Key        string    // Optional reconciler key for stable identity
	Component  Renderable // Set if NodeTypeComponent
}

// Renderable is implemented by anything that can render itself to a Node tree.
type Renderable interface {
	Render() *Node
}

// FuncComponent is a functional component that implements Renderable.
type FuncComponent func() *Node

func (f FuncComponent) Render() *Node {
	return f()
}

// FC wraps a function into a Node.
func FC(f func() *Node) *Node {
	return C(FuncComponent(f))
}

// H creates a new element Node (hyperscript-style).
//
//	goks.H("div", Props{"class": "container"}, child1, child2)
func H(tag string, props Props, children ...*Node) *Node {
	if props == nil {
		props = Props{}
	}
	return &Node{
		Type:     NodeTypeElement,
		Tag:      tag,
		Props:    props,
		Children: children,
	}
}

// Text creates a text Node.
func Text(content string) *Node {
	return &Node{
		Type: NodeTypeText,
		Text: content,
	}
}

// Fragment creates a fragment node containing multiple children.
func Fragment(children ...*Node) *Node {
	return &Node{
		Type:     NodeTypeFragment,
		Children: children,
	}
}

// WithKey attaches a reconciler key to a node, used for stable list rendering.
func (n *Node) WithKey(key string) *Node {
	n.Key = key
	return n
}

// Class appends or sets the CSS class attribute for chaining.
func (n *Node) Class(c string) *Node {
	if n.Props == nil {
		n.Props = Props{}
	}
	// Append class if already exists
	if existing, ok := n.Props["class"]; ok {
		n.Props["class"] = existing.(string) + " " + c
	} else {
		n.Props["class"] = c
	}
	return n
}

// ID sets the HTML id attribute for chaining.
func (n *Node) ID(id string) *Node {
	if n.Props == nil {
		n.Props = Props{}
	}
	n.Props["id"] = id
	return n
}

// Attr sets a generic HTML attribute for chaining.
func (n *Node) Attr(key string, val any) *Node {
	if n.Props == nil {
		n.Props = Props{}
	}
	n.Props[key] = val
	return n
}

// On sets an event listener (e.g., "onClick") for chaining.
func (n *Node) On(event string, handler any) *Node {
	if n.Props == nil {
		n.Props = Props{}
	}
	n.Props[event] = handler
	return n
}

// Render implements the Renderable interface so a *Node can be passed directly.
func (n *Node) Render() *Node {
	return n
}

// C creates a component node from a Renderable.
func C(r Renderable) *Node {
	return &Node{
		Type:      NodeTypeComponent,
		Component: r,
	}
}

// Expand recursively evaluates NodeTypeComponent nodes and returns a tree of real DOM representable nodes.
// It wires up re-render callbacks and manages the Fiber tree for functional hooks.
func Expand(n *Node, appRerender func(), currentFiber *FiberNode) *Node {
	if n == nil {
		return nil
	}

	if n.Type == NodeTypeComponent && n.Component != nil {
		// Wire up re-render for class components
		if cb, ok := n.Component.(interface{ bindRerender(func()) }); ok {
			cb.bindRerender(appRerender)
		}

		// Manage Fiber for functional components
		var childFiber *FiberNode
		if currentFiber != nil {
			compType := reflect.TypeOf(n.Component)
			var funcPtr uintptr
			
			// Extract function pointer if it's a FuncComponent to differentiate between different functions
			if compType == reflect.TypeOf(FuncComponent(nil)) {
				funcPtr = reflect.ValueOf(n.Component).Pointer()
			}

			childFiber = currentFiber.getOrCreateChild(currentFiber.ChildIndex)
			if childFiber.CompType != nil {
				if childFiber.CompType != compType || childFiber.FuncPtr != funcPtr {
					// Component type changed at this position! Reset the fiber node!
					childFiber = &FiberNode{}
					currentFiber.Children[currentFiber.ChildIndex] = childFiber
				}
			}
			childFiber.CompType = compType
			childFiber.FuncPtr = funcPtr
			childFiber.AppRerender = appRerender
			currentFiber.ChildIndex++

			// Reset hook and child index for the current render pass
			childFiber.HookIndex = 0
			childFiber.ChildIndex = 0
		}

		var prevFiber *FiberNode
		if childFiber != nil {
			prevFiber = setActiveFiber(childFiber)
		}

		// Recursively render the component
		rendered := n.Component.Render()
		child := Expand(rendered, appRerender, childFiber)
		
		if childFiber != nil {
			setActiveFiber(prevFiber)
		}

		if child != nil {
			// Attach the component instance to the resulting element for lifecycle hooks
			child.Component = n.Component
			return child
		}
		return nil
	}

	// Expand children and flatten fragments.
	// For standard DOM nodes, we don't advance the component fiber.
	var expandedChildren []*Node
	for _, c := range n.Children {
		child := Expand(c, appRerender, currentFiber)
		if child == nil {
			continue
		}
		if child.Type == NodeTypeFragment {
			// Flatten fragment children directly into the parent
			expandedChildren = append(expandedChildren, child.Children...)
		} else {
			expandedChildren = append(expandedChildren, child)
		}
	}
	n.Children = expandedChildren

	return n
}
