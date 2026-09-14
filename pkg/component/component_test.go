package component_test

import (
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/misbakhul29/goks/pkg/component"
)

// -----------------------------------------------------------------------
// Node creation tests
// -----------------------------------------------------------------------

func TestH_CreatesElementNode(t *testing.T) {
	n := component.H("div", component.Props{"class": "container"})
	if n.Type != component.NodeTypeElement {
		t.Fatalf("expected NodeTypeElement, got %v", n.Type)
	}
	if n.Tag != "div" {
		t.Fatalf("expected tag 'div', got %q", n.Tag)
	}
	if n.Props["class"] != "container" {
		t.Fatalf("expected class 'container'")
	}
}

func TestText_CreatesTextNode(t *testing.T) {
	n := component.Text("hello")
	if n.Type != component.NodeTypeText {
		t.Fatalf("expected NodeTypeText")
	}
	if n.Text != "hello" {
		t.Fatalf("expected text 'hello', got %q", n.Text)
	}
}

func TestFragment_CreatesFragmentNode(t *testing.T) {
	n := component.Fragment(component.Text("a"), component.Text("b"))
	if n.Type != component.NodeTypeFragment {
		t.Fatalf("expected NodeTypeFragment")
	}
	if len(n.Children) != 2 {
		t.Fatalf("expected 2 children, got %d", len(n.Children))
	}
}

func TestH_NilProps(t *testing.T) {
	n := component.H("span", nil, component.Text("hi"))
	if n.Props == nil {
		t.Fatal("props should not be nil even when nil is passed")
	}
}

func TestWithKey(t *testing.T) {
	n := component.H("li", nil).WithKey("item-1")
	if n.Key != "item-1" {
		t.Fatalf("expected key 'item-1', got %q", n.Key)
	}
}

// -----------------------------------------------------------------------
// Reconciler tests
// -----------------------------------------------------------------------

func TestReconcile_NoChange(t *testing.T) {
	old := component.H("div", component.Props{"class": "a"})
	new := component.H("div", component.Props{"class": "a"})
	patches := component.Reconcile(old, new)
	if len(patches) != 0 {
		t.Fatalf("expected no patches for identical trees, got %d", len(patches))
	}
}

func TestReconcile_TextChange(t *testing.T) {
	old := component.Text("hello")
	new := component.Text("world")
	patches := component.Reconcile(old, new)
	if len(patches) != 1 {
		t.Fatalf("expected 1 patch, got %d", len(patches))
	}
	if patches[0].Type != component.PatchText {
		t.Fatalf("expected PatchText, got %v", patches[0].Type)
	}
}

func TestReconcile_TagChange(t *testing.T) {
	old := component.H("div", nil)
	new := component.H("span", nil)
	patches := component.Reconcile(old, new)
	if len(patches) != 1 {
		t.Fatalf("expected 1 patch, got %d", len(patches))
	}
	if patches[0].Type != component.PatchReplace {
		t.Fatalf("expected PatchReplace, got %v", patches[0].Type)
	}
}

func TestReconcile_PropChange(t *testing.T) {
	old := component.H("div", component.Props{"class": "a"})
	new := component.H("div", component.Props{"class": "b"})
	patches := component.Reconcile(old, new)
	if len(patches) != 1 {
		t.Fatalf("expected 1 patch, got %d", len(patches))
	}
	if patches[0].Type != component.PatchUpdate {
		t.Fatalf("expected PatchUpdate, got %v", patches[0].Type)
	}
}

func TestReconcile_NewChild(t *testing.T) {
	old := component.H("div", nil)
	new := component.H("div", nil, component.Text("child"))
	patches := component.Reconcile(old, new)
	if len(patches) != 1 {
		t.Fatalf("expected 1 patch for new child, got %d", len(patches))
	}
	if patches[0].Type != component.PatchCreate {
		t.Fatalf("expected PatchCreate, got %v", patches[0].Type)
	}
}

func TestReconcile_RemovedChild(t *testing.T) {
	old := component.H("div", nil, component.Text("child"))
	new := component.H("div", nil)
	patches := component.Reconcile(old, new)
	if len(patches) != 1 {
		t.Fatalf("expected 1 patch for removed child, got %d", len(patches))
	}
	if patches[0].Type != component.PatchRemove {
		t.Fatalf("expected PatchRemove, got %v", patches[0].Type)
	}
}

func TestReconcile_OldNil(t *testing.T) {
	new := component.H("div", nil)
	patches := component.Reconcile(nil, new)
	if len(patches) != 1 || patches[0].Type != component.PatchCreate {
		t.Fatalf("expected PatchCreate when old is nil")
	}
}

func TestReconcile_NewNil(t *testing.T) {
	old := component.H("div", nil)
	patches := component.Reconcile(old, nil)
	if len(patches) != 1 || patches[0].Type != component.PatchRemove {
		t.Fatalf("expected PatchRemove when new is nil")
	}
}

func TestAny(t *testing.T) {
	node := component.H("div", nil)
	if component.Any(node) != node {
		t.Fatalf("expected Any(*Node) to return the original node")
	}

	text := component.Any("hello")
	if text.Type != component.NodeTypeText || text.Text != "hello" {
		t.Fatalf("expected Any(string) to create text node")
	}

	num := component.Any(42)
	if num.Type != component.NodeTypeText || num.Text != "42" {
		t.Fatalf("expected Any(int) to create text node with stringified number")
	}

	if component.Any(nil) != nil {
		t.Fatalf("expected Any(nil) to return nil")
	}
}

func TestRenderToString_EscapesTextAndAttrs(t *testing.T) {
	node := component.H("a", component.Props{
		"href":  "javascript:alert(\"xss\")",
		"title": "<script>alert(1)</script>",
	}, component.Text("<img src=x onerror=alert(1)>"))

	out := component.RenderToString(node)
	if strings.Contains(out, "<script>") || strings.Contains(out, "<img src=x") {
		t.Fatalf("un-escaped HTML tags found in SSR output: %s", out)
	}
	if !strings.Contains(out, "&lt;img") || !strings.Contains(out, "&lt;script&gt;") {
		t.Fatalf("expected HTML entities in SSR output, got: %s", out)
	}
}

func TestNeedsHydration(t *testing.T) {
	// Static node: no event handlers
	staticNode := component.H("div", component.Props{"class": "container"}, component.Text("Static Content"))
	if component.NeedsHydration(staticNode) {
		t.Errorf("expected static node to not need hydration")
	}

	// Interactive node with onClick
	interactiveNode := component.H("button", component.Props{
		"onClick": func() {},
	}, component.Text("Click me"))
	if !component.NeedsHydration(interactiveNode) {
		t.Errorf("expected interactive node with onClick to need hydration")
	}

	// Interactive node with onInput func(string)
	inputNode := component.H("input", component.Props{
		"onInput": func(val string) {},
	})
	if !component.NeedsHydration(inputNode) {
		t.Errorf("expected interactive node with onInput to need hydration")
	}

	// Nested interactive child
	parent := component.H("div", nil, interactiveNode)
	if !component.NeedsHydration(parent) {
		t.Errorf("expected parent with interactive child to need hydration")
	}
}

type StatefulComponent struct {
	Initial int
}

func (s *StatefulComponent) Render() *component.Node {
	val, _ := component.UseState(s.Initial)
	return component.H("div", nil, component.Text(fmt.Sprintf("Val: %d", val)))
}

func TestUseState_SSRFallback(t *testing.T) {
	// Rendering a stateful component during SSR should safely render initial value without panicking
	comp := &StatefulComponent{Initial: 42}
	out := component.RenderToString(component.C(comp))
	if !strings.Contains(out, "Val: 42") {
		t.Fatalf("expected SSR to render initial value 42, got: %s", out)
	}
}

func TestUseState_HookInvariantViolation(t *testing.T) {
	// Simulate a parent fiber whose child 0 already has an int hook at slot 0
	fiber := &component.FiberNode{
		Children: []*component.FiberNode{
			{Hooks: []any{100}}, // slot 0 was initialized as int
		},
	}

	// We temporarily set activeFiber and try to call UseState[string] at slot 0
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic on hook invariant violation")
		}
		errMsg := fmt.Sprintf("%v", r)
		if !strings.Contains(errMsg, "GoKS Hook Invariant Violation") {
			t.Fatalf("unexpected panic message: %s", errMsg)
		}
	}()

	// Simulate component execution
	component.Expand(component.FC(func() *component.Node {
		// Wrong type at slot 0
		component.UseState("mismatched type")
		return nil
	}), func() {}, fiber)
}

func TestRenderToString_DeterministicAttributes(t *testing.T) {
	node := component.H("div", component.Props{
		"z-index": "10",
		"class":   "box",
		"id":      "main",
		"alpha":   "first",
	}, component.Text("hello"))

	first := component.RenderToString(node)
	for i := 0; i < 20; i++ {
		repeat := component.RenderToString(node)
		if repeat != first {
			t.Fatalf("SSR attribute order changed between runs:\nFirst:  %s\nRepeat: %s", first, repeat)
		}
	}
}

func TestRenderToString_ConcurrentIsolation(t *testing.T) {
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		id := i
		go func() {
			defer wg.Done()
			comp := &StatefulComponent{Initial: id}
			out := component.RenderToString(component.C(comp))
			expected := fmt.Sprintf("Val: %d", id)
			if !strings.Contains(out, expected) {
				t.Errorf("concurrent SSR leaked state or rendered wrong value. Expected %s, got %s", expected, out)
			}
		}()
	}
	wg.Wait()
}
