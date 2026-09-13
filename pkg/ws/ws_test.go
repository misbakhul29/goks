package ws

import (
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
