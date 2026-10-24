package ws_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/misbakhul29/goks/pkg/ws"
)

func TestWS_MaxMessageSizeLimit(t *testing.T) {
	hub := ws.NewHub()
	var received []byte
	var mu sync.Mutex

	hub.OnMessage(func(c *ws.Client, msg []byte) {
		mu.Lock()
		received = msg
		mu.Unlock()
	})

	server := httptest.NewServer(hub.Handler())
	defer server.Close()

	u, _ := url.Parse(server.URL)
	wsURL := "ws://" + u.Host

	header := http.Header{}
	header.Set("Origin", server.URL)

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, header)
	if err != nil {
		t.Fatalf("failed to dial websocket: %v", err)
	}
	defer conn.Close()

	// 1. Send normal small message -> succeeds
	if err := conn.WriteMessage(websocket.TextMessage, []byte("hello")); err != nil {
		t.Fatalf("failed to write small message: %v", err)
	}
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	got := string(received)
	mu.Unlock()
	if got != "hello" {
		t.Fatalf("expected 'hello', got %q", got)
	}

	// 2. Send oversized message (> 512 KB) -> server read limit closes connection
	oversized := bytes.Repeat([]byte("A"), 600*1024)
	_ = conn.WriteMessage(websocket.TextMessage, oversized)

	time.Sleep(100 * time.Millisecond)

	// Attempting to read from client should now return error because server closed it
	_ = conn.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
	_, _, err = conn.ReadMessage()
	if err == nil {
		t.Fatal("expected connection to be closed after exceeding max message size limit")
	}
}

func TestWS_HandshakeAuthentication(t *testing.T) {
	hub := ws.NewHub()
	hub.Authenticate(func(r *http.Request) (string, error) {
		token := r.URL.Query().Get("token")
		if token != "valid-auth-token" {
			return "", http.ErrNoCookie
		}
		return "user-42", nil
	})

	server := httptest.NewServer(hub.Handler())
	defer server.Close()

	u, _ := url.Parse(server.URL)
	header := http.Header{}
	header.Set("Origin", server.URL)

	// 1. Connection without valid token -> Rejected with 401
	wsURLUnauth := "ws://" + u.Host + "/?token=invalid"
	_, resp, err := websocket.DefaultDialer.Dial(wsURLUnauth, header)
	if err == nil {
		t.Fatal("expected error dialing without valid auth token")
	}
	if resp != nil && resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized, got %d", resp.StatusCode)
	}

	// 2. Connection with valid token -> Succeeds
	wsURLAuth := "ws://" + u.Host + "/?token=valid-auth-token"
	conn, respAuth, err := websocket.DefaultDialer.Dial(wsURLAuth, header)
	if err != nil {
		t.Fatalf("expected successful connection with valid token, got error: %v", err)
	}
	defer conn.Close()
	if respAuth.StatusCode != http.StatusSwitchingProtocols {
		t.Fatalf("expected 101 Switching Protocols, got %d", respAuth.StatusCode)
	}
}
