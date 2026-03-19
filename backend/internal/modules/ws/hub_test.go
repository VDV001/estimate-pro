package ws

import (
	"encoding/json"
	"testing"
	"time"
)

func TestHub_RegisterUnregister(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	client := &Client{
		UserID:     "user-1",
		ProjectIDs: map[string]bool{"proj-1": true},
		Send:       make(chan []byte, 10),
	}

	hub.Register(client)
	time.Sleep(10 * time.Millisecond)

	if hub.ClientCount() != 1 {
		t.Errorf("client count = %d, want 1", hub.ClientCount())
	}

	hub.Unregister(client)
	time.Sleep(10 * time.Millisecond)

	if hub.ClientCount() != 0 {
		t.Errorf("client count = %d, want 0", hub.ClientCount())
	}
}

func TestHub_BroadcastToProjectMembers(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	// Client in proj-1
	c1 := &Client{
		UserID:     "user-1",
		ProjectIDs: map[string]bool{"proj-1": true},
		Send:       make(chan []byte, 10),
	}
	// Client in proj-2
	c2 := &Client{
		UserID:     "user-2",
		ProjectIDs: map[string]bool{"proj-2": true},
		Send:       make(chan []byte, 10),
	}
	// Client in both
	c3 := &Client{
		UserID:     "user-3",
		ProjectIDs: map[string]bool{"proj-1": true, "proj-2": true},
		Send:       make(chan []byte, 10),
	}

	hub.Register(c1)
	hub.Register(c2)
	hub.Register(c3)
	time.Sleep(10 * time.Millisecond)

	// Broadcast to proj-1
	hub.Broadcast(Event{Type: "test.event", ProjectID: "proj-1", Payload: "hello"})
	time.Sleep(10 * time.Millisecond)

	// c1 and c3 should receive, c2 should not
	if len(c1.Send) != 1 {
		t.Errorf("c1 got %d messages, want 1", len(c1.Send))
	}
	if len(c2.Send) != 0 {
		t.Errorf("c2 got %d messages, want 0", len(c2.Send))
	}
	if len(c3.Send) != 1 {
		t.Errorf("c3 got %d messages, want 1", len(c3.Send))
	}

	// Verify event content
	msg := <-c1.Send
	var event Event
	json.Unmarshal(msg, &event)
	if event.Type != "test.event" {
		t.Errorf("event type = %q, want test.event", event.Type)
	}
}

func TestHub_BroadcastGlobal(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	c1 := &Client{UserID: "u1", ProjectIDs: map[string]bool{"p1": true}, Send: make(chan []byte, 10)}
	c2 := &Client{UserID: "u2", ProjectIDs: map[string]bool{"p2": true}, Send: make(chan []byte, 10)}

	hub.Register(c1)
	hub.Register(c2)
	time.Sleep(10 * time.Millisecond)

	// Broadcast with empty ProjectID — goes to everyone
	hub.Broadcast(Event{Type: "global.event", ProjectID: ""})
	time.Sleep(10 * time.Millisecond)

	if len(c1.Send) != 1 {
		t.Errorf("c1 got %d messages, want 1", len(c1.Send))
	}
	if len(c2.Send) != 1 {
		t.Errorf("c2 got %d messages, want 1", len(c2.Send))
	}
}
