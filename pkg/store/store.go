// Package store provides global reactive state management for GoKS.
package store

import "sync"

// Store[T] is a typed global state store.
// Create one at package level and use it across components.
//
//	var CounterStore = store.New(CounterState{Count: 0})
//
//	// In a component:
//	state := CounterStore.Get()
//	CounterStore.Set(CounterState{Count: state.Count + 1})
type Store[T any] struct {
	mu        sync.RWMutex
	state     T
	listeners map[int]func(T)
	nextID    int
}

// New creates a new Store with an initial state.
func New[T any](initial T) *Store[T] {
	return &Store[T]{
		state:     initial,
		listeners: make(map[int]func(T)),
	}
}

// Get returns the current state (read-only copy).
func (s *Store[T]) Get() T {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state
}

// Set updates the state and notifies all subscribers.
func (s *Store[T]) Set(newState T) {
	s.mu.Lock()
	s.state = newState
	subs := make([]func(T), 0, len(s.listeners))
	for _, sub := range s.listeners {
		subs = append(subs, sub)
	}
	s.mu.Unlock()

	for _, sub := range subs {
		sub(newState)
	}
}

// Update applies a function to the current state, then notifies subscribers.
// Useful for partial updates:
//
//	CounterStore.Update(func(s CounterState) CounterState {
//	    s.Count++
//	    return s
//	})
func (s *Store[T]) Update(fn func(T) T) {
	s.mu.Lock()
	s.state = fn(s.state)
	newState := s.state
	subs := make([]func(T), 0, len(s.listeners))
	for _, sub := range s.listeners {
		subs = append(subs, sub)
	}
	s.mu.Unlock()

	for _, sub := range subs {
		sub(newState)
	}
}

// Subscribe registers a callback to be called whenever the state changes.
// Returns an unsubscribe function.
func (s *Store[T]) Subscribe(fn func(T)) func() {
	s.mu.Lock()
	id := s.nextID
	s.nextID++
	s.listeners[id] = fn
	s.mu.Unlock()

	return func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		delete(s.listeners, id)
	}
}
