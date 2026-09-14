package component_test

import (
	"fmt"
	"testing"

	"github.com/misbakhul29/goks/pkg/component"
)

func buildSampleTree(depth, breadth int) *component.Node {
	if depth <= 0 {
		return component.Text("leaf node content")
	}
	children := make([]*component.Node, breadth)
	for i := 0; i < breadth; i++ {
		children[i] = buildSampleTree(depth-1, breadth)
	}
	return component.H("div", component.Props{
		"class": "container-row",
		"id":    fmt.Sprintf("node-depth-%d", depth),
	}, children...)
}

func BenchmarkComponent_RenderToString(b *testing.B) {
	tree := buildSampleTree(3, 3)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = component.RenderToString(tree)
	}
}

func BenchmarkComponent_Reconcile(b *testing.B) {
	tree1 := buildSampleTree(3, 3)
	tree2 := buildSampleTree(3, 3)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = component.Reconcile(tree1, tree2)
	}
}
