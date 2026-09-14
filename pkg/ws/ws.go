// Package ws provides WebSocket support for GoKS.
// Server-side: wraps gorilla/websocket with a simple hub pattern.
// Client-side (WASM): wraps browser WebSocket API.
package ws

import (
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
)

// -----------------------------------------------------------------------
// Server-side WebSocket Hub
// -----------------------------------------------------------------------

var upgrader = websocket.Upgrader{
	// Only allow same-origin WebSocket connections to prevent
	// Cross-Site WebSocket Hijacking (CSWSH) attacks.
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		// No Origin header = same-origin (curl, native clients, etc.)
		if origin == "" {
			return true
		}

		originURL, err := url.Parse(origin)
		if err != nil || originURL.Scheme == "" || originURL.Host == "" ||
			originURL.User != nil || originURL.Path != "" ||
			originURL.RawQuery != "" || originURL.Fragment != "" {
			return false
		}

		scheme := "http"
		if r.TLS != nil {
			scheme = "https"
		} else if r.URL != nil && r.URL.Scheme != "" {
			scheme = strings.ToLower(r.URL.Scheme)
		}
		if !strings.EqualFold(originURL.Scheme, scheme) {
			return false
		}

		requestURL, err := url.Parse(scheme + "://" + r.Host)
		if err != nil || requestURL.Host == "" {
			return false
		}
		return sameHost(originURL, requestURL, scheme)
	},
}

func sameHost(a, b *url.URL, scheme string) bool {
	aHost := strings.ToLower(strings.TrimSuffix(a.Hostname(), "."))
	bHost := strings.ToLower(strings.TrimSuffix(b.Hostname(), "."))
	if aHost == "" || aHost != bHost {
		return false
	}

	aPort := a.Port()
	bPort := b.Port()
	if aPort == "" {
		aPort = defaultPort(scheme)
	}
	if bPort == "" {
		bPort = defaultPort(scheme)
	}
	return aPort == bPort
}

func defaultPort(scheme string) string {
	if strings.EqualFold(scheme, "https") {
		return "443"
	}
	return "80"
}

// Client represents a connected WebSocket client.
type Client struct {
	conn   *websocket.Conn
	send   chan []byte
	hub    *Hub
	ID     string
	sendMu sync.RWMutex
	closed bool
}

func (c *Client) enqueue(msg []byte) bool {
	c.sendMu.RLock()
	defer c.sendMu.RUnlock()
	if c.closed {
		return false
	}
	select {
	case c.send <- msg:
		return true
	default:
		return false
	}
}

func (c *Client) closeSend() {
	c.sendMu.Lock()
	defer c.sendMu.Unlock()
	if c.closed {
		return
	}
	c.closed = true
	close(c.send)
}

// Hub manages connected WebSocket clients and message broadcasting.
type Hub struct {
	mu           sync.RWMutex
	clients      map[*Client]bool
	onMsg        func(client *Client, msg []byte)
	onDisconnect func(client *Client)
}

// NewHub creates a new WebSocket hub.
func NewHub() *Hub {
	return &Hub{
		clients: make(map[*Client]bool),
	}
}

// OnMessage registers a handler called when any client sends a message.
func (h *Hub) OnMessage(fn func(client *Client, msg []byte)) {
	h.onMsg = fn
}

// Broadcast sends a message to all connected clients.
// Clients with a full send buffer are disconnected.
func (h *Hub) Broadcast(msg []byte) {
	// Use full Lock (not RLock) because we may delete from the map.
	h.mu.Lock()
	defer h.mu.Unlock()
	for client := range h.clients {
		if !client.enqueue(msg) {
			// Buffer full — disconnect this client.
			delete(h.clients, client)
			client.closeSend()
		}
	}
}

func (h *Hub) remove(client *Client) {
	h.mu.Lock()
	delete(h.clients, client)
	h.mu.Unlock()
	client.closeSend()
	if h.onDisconnect != nil {
		h.onDisconnect(client)
	}
}

// Send sends a message to a specific client.
func (c *Client) Send(msg []byte) {
	_ = c.enqueue(msg)
}

// Handler returns an http.HandlerFunc that upgrades connections and registers them with the hub.
func (h *Hub) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("[GoKS/ws] upgrade error: %v", err)
			return
		}

		client := &Client{
			conn: conn,
			send: make(chan []byte, 256),
			hub:  h,
		}

		h.mu.Lock()
		h.clients[client] = true
		h.mu.Unlock()

		// Writer goroutine
		go func() {
			defer conn.Close()
			for msg := range client.send {
				if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
					break
				}
			}
		}()

		// Reader goroutine (blocking)
		defer func() {
			h.remove(client)
			conn.Close()
		}()

		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				break
			}
			if h.onMsg != nil {
				h.onMsg(client, msg)
			}
		}
	}
}

// ClientCount returns the number of currently connected clients.
func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}
