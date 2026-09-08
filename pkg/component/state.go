// Package component - State management for GoKS components.
// This file runs on the WASM (browser) side.
package component

import "sync"

// State holds a reactive value. When set, the owning component will re-render.
type State[T any] struct {
	mu       sync.Mutex
	value    T
	onChange func()
}

// NewState creates a new State with an initial value.
func NewState[T any](initial T) *State[T] {
	return &State[T]{value: initial}
}

// Get returns the current value.
func (s *State[T]) Get() T {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.value
}

// Set updates the value and triggers a re-render if a callback is registered.
func (s *State[T]) Set(val T) {
	s.mu.Lock()
	s.value = val
	cb := s.onChange
	s.mu.Unlock()
	if cb != nil {
		cb()
	}
}

// bind attaches a re-render callback to this state (called by the component system).
func (s *State[T]) bind(cb func()) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onChange = cb
}

// -----------------------------------------------------------------------
// Component base
// -----------------------------------------------------------------------

// ComponentBase provides the base lifecycle and re-render hooks for a GoKS component.
// Embed this in your own structs.
//
//	type MyComp struct {
//	    component.ComponentBase
//	    Count *component.State[int]
//	}
type ComponentBase struct {
	rerender func()
	mounted  bool
}

// Rerender triggers a re-render of this component.
func (c *ComponentBase) Rerender() {
	if c.rerender != nil {
		c.rerender()
	}
}

// IsMounted returns true if the component has been mounted to the DOM.
func (c *ComponentBase) IsMounted() bool {
	return c.mounted
}

// bindRerender is called internally by the runtime to wire up the re-render hook.
func (c *ComponentBase) bindRerender(fn func()) {
	c.rerender = fn
}

// setMounted is called by the runtime after the component mounts.
func (c *ComponentBase) setMounted(v bool) {
	c.mounted = v
}

// Lifecycle interfaces — implement these in your component struct as needed.

// Mounter is implemented by components that want an OnMount lifecycle hook.
type Mounter interface {
	OnMount()
}

// Unmounter is implemented by components that want an OnUnmount lifecycle hook.
type Unmounter interface {
	OnUnmount()
}

// Updater is implemented by components that want an OnUpdate lifecycle hook.
type Updater interface {
	OnUpdate(prevProps Props)
}
