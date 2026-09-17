package component_test

// Regression tests for WASM renderer bugs fixed in v1.1.1–v1.1.5.
//
// renderer.go is guarded by `//go:build js && wasm` so we cannot call it
// directly in a native test binary. What we can test is the virtual DOM
// layer (node.go, reconciler.go) that feeds the renderer, covering every
// behavioural change that was the root cause of those bugs.
//
// Each group is labelled with the version that introduced the fix.

import (
	"strings"
	"testing"

	"github.com/misbakhul29/goks/pkg/component"
)

// -----------------------------------------------------------------------
// v1.1.1 — func(string) event listener + value property sync
// -----------------------------------------------------------------------

// NeedsHydration must detect onInput: func(string) as requiring WASM.
func TestRegression_V1_1_1_FuncStringListenerNeedsHydration(t *testing.T) {
	node := component.H("input", component.Props{
		"onInput": func(val string) {},
	})
	if !component.NeedsHydration(node) {
		t.Fatal("func(string) onInput must require hydration")
	}
}

// onChange: func(string) on a select element must also require hydration.
func TestRegression_V1_1_1_FuncStringOnChangeNeedsHydration(t *testing.T) {
	node := component.H("select", component.Props{
		"onChange": func(val string) {},
	}, component.H("option", component.Props{"value": "a"}, component.Text("A")))
	if !component.NeedsHydration(node) {
		t.Fatal("func(string) onChange must require hydration")
	}
}

// value prop must appear in SSR output so the server-side and client-side
// initial renders agree (value property sync regression).
func TestRegression_V1_1_1_ValuePropRenderedInSSR(t *testing.T) {
	node := component.H("input", component.Props{
		"type":  "text",
		"value": "hello",
	})
	out := component.RenderToString(node)
	if !strings.Contains(out, `value="hello"`) {
		t.Fatalf("expected value attribute in SSR output, got: %s", out)
	}
}

// -----------------------------------------------------------------------
// v1.1.2 — LSP false positive on multiline tag attributes
// (pure virtual DOM: nothing to test here; LSP tests live in internal/lsp)
// Placeholder so the gap is documented.
// -----------------------------------------------------------------------

// -----------------------------------------------------------------------
// v1.1.3 — component.Any handles slices of nodes, renderables, values
// -----------------------------------------------------------------------

// Already covered in TestAny in component_test.go. Adding edge cases.

func TestRegression_V1_1_3_AnyNilSliceEntry(t *testing.T) {
	nodes := []*component.Node{
		component.H("p", nil, component.Text("first")),
		nil, // must be silently dropped
		component.H("p", nil, component.Text("third")),
	}
	frag := component.Any(nodes)
	if frag == nil {
		t.Fatal("Any([]*Node with nil entry) must not return nil")
	}
	if frag.Type != component.NodeTypeFragment {
		t.Fatalf("expected Fragment, got %v", frag.Type)
	}
	if len(frag.Children) != 2 {
		t.Fatalf("expected 2 children (nil dropped), got %d", len(frag.Children))
	}
}

func TestRegression_V1_1_3_AnyEmptySlice(t *testing.T) {
	frag := component.Any([]*component.Node{})
	if frag == nil {
		t.Fatal("Any(empty []*Node) must not return nil")
	}
	if frag.Type != component.NodeTypeFragment {
		t.Fatalf("expected Fragment for empty slice, got %v", frag.Type)
	}
	if len(frag.Children) != 0 {
		t.Fatalf("expected 0 children, got %d", len(frag.Children))
	}
}

func TestRegression_V1_1_3_AnySliceOfRenderables(t *testing.T) {
	type SimpleComp struct{}
	// SimpleComp is not used here; use FuncComponent instead so we don't
	// need to implement Renderable on a local type.
	r1 := component.FC(func() *component.Node { return component.Text("r1") })
	r2 := component.FC(func() *component.Node { return component.Text("r2") })

	// Build a []any that mixes nodes and func-components.
	mixed := []any{r1, r2, "plain text"}
	frag := component.Any(mixed)
	if frag == nil || frag.Type != component.NodeTypeFragment {
		t.Fatalf("expected Fragment from []any, got %v", frag)
	}
	if len(frag.Children) != 3 {
		t.Fatalf("expected 3 children, got %d", len(frag.Children))
	}
}

// -----------------------------------------------------------------------
// v1.1.4 — nil parent guards in reconciler
// -----------------------------------------------------------------------

// Reconcile(old, nil) must produce a single PatchRemove and not panic.
func TestRegression_V1_1_4_ReconcileOldToNilNoPanic(t *testing.T) {
	old := component.H("div", nil, component.Text("child"))
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Reconcile(old, nil) panicked: %v", r)
		}
	}()
	patches := component.Reconcile(old, nil)
	if len(patches) != 1 || patches[0].Type != component.PatchRemove {
		t.Fatalf("expected exactly 1 PatchRemove, got %v", patches)
	}
}

// Reconcile(nil, new) must produce a single PatchCreate and not panic.
func TestRegression_V1_1_4_ReconcileNilToNewNoPanic(t *testing.T) {
	newNode := component.H("section", nil, component.H("p", nil, component.Text("hi")))
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Reconcile(nil, new) panicked: %v", r)
		}
	}()
	patches := component.Reconcile(nil, newNode)
	if len(patches) != 1 || patches[0].Type != component.PatchCreate {
		t.Fatalf("expected exactly 1 PatchCreate, got %v", patches)
	}
}

// ReconcileChildren with more old children than new must remove extras
// and not produce an index-out-of-bounds or nil-pointer panic.
func TestRegression_V1_1_4_ReconcileChildrenShrinkNoPanic(t *testing.T) {
	old := []*component.Node{
		component.H("li", nil, component.Text("a")),
		component.H("li", nil, component.Text("b")),
		component.H("li", nil, component.Text("c")),
	}
	newNodes := []*component.Node{
		component.H("li", nil, component.Text("a")),
	}
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("ReconcileChildren shrink panicked: %v", r)
		}
	}()
	patches := component.ReconcileChildren(old, newNodes)
	removes := 0
	for _, p := range patches {
		if p.Type == component.PatchRemove {
			removes++
		}
	}
	if removes != 2 {
		t.Fatalf("expected 2 PatchRemove patches, got %d", removes)
	}
}

// -----------------------------------------------------------------------
// v1.1.5 — SVG namespace + setAttribute for class
// -----------------------------------------------------------------------

// SVG elements must be constructed correctly at the virtual DOM level.
// The namespace itself is a renderer concern (js/wasm), but we can verify
// that the node tree is correctly built with the right tags and props so
// the renderer receives the right input.
func TestRegression_V1_1_5_SVGNodeTree(t *testing.T) {
	svg := component.H("svg", component.Props{
		"viewBox": "0 0 100 100",
		"class":   "icon",
	},
		component.H("circle", component.Props{
			"cx": "50",
			"cy": "50",
			"r":  "40",
		}),
		component.H("path", component.Props{
			"d":     "M10 10 L90 90",
			"class": "line",
		}),
	)

	if svg.Tag != "svg" {
		t.Fatalf("expected tag 'svg', got %q", svg.Tag)
	}
	if svg.Props["class"] != "icon" {
		t.Fatalf("expected class 'icon' on svg, got %q", svg.Props["class"])
	}
	if len(svg.Children) != 2 {
		t.Fatalf("expected 2 SVG children, got %d", len(svg.Children))
	}
	circle := svg.Children[0]
	if circle.Tag != "circle" {
		t.Fatalf("expected 'circle' child, got %q", circle.Tag)
	}
	path := svg.Children[1]
	if path.Props["class"] != "line" {
		t.Fatalf("expected class 'line' on path, got %q", path.Props["class"])
	}
}

// class prop must survive a reconciler prop-change round-trip.
// This exercises the same code path that setAttribute("class", ...) covers.
func TestRegression_V1_1_5_ClassPropReconcile(t *testing.T) {
	old := component.H("div", component.Props{"class": "foo"})
	newNode := component.H("div", component.Props{"class": "foo bar"})
	patches := component.Reconcile(old, newNode)
	if len(patches) != 1 {
		t.Fatalf("expected 1 PatchUpdate for class change, got %d", len(patches))
	}
	if patches[0].Type != component.PatchUpdate {
		t.Fatalf("expected PatchUpdate, got %v", patches[0].Type)
	}
	if patches[0].NewNode.Props["class"] != "foo bar" {
		t.Fatalf("expected new class 'foo bar', got %q", patches[0].NewNode.Props["class"])
	}
}

// SSR output must contain the class attribute for SVG-like elements.
func TestRegression_V1_1_5_SVGClassInSSR(t *testing.T) {
	node := component.H("svg", component.Props{
		"class":   "my-icon",
		"viewBox": "0 0 24 24",
	})
	out := component.RenderToString(node)
	if !strings.Contains(out, `class="my-icon"`) {
		t.Fatalf("expected class attribute in SSR output, got: %s", out)
	}
}
