package ws

import (
	"encoding/json"
	"sync"
)

// Event represents a real-time event sent to clients.
type Event struct {
	Type      string `json:"type"`      // e.g. "estimation.submitted"
	ProjectID string `json:"project_id"`
	Payload   any    `json:"payload,omitempty"`
}

// Client represents a connected WebSocket client.
type Client struct {
	UserID    string
	ProjectIDs map[string]bool // projects this user is member of
	Send      chan []byte
}

// Hub manages WebSocket connections and broadcasts events.
type Hub struct {
	mu         sync.RWMutex
	clients    map[*Client]bool
	register   chan *Client
	unregister chan *Client
	broadcast  chan Event
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan Event, 256),
	}
}

// Run starts the hub event loop. Call in a goroutine.
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.Send)
			}
			h.mu.Unlock()

		case event := <-h.broadcast:
			data, err := json.Marshal(event)
			if err != nil {
				continue
			}
			h.mu.RLock()
			for client := range h.clients {
				// Only send to clients that are members of the event's project
				if event.ProjectID == "" || client.ProjectIDs[event.ProjectID] {
					select {
					case client.Send <- data:
					default:
						// Client buffer full — skip
					}
				}
			}
			h.mu.RUnlock()
		}
	}
}

// Register adds a client to the hub.
func (h *Hub) Register(client *Client) {
	h.register <- client
}

// Unregister removes a client from the hub.
func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}

// Broadcast sends an event to all relevant clients.
func (h *Hub) Broadcast(event Event) {
	h.broadcast <- event
}

// ClientCount returns the number of connected clients (for testing/monitoring).
func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}
