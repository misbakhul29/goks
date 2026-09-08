package component_test

import (
	"testing"

	"github.com/misbakhulmunir/goks/pkg/component"
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
