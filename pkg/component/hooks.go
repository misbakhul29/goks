// Package component - Hooks Engine for GoKS functional components.
package component

import (
	"fmt"
	"reflect"
	"sync"
)

// FiberNode tracks the state and hooks of a component across re-renders.
// It forms a tree that mirrors the component hierarchy.
type FiberNode struct {
	mu          sync.Mutex
	CompType    reflect.Type
	FuncPtr     uintptr
	Hooks       []any
	HookIndex   int
	ChildIndex  int
	Children    []*FiberNode
	AppRerender func()
}

func (f *FiberNode) getOrCreateChild(index int) *FiberNode {
	f.mu.Lock()
	defer f.mu.Unlock()
	if index >= len(f.Children) {
		f.Children = append(f.Children, &FiberNode{})
	}
	return f.Children[index]
}

var (
	activeFiber *FiberNode
	fiberMu     sync.Mutex
)

// setActiveFiber sets the current fiber node being rendered.
// It returns the previous fiber node to allow restoring the context.
func setActiveFiber(f *FiberNode) *FiberNode {
	fiberMu.Lock()
	defer fiberMu.Unlock()
	prev := activeFiber
	activeFiber = f
	return prev
}

// UseState is a React-like hook for managing state in functional components.
// It returns the current state value and a function to update it.
// Calling the update function will trigger an app re-render.
// If called outside of an active client fiber (e.g. during SSR), it safely returns the initial state.
func UseState[T any](initial T) (T, func(T)) {
	fiberMu.Lock()
	f := activeFiber
	fiberMu.Unlock()

	if f == nil {
		// Standalone / Server-side rendering (SSR) fallback: return initial state with no-op setter.
		return initial, func(T) {}
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	idx := f.HookIndex
	f.HookIndex++

	// Initialize state if it's the first render for this hook
	if idx >= len(f.Hooks) {
		f.Hooks = append(f.Hooks, initial)
	} else {
		// Verify hook order invariant: type at this slot must match previous render
		prev := f.Hooks[idx]
		if prev != nil {
			prevType := reflect.TypeOf(prev)
			expectedType := reflect.TypeOf((*T)(nil)).Elem()
			if prevType != expectedType {
				panic(fmt.Sprintf("GoKS Hook Invariant Violation: Hook at index %d expected type %v, but was previously initialized with type %v. Hooks must not be called conditionally or in varying order.", idx, expectedType, prevType))
			}
		}
	}

	val := f.Hooks[idx].(T)

	setVal := func(newVal T) {
		// Update the state in the fiber tree
		f.Hooks[idx] = newVal
		// Trigger a re-render
		if f.AppRerender != nil {
			f.AppRerender()
		}
	}

	return val, setVal
}
