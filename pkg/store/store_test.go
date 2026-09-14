package store_test

import (
	"sync"
	"testing"

	"github.com/misbakhul29/goks/pkg/store"
)

type CounterState struct {
	Count int
}

func TestStore_BasicGetSetUpdate(t *testing.T) {
	s := store.New(CounterState{Count: 10})
	if s.Get().Count != 10 {
		t.Fatalf("expected count 10, got %d", s.Get().Count)
	}

	s.Set(CounterState{Count: 20})
	if s.Get().Count != 20 {
		t.Fatalf("expected count 20, got %d", s.Get().Count)
	}

	s.Update(func(cs CounterState) CounterState {
		cs.Count += 5
		return cs
	})
	if s.Get().Count != 25 {
		t.Fatalf("expected count 25, got %d", s.Get().Count)
	}
}

func TestStore_SubscriptionAndUnsubscribe(t *testing.T) {
	s := store.New(0)

	var received1, received2, received3 int
	unsub1 := s.Subscribe(func(v int) { received1 = v })
	unsub2 := s.Subscribe(func(v int) { received2 = v })
	unsub3 := s.Subscribe(func(v int) { received3 = v })

	s.Set(1)
	if received1 != 1 || received2 != 1 || received3 != 1 {
		t.Fatalf("expected all subscribers to receive 1, got %d, %d, %d", received1, received2, received3)
	}

	// Unsubscribe the first and third subscribers (non-contiguous)
	unsub1()
	unsub3()

	s.Set(2)
	if received1 != 1 {
		t.Errorf("unsubscribed subscriber 1 received update: %d", received1)
	}
	if received2 != 2 {
		t.Errorf("active subscriber 2 did not receive update: %d", received2)
	}
	if received3 != 1 {
		t.Errorf("unsubscribed subscriber 3 received update: %d", received3)
	}

	// Unsubscribe the remaining subscriber
	unsub2()
	s.Set(3)
	if received2 != 2 {
		t.Errorf("unsubscribed subscriber 2 received update: %d", received2)
	}
}

func TestStore_ConcurrentAccess(t *testing.T) {
	s := store.New(0)
	var wg sync.WaitGroup

	// Concurrently subscribe and unsubscribe
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			unsub := s.Subscribe(func(v int) {
				_ = v
			})
			defer unsub()
			for j := 0; j < 50; j++ {
				_ = s.Get()
			}
		}()
	}

	// Concurrently update state
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				s.Update(func(val int) int {
					return val + 1
				})
			}
		}()
	}

	wg.Wait()
}
