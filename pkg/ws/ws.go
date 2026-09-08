// Package ws provides WebSocket support for GoKS.
// Server-side: wraps gorilla/websocket with a simple hub pattern.
// Client-side (WASM): wraps browser WebSocket API.
package ws

import (
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

// -----------------------------------------------------------------------
// Server-side WebSocket Hub
// -----------------------------------------------------------------------

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// Client represents a connected WebSocket client.
type Client struct {
	conn *websocket.Conn
	send chan []byte
	hub  *Hub
	ID   string
}

// Hub manages connected WebSocket clients and message broadcasting.
type Hub struct {
	mu      sync.RWMutex
	clients map[*Client]bool
	onMsg   func(client *Client, msg []byte)
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
func (h *Hub) Broadcast(msg []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for client := range h.clients {
		select {
		case client.send <- msg:
		default:
			close(client.send)
			delete(h.clients, client)
		}
	}
}

// Send sends a message to a specific client.
func (c *Client) Send(msg []byte) {
	c.send <- msg
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
			h.mu.Lock()
			delete(h.clients, client)
			h.mu.Unlock()
			close(client.send)
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
