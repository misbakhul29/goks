package ws

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
)

func TestUpgrader_CheckOrigin(t *testing.T) {
	// Same origin / host matches
	req1 := httptest.NewRequest("GET", "http://example.com/ws", nil)
	req1.Host = "example.com"
	req1.Header.Set("Origin", "http://example.com")
	if !upgrader.CheckOrigin(req1) {
		t.Fatalf("expected same origin to be accepted")
	}

	// No Origin header (native clients / curl)
	req2 := httptest.NewRequest("GET", "http://example.com/ws", nil)
	req2.Host = "example.com"
	if !upgrader.CheckOrigin(req2) {
		t.Fatalf("expected no origin to be accepted")
	}

	// Cross-site Origin
	req3 := httptest.NewRequest("GET", "http://example.com/ws", nil)
	req3.Host = "example.com"
	req3.Header.Set("Origin", "http://malicious-site.com")
	if upgrader.CheckOrigin(req3) {
		t.Fatalf("expected cross origin to be rejected")
	}

	// A host suffix is not the same origin.
	req4 := httptest.NewRequest("GET", "http://example.com/ws", nil)
	req4.Host = "example.com"
	req4.Header.Set("Origin", "https://example.com.attacker.test")
	if upgrader.CheckOrigin(req4) {
		t.Fatalf("expected host suffix origin to be rejected")
	}
}

func TestHub_Broadcast_DoesNotPanic(t *testing.T) {
	hub := NewHub()

	// Add dummy client with full buffer
	client := &Client{
		send: make(chan []byte, 1),
		hub:  hub,
	}
	client.send <- []byte("existing")

	hub.clients[client] = true

	// Broadcast when channel is full should safely disconnect client without panicking or deadlock
	hub.Broadcast([]byte("new message"))

	hub.mu.RLock()
	_, exists := hub.clients[client]
	hub.mu.RUnlock()

	if exists {
		t.Fatalf("expected disconnected client to be removed from hub")
	}
}

func TestClient_SendAfterDisconnectDoesNotPanic(t *testing.T) {
	hub := NewHub()
	client := &Client{
		send: make(chan []byte, 1),
		hub:  hub,
	}
	client.send <- []byte("existing")
	hub.clients[client] = true

	hub.Broadcast([]byte("disconnect"))

	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("Send panicked after disconnect: %v", recovered)
		}
	}()
	client.Send([]byte("late message"))
}

func TestEventHub_RemovesDisconnectedClientsFromRooms(t *testing.T) {
	hub := NewEventHub()
	client := &Client{send: make(chan []byte, 1), hub: hub.Hub}
	hub.clients[client] = true
	hub.Join("general", client)

	hub.remove(client)

	if got := hub.Rooms().RoomCount("general"); got != 0 {
		t.Fatalf("expected disconnected client to leave room, got %d clients", got)
	}
}

func TestEventHub_HandlerRegistrationConcurrent(t *testing.T) {
	hub := NewEventHub()
	client := &Client{send: make(chan []byte, 1), hub: hub.Hub}
	done := make(chan struct{})

	go func() {
		for i := 0; i < 100; i++ {
			hub.On("message", func(*Client, json.RawMessage) {})
		}
		close(done)
	}()

	for i := 0; i < 100; i++ {
		hub.dispatch(client, EventMessage{Event: "message"})
	}
	<-done
}
