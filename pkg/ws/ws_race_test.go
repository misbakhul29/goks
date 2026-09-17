package ws

// Race-condition and concurrency tests for pkg/ws hub.
// Run with: go test -race ./pkg/ws/...
//
// The existing ws_test.go covers origin validation and basic
// broadcast/disconnect behaviour. These tests add concurrent scenarios
// that the race detector can catch.

import (
	"sync"
	"testing"
)

// TestHub_ConcurrentBroadcastAndDisconnect exercises the Hub under
// simultaneous Broadcast calls and client disconnects.
// This is the primary race scenario flagged in the architecture audit.
func TestHub_ConcurrentBroadcastAndDisconnect(t *testing.T) {
	const clients = 20
	const broadcasts = 50

	hub := NewHub()

	// Register clients with a large enough buffer so they won't be
	// force-disconnected by a full buffer — only by explicit remove.
	cs := make([]*Client, clients)
	for i := range cs {
		cs[i] = &Client{
			send: make(chan []byte, 256),
			hub:  hub,
		}
		hub.mu.Lock()
		hub.clients[cs[i]] = true
		hub.mu.Unlock()
	}

	var wg sync.WaitGroup

	// Concurrent broadcasters
	for i := 0; i < broadcasts; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			hub.Broadcast([]byte("ping"))
		}()
	}

	// Concurrent disconnects (half the clients)
	for i := 0; i < clients/2; i++ {
		wg.Add(1)
		client := cs[i]
		go func() {
			defer wg.Done()
			hub.remove(client)
		}()
	}

	wg.Wait()

	// After the storm, ClientCount must be consistent and non-negative.
	count := hub.ClientCount()
	if count < 0 {
		t.Fatalf("ClientCount returned negative value: %d", count)
	}
}

// TestHub_ConcurrentClientCountConsistency verifies that ClientCount
// never returns a stale or torn read while clients are added/removed.
func TestHub_ConcurrentClientCountConsistency(t *testing.T) {
	hub := NewHub()

	var wg sync.WaitGroup

	// Goroutines that add then immediately remove clients
	for i := 0; i < 30; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c := &Client{send: make(chan []byte, 1), hub: hub}
			hub.mu.Lock()
			hub.clients[c] = true
			hub.mu.Unlock()

			// Yield before removing to maximise race window
			hub.remove(c)
		}()
	}

	// Goroutines that continuously read ClientCount
	done := make(chan struct{})
	go func() {
		for {
			select {
			case <-done:
				return
			default:
				if hub.ClientCount() < 0 {
					t.Errorf("ClientCount < 0")
				}
			}
		}
	}()

	wg.Wait()
	close(done)
}

// TestRoomManager_ConcurrentJoinLeave exercises the RoomManager under
// concurrent Join and Leave calls for the same room.
func TestRoomManager_ConcurrentJoinLeave(t *testing.T) {
	rm := NewRoomManager()

	const goroutines = 40
	const room = "lobby"
	hub := NewHub()

	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c := &Client{send: make(chan []byte, 1), hub: hub}
			rm.Join(room, c)
			rm.Leave(room, c)
		}()
	}
	wg.Wait()

	// After all joins and leaves the room count must be 0 or positive.
	count := rm.RoomCount(room)
	if count < 0 {
		t.Fatalf("RoomCount < 0 after concurrent join/leave: %d", count)
	}
}

// TestRoomManager_ConcurrentBroadcast verifies that room broadcast
// under concurrent Join does not race.
func TestRoomManager_ConcurrentBroadcast(t *testing.T) {
	rm := NewRoomManager()
	const room = "chan1"
	hub := NewHub()

	var wg sync.WaitGroup

	// Pre-populate some clients
	for i := 0; i < 10; i++ {
		c := &Client{send: make(chan []byte, 4), hub: hub}
		rm.Join(room, c)
	}

	// Concurrent broadcasts
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rm.Broadcast(room, []byte("hello"))
		}()
	}

	// Concurrent joins while broadcasting
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c := &Client{send: make(chan []byte, 4), hub: hub}
			rm.Join(room, c)
		}()
	}

	wg.Wait()
}

// TestHub_BroadcastAfterAllClientsRemoved must not panic when the hub
// is empty.
func TestHub_BroadcastAfterAllClientsRemoved(t *testing.T) {
	hub := NewHub()
	c := &Client{send: make(chan []byte, 1), hub: hub}
	hub.mu.Lock()
	hub.clients[c] = true
	hub.mu.Unlock()
	hub.remove(c)

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Broadcast on empty hub panicked: %v", r)
		}
	}()
	hub.Broadcast([]byte("after remove"))
}

// TestClient_DoubleCloseSend verifies that calling closeSend twice
// does not panic (channel double-close guard).
func TestClient_DoubleCloseSend(t *testing.T) {
	c := &Client{send: make(chan []byte, 1)}
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("double closeSend panicked: %v", r)
		}
	}()
	c.closeSend()
	c.closeSend() // must be a no-op, not a panic
}
