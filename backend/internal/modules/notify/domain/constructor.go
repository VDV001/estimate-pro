// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package domain

import (
	"time"

	"github.com/google/uuid"
)

// IsValid reports whether the EventType is a known domain event.
func (e EventType) IsValid() bool {
	switch e {
	case EventMemberAdded, EventDocumentUploaded, EventEstimationSubmitted, EventEstimationAggregated, EventEstimationRequested:
		return true
	}
	return false
}

// NewNotification constructs a Notification enforcing invariants: non-empty
// user, valid event type, non-empty title. ID is auto-generated, Read=false,
// CreatedAt=now. ProjectID is optional.
func NewNotification(userID string, eventType EventType, title, message, projectID string) (*Notification, error) {
	if userID == "" {
		return nil, ErrMissingUser
	}
	if !eventType.IsValid() {
		return nil, ErrInvalidEventType
	}
	if title == "" {
		return nil, ErrMissingTitle
	}
	return &Notification{
		ID:        uuid.New().String(),
		UserID:    userID,
		EventType: eventType,
		Title:     title,
		Message:   message,
		ProjectID: projectID,
		Read:      false,
		CreatedAt: time.Now(),
	}, nil
}

// MarkRead sets Read=true. Idempotent — safe to call multiple times.
func (n *Notification) MarkRead() {
	n.Read = true
}

// NewPreference constructs a Preference enforcing valid user and channel.
func NewPreference(userID string, channel Channel, enabled bool) (*Preference, error) {
	if userID == "" {
		return nil, ErrMissingUser
	}
	if !channel.IsValid() {
		return nil, ErrInvalidChannel
	}
	return &Preference{
		UserID:  userID,
		Channel: channel,
		Enabled: enabled,
	}, nil
}

// NewDeliveryLog constructs a DeliveryLog enforcing valid user/event/channel.
// ID is auto-generated, SentAt=now.
func NewDeliveryLog(userID string, eventType EventType, channel Channel, status string) (*DeliveryLog, error) {
	if userID == "" {
		return nil, ErrMissingUser
	}
	if !eventType.IsValid() {
		return nil, ErrInvalidEventType
	}
	if !channel.IsValid() {
		return nil, ErrInvalidChannel
	}
	return &DeliveryLog{
		ID:        uuid.New().String(),
		UserID:    userID,
		EventType: eventType,
		Channel:   channel,
		SentAt:    time.Now(),
		Status:    status,
	}, nil
}
