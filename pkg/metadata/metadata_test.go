package metadata

import (
	"strings"
	"testing"

	"github.com/misbakhul29/goks/pkg/component"
)

type mockLayout struct {
	component.ComponentBase
}

func (m *mockLayout) Metadata() Metadata {
	return Metadata{
		Title:         "GoKS",
		TitleTemplate: "%s | GoKS Framework",
		Description:   "Fullstack Go framework",
		Keywords:      []string{"go", "wasm"},
		OpenGraph: &OpenGraph{
			SiteName: "GoKS Site",
			Type:     "website",
		},
	}
}

func (m *mockLayout) Render() *component.Node {
	return component.Text("layout")
}

type mockPage struct {
	component.ComponentBase
}

func (m *mockPage) Metadata() Metadata {
	return Metadata{
		Title:       "Documentation",
		Description: "Read the GoKS docs",
		OpenGraph: &OpenGraph{
			Title: "GoKS Docs",
		},
	}
}

func (m *mockPage) Render() *component.Node {
	return component.Text("page")
}

func TestMerge(t *testing.T) {
	parent := (&mockLayout{}).Metadata()
	child := (&mockPage{}).Metadata()

	merged := Merge(parent, child)

	if merged.Title != "Documentation | GoKS Framework" {
		t.Errorf("expected title 'Documentation | GoKS Framework', got %q", merged.Title)
	}

	if merged.Description != "Read the GoKS docs" {
		t.Errorf("expected description 'Read the GoKS docs', got %q", merged.Description)
	}

	if len(merged.Keywords) != 2 || merged.Keywords[0] != "go" {
		t.Errorf("expected inherited keywords, got %v", merged.Keywords)
	}

	if merged.OpenGraph.SiteName != "GoKS Site" {
		t.Errorf("expected inherited og:site_name 'GoKS Site', got %q", merged.OpenGraph.SiteName)
	}

	if merged.OpenGraph.Title != "GoKS Docs" {
		t.Errorf("expected og:title 'GoKS Docs', got %q", merged.OpenGraph.Title)
	}
}

func TestRenderHTML(t *testing.T) {
	meta := Metadata{
		Title:       "My Page | Test",
		Description: "Test description",
		Keywords:    []string{"seo", "golang"},
		Canonical:   "https://example.com/page",
		ThemeColor:  "#3b82f6",
		OpenGraph: &OpenGraph{
			Title:    "OG Title",
			Images:   []OGImage{{URL: "https://example.com/og.png", Width: 1200, Height: 630}},
			SiteName: "My Site",
		},
		Twitter: &Twitter{
			Card:    "summary_large_image",
			Creator: "@developer",
		},
	}

	html := RenderHTML(meta)

	checks := []string{
		`<meta charset="UTF-8" />`,
		`<meta name="viewport" content="width=device-width, initial-scale=1.0" />`,
		`<title>My Page | Test</title>`,
		`<meta name="description" content="Test description" />`,
		`<meta name="keywords" content="seo, golang" />`,
		`<link rel="canonical" href="https://example.com/page" />`,
		`<meta name="theme-color" content="#3b82f6" />`,
		`<meta property="og:title" content="OG Title" />`,
		`<meta property="og:site_name" content="My Site" />`,
		`<meta property="og:image" content="https://example.com/og.png" />`,
		`<meta property="og:image:width" content="1200" />`,
		`<meta property="og:image:height" content="630" />`,
		`<meta name="twitter:card" content="summary_large_image" />`,
		`<meta name="twitter:creator" content="@developer" />`,
	}

	for _, check := range checks {
		if !strings.Contains(html, check) {
			t.Errorf("expected HTML to contain %q, but was:\n%s", check, html)
		}
	}
}

func TestExtractFromTree(t *testing.T) {
	layout := &mockLayout{}
	page := &mockPage{}

	tree := component.H("div", nil,
		component.C(layout),
		component.C(page),
	)

	extracted := ExtractFromTree(tree)
	if extracted.Title != "Documentation | GoKS Framework" {
		t.Errorf("expected extracted title 'Documentation | GoKS Framework', got %q", extracted.Title)
	}
}
