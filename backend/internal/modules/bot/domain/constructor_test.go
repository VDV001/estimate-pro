// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package domain_test

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/VDV001/estimate-pro/backend/internal/modules/bot/domain"
)

func TestNewBotSession_Valid(t *testing.T) {
	state := json.RawMessage(`{"foo":"bar"}`)
	s, err := domain.NewBotSession("chat-1", "user-1", domain.IntentCreateProject, state, 10*time.Minute)
	if err != nil {
		t.Fatalf("NewBotSession: %v", err)
	}
	if s.ID == "" {
		t.Error("ID must be auto-generated")
	}
	if s.ChatID != "chat-1" || s.UserID != "user-1" {
		t.Errorf("fields wrong: %+v", s)
	}
	if s.Intent != domain.IntentCreateProject {
		t.Errorf("Intent = %q", s.Intent)
	}
	if s.Step != 0 {
		t.Errorf("Step = %d, want 0", s.Step)
	}
	if s.CreatedAt.IsZero() || s.UpdatedAt.IsZero() || s.ExpiresAt.IsZero() {
		t.Error("timestamps must be set")
	}
	if !s.ExpiresAt.After(s.CreatedAt) {
		t.Error("ExpiresAt must be after CreatedAt")
	}
}

func TestNewBotSession_Validation(t *testing.T) {
	tests := []struct {
		name    string
		chatID  string
		userID  string
		intent  domain.IntentType
		ttl     time.Duration
		want    error
	}{
		{"empty chat", "", "u1", domain.IntentHelp, time.Minute, domain.ErrMissingChat},
		{"empty user", "c1", "", domain.IntentHelp, time.Minute, domain.ErrMissingUser},
		{"invalid intent", "c1", "u1", domain.IntentType("bogus"), time.Minute, domain.ErrInvalidIntent},
		{"zero ttl", "c1", "u1", domain.IntentHelp, 0, domain.ErrInvalidTTL},
		{"negative ttl", "c1", "u1", domain.IntentHelp, -time.Second, domain.ErrInvalidTTL},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := domain.NewBotSession(tc.chatID, tc.userID, tc.intent, json.RawMessage("{}"), tc.ttl)
			if !errors.Is(err, tc.want) {
				t.Errorf("err = %v, want %v", err, tc.want)
			}
		})
	}
}

func TestBotSession_Advance(t *testing.T) {
	s, _ := domain.NewBotSession("c1", "u1", domain.IntentHelp, json.RawMessage("{}"), time.Minute)
	before := s.UpdatedAt
	time.Sleep(1 * time.Millisecond)

	s.Advance(json.RawMessage(`{"k":"v"}`))
	if s.Step != 1 {
		t.Errorf("Step = %d, want 1", s.Step)
	}
	if string(s.State) != `{"k":"v"}` {
		t.Errorf("State = %s", s.State)
	}
	if !s.UpdatedAt.After(before) {
		t.Error("UpdatedAt must advance")
	}
}

func TestNewMemoryEntry_Valid(t *testing.T) {
	e, err := domain.NewMemoryEntry("user-1", "chat-1", domain.MemoryRoleUser, "hello", "help")
	if err != nil {
		t.Fatalf("NewMemoryEntry: %v", err)
	}
	if e.ID == "" {
		t.Error("ID must be auto-generated")
	}
	if e.UserID != "user-1" || e.ChatID != "chat-1" || e.Role != domain.MemoryRoleUser {
		t.Errorf("fields wrong: %+v", e)
	}
	if e.Content != "hello" || e.Intent != "help" {
		t.Errorf("content/intent wrong: %+v", e)
	}
	if e.CreatedAt.IsZero() {
		t.Error("CreatedAt must be set")
	}
}

func TestNewMemoryEntry_Validation(t *testing.T) {
	tests := []struct {
		name    string
		userID  string
		chatID  string
		role    domain.MemoryRole
		content string
		want    error
	}{
		{"empty user", "", "c1", domain.MemoryRoleUser, "x", domain.ErrMissingUser},
		{"empty chat", "u1", "", domain.MemoryRoleUser, "x", domain.ErrMissingChat},
		{"invalid role", "u1", "c1", domain.MemoryRole("admin"), "x", domain.ErrInvalidRole},
		{"empty content", "u1", "c1", domain.MemoryRoleUser, "", domain.ErrEmptyContent},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := domain.NewMemoryEntry(tc.userID, tc.chatID, tc.role, tc.content, "")
			if !errors.Is(err, tc.want) {
				t.Errorf("err = %v, want %v", err, tc.want)
			}
		})
	}
}

func TestNewMemoryEntry_RolesValid(t *testing.T) {
	for _, role := range []domain.MemoryRole{domain.MemoryRoleUser, domain.MemoryRoleEsti} {
		t.Run(string(role), func(t *testing.T) {
			_, err := domain.NewMemoryEntry("u1", "c1", role, "x", "")
			if err != nil {
				t.Errorf("role %q should be valid: %v", role, err)
			}
		})
	}
}
