package component_test

import (
	"strings"
	"testing"

	"github.com/misbakhul29/goks/pkg/component"
)

// StaticCard represents a purely static server component.
type StaticCard struct {
	component.ComponentBase
	Title string
}

func (c *StaticCard) Render() *component.Node {
	return component.H("div", component.Props{"class": "card"},
		component.H("h2", nil, component.Text(c.Title)),
		component.H("p", nil, component.Text("Static marketing content with zero JS/WASM overhead.")),
	)
}

// InteractiveCounter represents a client island requiring WASM hydration.
type InteractiveCounter struct {
	component.ComponentBase
	component.ClientBase
}

func (c *InteractiveCounter) Render() *component.Node {
	return component.H("button", component.Props{
		"onClick": func() {},
	}, component.Text("Click me"))
}

func TestBundleInvariant_ZeroWASMOnStaticComponent(t *testing.T) {
	staticComp := component.C(&StaticCard{Title: "About Us"})
	expanded := component.Expand(staticComp, func() {}, nil)

	if component.NeedsHydration(expanded) {
		t.Fatal("invariant violation: static component must require 0 KB WASM (NeedsHydration == false)")
	}

	html := component.RenderToString(expanded)
	if strings.Contains(html, "app.wasm") || strings.Contains(html, "wasm_exec.js") {
		t.Fatalf("invariant violation: static component HTML must not reference WASM assets, got: %s", html)
	}
}

func TestBundleInvariant_WASMRequiredOnInteractiveIsland(t *testing.T) {
	islandComp := component.C(&InteractiveCounter{})
	expanded := component.Expand(islandComp, func() {}, nil)

	if !component.NeedsHydration(expanded) {
		t.Fatal("interactive island component must require hydration (NeedsHydration == true)")
	}
}

func TestBundleInvariant_HybridPageIslandsIsolation(t *testing.T) {
	// A page containing multiple static sections and exactly one isolated island
	page := component.H("main", nil,
		component.H("header", nil, component.Text("Static Header")),
		component.C(&StaticCard{Title: "Feature 1"}),
		component.C(&StaticCard{Title: "Feature 2"}),
		component.H("footer", nil, component.Text("Static Footer")),
	)

	expandedStaticPage := component.Expand(page, func() {}, nil)
	if component.NeedsHydration(expandedStaticPage) {
		t.Fatal("fully static composite page must not require hydration")
	}

	// Now append the interactive island
	pageWithIsland := component.H("main", nil,
		component.H("header", nil, component.Text("Static Header")),
		component.C(&StaticCard{Title: "Feature 1"}),
		component.C(&InteractiveCounter{}),
		component.H("footer", nil, component.Text("Static Footer")),
	)

	expandedHybridPage := component.Expand(pageWithIsland, func() {}, nil)
	if !component.NeedsHydration(expandedHybridPage) {
		t.Fatal("hybrid page containing an interactive island must require hydration")
	}
}
