// Package livereload provides a lightweight live-reload server for GoKS dev mode.
// It injects a small JS snippet into HTML responses that connects via WebSocket
// and reloads the page when the server sends a reload signal.
package livereload

import (
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

const injectedScript = `
<script>
(function() {
  var ws = new WebSocket("ws://" + location.host + "/__goks_livereload");
  ws.onmessage = function(e) {
    if (e.data === "reload") {
      console.log("[GoKS] Live reload triggered");
      location.reload();
    }
  };
  ws.onclose = function() {
    // Reconnect loop
    setTimeout(function() { location.reload(); }, 1000);
  };
})();
</script>
`

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// Server manages live-reload WebSocket connections.
type Server struct {
	mu      sync.Mutex
	clients map[*websocket.Conn]bool
}

// New creates a new live-reload server.
func New() *Server {
	return &Server{clients: make(map[*websocket.Conn]bool)}
}

// Handler returns an http.HandlerFunc for the WebSocket endpoint.
func (s *Server) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("[GoKS/livereload] WebSocket upgrade error: %v", err)
			return
		}
		s.mu.Lock()
		s.clients[conn] = true
		s.mu.Unlock()

		// Keep connection open; remove on close
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				break
			}
		}
		s.mu.Lock()
		delete(s.clients, conn)
		conn.Close()
		s.mu.Unlock()
	}
}

// Reload broadcasts a reload signal to all connected browser clients.
func (s *Server) Reload() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for conn := range s.clients {
		if err := conn.WriteMessage(websocket.TextMessage, []byte("reload")); err != nil {
			conn.Close()
			delete(s.clients, conn)
		}
	}
	log.Println("[GoKS] Live reload triggered →", len(s.clients), "client(s)")
}

// Script returns the JavaScript snippet to inject into HTML pages.
func Script() string {
	return injectedScript
}
