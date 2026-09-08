package ws

import "sync"

// Room manages a named group of WebSocket clients.
// Use rooms to broadcast to specific subsets of clients (e.g., chat rooms, user channels).
//
// Example:
//
//	rooms := ws.NewRoomManager()
//	hub.OnConnect(func(client *ws.Client) {
//	    rooms.Join("general", client)
//	})
//
//	// Broadcast to all clients in "general"
//	rooms.Broadcast("general", []byte(`{"msg":"hello"}`))
type Room struct {
	mu      sync.RWMutex
	clients map[*Client]bool
}

// RoomManager manages named rooms.
type RoomManager struct {
	mu    sync.RWMutex
	rooms map[string]*Room
}

// NewRoomManager creates a new RoomManager.
func NewRoomManager() *RoomManager {
	return &RoomManager{rooms: make(map[string]*Room)}
}

// Join adds a client to a named room, creating it if necessary.
func (rm *RoomManager) Join(roomName string, client *Client) {
	rm.mu.Lock()
	r, ok := rm.rooms[roomName]
	if !ok {
		r = &Room{clients: make(map[*Client]bool)}
		rm.rooms[roomName] = r
	}
	rm.mu.Unlock()

	r.mu.Lock()
	r.clients[client] = true
	r.mu.Unlock()
}

// Leave removes a client from a named room.
func (rm *RoomManager) Leave(roomName string, client *Client) {
	rm.mu.RLock()
	r, ok := rm.rooms[roomName]
	rm.mu.RUnlock()
	if !ok {
		return
	}

	r.mu.Lock()
	delete(r.clients, client)
	r.mu.Unlock()
}

// LeaveAll removes a client from all rooms. Call this on disconnect.
func (rm *RoomManager) LeaveAll(client *Client) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	for _, r := range rm.rooms {
		r.mu.Lock()
		delete(r.clients, client)
		r.mu.Unlock()
	}
}

// Broadcast sends a message to all clients in the given room.
func (rm *RoomManager) Broadcast(roomName string, msg []byte) {
	rm.mu.RLock()
	r, ok := rm.rooms[roomName]
	rm.mu.RUnlock()
	if !ok {
		return
	}

	r.mu.RLock()
	defer r.mu.RUnlock()
	for client := range r.clients {
		select {
		case client.send <- msg:
		default:
			// Drop the message if the client's buffer is full
		}
	}
}

// Clients returns all clients in the given room.
func (rm *RoomManager) Clients(roomName string) []*Client {
	rm.mu.RLock()
	r, ok := rm.rooms[roomName]
	rm.mu.RUnlock()
	if !ok {
		return nil
	}

	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]*Client, 0, len(r.clients))
	for c := range r.clients {
		result = append(result, c)
	}
	return result
}

// RoomCount returns the number of clients in a given room.
func (rm *RoomManager) RoomCount(roomName string) int {
	rm.mu.RLock()
	r, ok := rm.rooms[roomName]
	rm.mu.RUnlock()
	if !ok {
		return 0
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.clients)
}
