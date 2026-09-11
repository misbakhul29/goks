package server

import (
	"strings"
	"testing"

	"github.com/misbakhul29/goks/pkg/component"
	"github.com/misbakhul29/goks/pkg/html"
	"github.com/misbakhul29/goks/pkg/metadata"
)

type testLayout struct {
	component.ComponentBase
	Children component.Renderable
}

func (l *testLayout) Metadata() metadata.Metadata {
	return metadata.Metadata{
		Title:         "GoKS",
		TitleTemplate: "%s | GoKS App",
		Description:   "Test GoKS App Description",
		OpenGraph: &metadata.OpenGraph{
			SiteName: "GoKS Site",
			Type:     "website",
		},
	}
}

func (l *testLayout) Render() *component.Node {
	return html.Html(
		html.Body(
			component.C(l.Children),
		).Class("antialiased"),
	).Attr("lang", "en")
}

type testPage struct {
	component.ComponentBase
}

func (p *testPage) Metadata() metadata.Metadata {
	return metadata.Metadata{
		Title:       "Dashboard",
		Description: "User Dashboard",
	}
}

func (p *testPage) Render() *component.Node {
	return html.Div(
		html.H1(component.Text("Dashboard Content")),
	)
}

func TestRenderDocumentHTML_AutoHead(t *testing.T) {
	page := &testPage{}
	layout := &testLayout{Children: page}

	rootComponentNode := component.C(layout)
	meta := metadata.ExtractFromTree(rootComponentNode)
	renderedNode := component.Expand(rootComponentNode, func() {}, nil)

	docHTML := renderDocumentHTML(renderedNode, meta, "<script>env</script>", "<script>lr</script>")

	expectedSubstrings := []string{
		"<!DOCTYPE html>",
		`<html lang="en"`,
		`<head>`,
		`<title>Dashboard | GoKS App</title>`,
		`<meta name="description" content="User Dashboard" />`,
		`<meta property="og:site_name" content="GoKS Site" />`,
		`<link rel="stylesheet" href="/app.css" />`,
		`<script>env</script>`,
		`<body class="antialiased">`,
		`<div id="__goks">`,
		`<h1>Dashboard Content</h1>`,
		`/wasm_exec.js`,
		`<script>lr</script>`,
		`</body>`,
		`</html>`,
	}

	for _, sub := range expectedSubstrings {
		if !strings.Contains(docHTML, sub) {
			t.Errorf("expected HTML to contain %q, but got:\n%s", sub, docHTML)
		}
	}
}

func TestRenderDocumentHTML_CustomHead(t *testing.T) {
	customLayoutNode := html.Html(
		html.Head(
			html.Script().Attr("src", "https://example.com/custom.js"),
		),
		html.Body(
			html.Div(component.Text("Hello")),
		).Class("bg-dark"),
	).Attr("lang", "id")

	meta := metadata.Metadata{
		Title:       "Custom App",
		Description: "Testing custom head",
	}

	docHTML := renderDocumentHTML(customLayoutNode, meta, "<script>env</script>", "")

	expectedSubstrings := []string{
		"<!DOCTYPE html>",
		`<html lang="id"`,
		`<head>`,
		`https://example.com/custom.js`,
		`<title>Custom App</title>`,
		`<meta name="description" content="Testing custom head" />`,
		`<body class="bg-dark">`,
		`<div id="__goks">`,
		`Hello`,
		`/wasm_exec.js`,
		`</body>`,
	}

	for _, sub := range expectedSubstrings {
		if !strings.Contains(docHTML, sub) {
			t.Errorf("expected HTML to contain %q, but got:\n%s", sub, docHTML)
		}
	}
}
