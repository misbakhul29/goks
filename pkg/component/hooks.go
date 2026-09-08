// Package component - Hooks Engine for GoKS functional components.
package component

import (
	"reflect"
	"sync"
)

// FiberNode tracks the state and hooks of a component across re-renders.
// It forms a tree that mirrors the component hierarchy.
type FiberNode struct {
	CompType    reflect.Type
	FuncPtr     uintptr
	Hooks       []any
	HookIndex   int
	ChildIndex  int
	Children    []*FiberNode
	AppRerender func()
}

func (f *FiberNode) getOrCreateChild(index int) *FiberNode {
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
func UseState[T any](initial T) (T, func(T)) {
	fiberMu.Lock()
	f := activeFiber
	fiberMu.Unlock()

	if f == nil {
		panic("UseState must be called inside a functional component's Render method")
	}

	idx := f.HookIndex
	f.HookIndex++

	// Initialize state if it's the first render for this hook
	if idx >= len(f.Hooks) {
		f.Hooks = append(f.Hooks, initial)
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
