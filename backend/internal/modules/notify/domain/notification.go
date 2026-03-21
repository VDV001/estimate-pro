// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package domain

import "time"

// Channel represents a notification delivery channel.
type Channel string

const (
	ChannelInApp    Channel = "in_app"
	ChannelEmail    Channel = "email"
	ChannelTelegram Channel = "telegram"
)

func (c Channel) IsValid() bool {
	switch c {
	case ChannelInApp, ChannelEmail, ChannelTelegram:
		return true
	}
	return false
}

// EventType represents the type of event that triggered the notification.
type EventType string

const (
	EventMemberAdded          EventType = "member.added"
	EventDocumentUploaded     EventType = "document.uploaded"
	EventEstimationSubmitted  EventType = "estimation.submitted"
	EventEstimationAggregated EventType = "estimation.aggregated"
)

// Notification is an in-app notification for a user.
type Notification struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	EventType EventType `json:"event_type"`
	Title     string    `json:"title"`
	Message   string    `json:"message"`
	ProjectID string    `json:"project_id,omitempty"`
	Read      bool      `json:"read"`
	CreatedAt time.Time `json:"created_at,omitzero"`
}

// Preference stores a user's preference for a notification channel.
type Preference struct {
	UserID  string  `json:"user_id"`
	Channel Channel `json:"channel"`
	Enabled bool    `json:"enabled"`
}

// DeliveryLog records a notification sent via external channel.
type DeliveryLog struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	EventType EventType `json:"event_type"`
	Channel   Channel   `json:"channel"`
	SentAt    time.Time `json:"sent_at,omitzero"`
	Status    string    `json:"status"`
}

// NotifyEvent is the input for creating notifications from domain events.
type NotifyEvent struct {
	EventType EventType
	ProjectID string
	ActorID   string
	Title     string
	Message   string
}
