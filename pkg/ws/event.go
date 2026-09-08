package ws

import (
	"encoding/json"
	"log"
)

// EventMessage is the standard JSON structure for typed WebSocket events.
// On the client side, send: {"event": "chat.message", "data": {...}}
type EventMessage struct {
	Event string          `json:"event"`
	Data  json.RawMessage `json:"data"`
}

// EventHandler is a function that handles a specific WebSocket event.
type EventHandler func(client *Client, data json.RawMessage)

// EventHub extends Hub with typed event routing.
type EventHub struct {
	*Hub
	handlers map[string]EventHandler
	rooms    *RoomManager
}

// NewEventHub creates a new event-driven WebSocket hub.
func NewEventHub() *EventHub {
	h := &EventHub{
		Hub:      NewHub(),
		handlers: make(map[string]EventHandler),
		rooms:    NewRoomManager(),
	}
	// Wire up the low-level OnMessage to our event router
	h.Hub.OnMessage(func(client *Client, msg []byte) {
		var ev EventMessage
		if err := json.Unmarshal(msg, &ev); err != nil {
			log.Printf("[GoKS/ws] invalid event message: %v", err)
			return
		}
		h.dispatch(client, ev)
	})
	return h
}

// On registers a handler for a specific event name.
//
// Example:
//
//	hub.On("chat.message", func(client *ws.Client, data json.RawMessage) {
//	    var msg ChatMessage
//	    json.Unmarshal(data, &msg)
//	    hub.BroadcastEvent("chat.message", msg)
//	})
func (h *EventHub) On(event string, handler EventHandler) {
	h.handlers[event] = handler
}

// dispatch routes an incoming event to its registered handler.
func (h *EventHub) dispatch(client *Client, ev EventMessage) {
	handler, ok := h.handlers[ev.Event]
	if !ok {
		log.Printf("[GoKS/ws] no handler for event %q", ev.Event)
		return
	}
	handler(client, ev.Data)
}

// Emit sends a typed event to a specific client.
func (h *EventHub) Emit(client *Client, event string, data any) {
	payload, err := json.Marshal(data)
	if err != nil {
		log.Printf("[GoKS/ws] emit marshal error: %v", err)
		return
	}
	msg, _ := json.Marshal(EventMessage{Event: event, Data: payload})
	client.Send(msg)
}

// BroadcastEvent sends a typed event to all connected clients.
func (h *EventHub) BroadcastEvent(event string, data any) {
	payload, err := json.Marshal(data)
	if err != nil {
		log.Printf("[GoKS/ws] broadcast marshal error: %v", err)
		return
	}
	msg, _ := json.Marshal(EventMessage{Event: event, Data: payload})
	h.Hub.Broadcast(msg)
}

// BroadcastRoom sends a typed event to all clients in a room.
func (h *EventHub) BroadcastRoom(room, event string, data any) {
	payload, err := json.Marshal(data)
	if err != nil {
		return
	}
	msg, _ := json.Marshal(EventMessage{Event: event, Data: payload})
	h.rooms.Broadcast(room, msg)
}

// Join adds a client to a room.
func (h *EventHub) Join(room string, client *Client) {
	h.rooms.Join(room, client)
}

// Leave removes a client from a room.
func (h *EventHub) Leave(room string, client *Client) {
	h.rooms.Leave(room, client)
}

// LeaveAll removes a client from all rooms.
func (h *EventHub) LeaveAll(client *Client) {
	h.rooms.LeaveAll(client)
}

// Rooms returns the underlying RoomManager.
func (h *EventHub) Rooms() *RoomManager {
	return h.rooms
}
