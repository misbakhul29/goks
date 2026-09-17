package ws

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestHub_ShutdownAndClose(t *testing.T) {
	h := NewHub()

	var disconnectMu sync.Mutex
	disconnectCount := 0
	h.onDisconnect = func(c *Client) {
		disconnectMu.Lock()
		disconnectCount++
		disconnectMu.Unlock()
	}

	// Add 3 mock clients to the hub
	c1 := &Client{send: make(chan []byte, 10), hub: h}
	c2 := &Client{send: make(chan []byte, 10), hub: h}
	c3 := &Client{send: make(chan []byte, 10), hub: h}

	h.mu.Lock()
	h.clients[c1] = true
	h.clients[c2] = true
	h.clients[c3] = true
	h.mu.Unlock()

	if h.ClientCount() != 3 {
		t.Fatalf("expected 3 clients before shutdown, got %d", h.ClientCount())
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	if err := h.Shutdown(ctx); err != nil {
		t.Fatalf("unexpected shutdown error: %v", err)
	}

	if h.ClientCount() != 0 {
		t.Fatalf("expected 0 clients after shutdown, got %d", h.ClientCount())
	}

	disconnectMu.Lock()
	if disconnectCount != 3 {
		t.Fatalf("expected 3 disconnect callbacks, got %d", disconnectCount)
	}
	disconnectMu.Unlock()

	// Verify clients send channel is closed
	c1.sendMu.RLock()
	if !c1.closed {
		t.Fatal("expected c1 to be closed")
	}
	c1.sendMu.RUnlock()

	// Repeated call to Close() should be safe no-op
	h.Close()
	if h.ClientCount() != 0 {
		t.Fatalf("expected 0 clients after second Close(), got %d", h.ClientCount())
	}
}
