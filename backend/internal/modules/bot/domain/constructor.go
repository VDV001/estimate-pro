// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// NewBotSession constructs a BotSession enforcing invariants:
// non-empty ChatID, UserID, valid Intent, positive TTL.
// Auto ID, CreatedAt=now, UpdatedAt=now, ExpiresAt=now+ttl, Step=0.
func NewBotSession(chatID, userID string, intent IntentType, state json.RawMessage, ttl time.Duration) (*BotSession, error) {
	if chatID == "" {
		return nil, ErrMissingChat
	}
	if userID == "" {
		return nil, ErrMissingUser
	}
	if !intent.IsValid() {
		return nil, ErrInvalidIntent
	}
	if ttl <= 0 {
		return nil, ErrInvalidTTL
	}
	if len(state) == 0 {
		state = json.RawMessage("{}")
	}
	now := time.Now()
	return &BotSession{
		ID:        uuid.New().String(),
		ChatID:    chatID,
		UserID:    userID,
		Intent:    intent,
		State:     state,
		Step:      0,
		ExpiresAt: now.Add(ttl),
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// Advance replaces the session state, increments Step, stamps UpdatedAt.
// Does not validate state JSON — caller is expected to supply marshaled input.
func (s *BotSession) Advance(state json.RawMessage) {
	s.State = state
	s.Step++
	s.UpdatedAt = time.Now()
}

// NewMemoryEntry constructs a MemoryEntry enforcing invariants:
// non-empty UserID/ChatID, Role in {"user","esti"}, non-empty Content.
// Intent is optional. Auto ID, CreatedAt=now.
func NewMemoryEntry(userID, chatID, role, content, intent string) (*MemoryEntry, error) {
	if userID == "" {
		return nil, ErrMissingUser
	}
	if chatID == "" {
		return nil, ErrMissingChat
	}
	if role != "user" && role != "esti" {
		return nil, ErrInvalidRole
	}
	if content == "" {
		return nil, ErrEmptyContent
	}
	return &MemoryEntry{
		ID:        uuid.New().String(),
		UserID:    userID,
		ChatID:    chatID,
		Role:      role,
		Content:   content,
		Intent:    intent,
		CreatedAt: time.Now(),
	}, nil
}
