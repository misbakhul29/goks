package component_test

import (
	"strings"
	"testing"

	"github.com/misbakhul29/goks/pkg/component"
)

type StaticPage struct{}

func (s *StaticPage) Render() *component.Node {
	return component.H("html", nil,
		component.H("head", nil, component.H("title", nil, component.Text("Static Blog"))),
		component.H("body", nil,
			component.H("h1", nil, component.Text("Welcome to my blog")),
			component.H("p", nil, component.Text("Pure HTML with zero JavaScript and zero WASM.")),
		),
	)
}

type InteractiveIsland struct {
	component.ClientBase
}

func (i *InteractiveIsland) Render() *component.Node {
	return component.H("div", component.Props{
		"class": "island-counter",
	}, component.H("button", component.Props{
		"onClick": func() {},
	}, component.Text("+1")))
}

type HybridPage struct{}

func (h *HybridPage) Render() *component.Node {
	return component.H("div", component.Props{"class": "container"},
		component.H("h1", nil, component.Text("Hybrid Article")),
		component.C(&InteractiveIsland{}),
	)
}

func TestIslands_ZeroWASMForStaticPage(t *testing.T) {
	staticTree := component.C(&StaticPage{})

	if component.NeedsHydration(staticTree) {
		t.Fatal("expected static page to require zero hydration (NeedsHydration == false)")
	}

	htmlOut := component.RenderToString(staticTree)
	if strings.Contains(htmlOut, "wasm_exec.js") || strings.Contains(htmlOut, "app.wasm") {
		t.Fatalf("static page SSR output should contain 0 WASM script tags, got:\n%s", htmlOut)
	}
}

func TestIslands_HydrationRequiredForInteractiveIsland(t *testing.T) {
	islandTree := component.C(&InteractiveIsland{})

	if !component.NeedsHydration(islandTree) {
		t.Fatal("expected interactive island to require hydration (NeedsHydration == true)")
	}
}

func TestIslands_HydrationRequiredForHybridPage(t *testing.T) {
	hybridTree := component.C(&HybridPage{})

	if !component.NeedsHydration(hybridTree) {
		t.Fatal("expected hybrid page with interactive island to require hydration")
	}
}
