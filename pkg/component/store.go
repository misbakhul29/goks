package component

import "sync"

// Store is a global reactive state container.
// It allows multiple components to subscribe to state changes.
type Store[T any] struct {
	mu        sync.RWMutex
	value     T
	listeners map[int]func(T)
	nextID    int
}

// NewStore creates a new Store with an initial value.
func NewStore[T any](initial T) *Store[T] {
	return &Store[T]{
		value:     initial,
		listeners: make(map[int]func(T)),
	}
}

// Get returns the current value of the store.
func (s *Store[T]) Get() T {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.value
}

// Set updates the store's value and notifies all subscribers.
func (s *Store[T]) Set(val T) {
	s.mu.Lock()
	s.value = val
	// Copy listeners to avoid deadlocks if a callback tries to read/write the store
	cbs := make([]func(T), 0, len(s.listeners))
	for _, cb := range s.listeners {
		cbs = append(cbs, cb)
	}
	s.mu.Unlock()

	for _, cb := range cbs {
		cb(val)
	}
}

// Subscribe adds a listener to the store and returns an unsubscribe function.
// Best used inside a component's OnMount() method, with the returned function
// called inside OnUnmount() to prevent memory leaks.
func (s *Store[T]) Subscribe(cb func(T)) func() {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := s.nextID
	s.nextID++
	s.listeners[id] = cb

	return func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		delete(s.listeners, id)
	}
}
