package ws

// EventPublisher is used by handlers to broadcast events.
// Pass *Hub as EventPublisher to handlers that need to send real-time updates.
type EventPublisher interface {
	Broadcast(event Event)
}
