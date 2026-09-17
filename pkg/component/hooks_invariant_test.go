package component_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/misbakhul29/goks/pkg/component"
)

func TestHookInvariant_ConsistentRenders(t *testing.T) {
	fiber := &component.FiberNode{}
	rerenderCalled := false
	appRerender := func() {
		rerenderCalled = true
	}

	var currentCount int
	var countSetter func(int)

	comp := component.FC(func() *component.Node {
		val, setVal := component.UseState(10)
		name, _ := component.UseState("hello")
		currentCount = val
		countSetter = setVal
		return component.H("div", nil, component.Text(fmt.Sprintf("%s: %d", name, val)))
	})

	// Pass 1
	fiber.ChildIndex = 0
	n1 := component.Expand(comp, appRerender, fiber)
	if n1 == nil {
		t.Fatal("expected rendered node on pass 1")
	}
	if currentCount != 10 {
		t.Fatalf("expected count 10 on pass 1, got %d", currentCount)
	}

	// Update state
	countSetter(25)
	if !rerenderCalled {
		t.Fatal("expected appRerender to be called")
	}

	// Pass 2
	fiber.ChildIndex = 0
	n2 := component.Expand(comp, appRerender, fiber)
	if n2 == nil {
		t.Fatal("expected rendered node on pass 2")
	}
	if currentCount != 25 {
		t.Fatalf("expected count 25 on pass 2, got %d", currentCount)
	}

	// Pass 3 (no change)
	fiber.ChildIndex = 0
	n3 := component.Expand(comp, appRerender, fiber)
	if n3 == nil {
		t.Fatal("expected rendered node on pass 3")
	}
	if currentCount != 25 {
		t.Fatalf("expected count 25 on pass 3, got %d", currentCount)
	}
}

func TestHookInvariant_FewerHooksOnSubsequentRender(t *testing.T) {
	fiber := &component.FiberNode{}
	appRerender := func() {}

	condition := true

	comp := component.FC(func() *component.Node {
		component.UseState(1)
		if condition {
			component.UseState("extra")
		}
		return component.H("div", nil)
	})

	// Pass 1: renders 2 hooks
	fiber.ChildIndex = 0
	component.Expand(comp, appRerender, fiber)

	// Pass 2: condition = false, renders only 1 hook (fewer)
	condition = false
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic when fewer hooks are rendered")
		}
		msg := fmt.Sprintf("%v", r)
		if !strings.Contains(msg, "Rendered fewer hooks than during the previous render pass") {
			t.Fatalf("unexpected panic message: %s", msg)
		}
	}()

	fiber.ChildIndex = 0
	component.Expand(comp, appRerender, fiber)
}

func TestHookInvariant_MoreHooksOnSubsequentRender(t *testing.T) {
	fiber := &component.FiberNode{}
	appRerender := func() {}

	condition := false

	comp := component.FC(func() *component.Node {
		component.UseState(1)
		if condition {
			component.UseState("extra-conditional-hook")
		}
		return component.H("div", nil)
	})

	// Pass 1: renders 1 hook
	fiber.ChildIndex = 0
	component.Expand(comp, appRerender, fiber)

	// Pass 2: condition = true, calls 2nd hook (more)
	condition = true
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic when more hooks are rendered")
		}
		msg := fmt.Sprintf("%v", r)
		if !strings.Contains(msg, "Rendered more hooks than during the previous render pass") {
			t.Fatalf("unexpected panic message: %s", msg)
		}
	}()

	fiber.ChildIndex = 0
	component.Expand(comp, appRerender, fiber)
}

func TestHookInvariant_TypeMismatchOnSubsequentRender(t *testing.T) {
	fiber := &component.FiberNode{}
	appRerender := func() {}

	renderString := false

	comp := component.FC(func() *component.Node {
		if renderString {
			component.UseState("string-value")
		} else {
			component.UseState(42)
		}
		return component.H("div", nil)
	})

	// Pass 1: renders int
	fiber.ChildIndex = 0
	component.Expand(comp, appRerender, fiber)

	// Pass 2: renders string at slot 0
	renderString = true
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic when hook type mismatches")
		}
		msg := fmt.Sprintf("%v", r)
		if !strings.Contains(msg, "GoKS Hook Invariant Violation: Hook at index 0 expected type string, but was previously initialized with type int") {
			t.Fatalf("unexpected panic message: %s", msg)
		}
	}()

	fiber.ChildIndex = 0
	component.Expand(comp, appRerender, fiber)
}
