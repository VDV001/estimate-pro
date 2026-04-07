// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/VDV001/estimate-pro/backend/internal/modules/bot/domain"
)

// --- Mock SessionRepository ---

type mockSessionRepo struct {
	CreateFn            func(ctx context.Context, session *domain.BotSession) error
	GetActiveByChatIDFn func(ctx context.Context, chatID string) (*domain.BotSession, error)
	UpdateFn            func(ctx context.Context, session *domain.BotSession) error
	DeleteFn            func(ctx context.Context, id string) error
	DeleteExpiredFn     func(ctx context.Context) error
}

func (m *mockSessionRepo) Create(ctx context.Context, session *domain.BotSession) error {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, session)
	}
	return nil
}

func (m *mockSessionRepo) GetActiveByChatID(ctx context.Context, chatID string) (*domain.BotSession, error) {
	if m.GetActiveByChatIDFn != nil {
		return m.GetActiveByChatIDFn(ctx, chatID)
	}
	return nil, domain.ErrSessionNotFound
}

func (m *mockSessionRepo) Update(ctx context.Context, session *domain.BotSession) error {
	if m.UpdateFn != nil {
		return m.UpdateFn(ctx, session)
	}
	return nil
}

func (m *mockSessionRepo) Delete(ctx context.Context, id string) error {
	if m.DeleteFn != nil {
		return m.DeleteFn(ctx, id)
	}
	return nil
}

func (m *mockSessionRepo) DeleteExpired(ctx context.Context) error {
	if m.DeleteExpiredFn != nil {
		return m.DeleteExpiredFn(ctx)
	}
	return nil
}

// --- Tests ---

func TestSessionManager_Create(t *testing.T) {
	var created *domain.BotSession
	repo := &mockSessionRepo{
		CreateFn: func(_ context.Context, s *domain.BotSession) error {
			created = s
			return nil
		},
	}
	sm := NewSessionManager(repo)

	initialState := map[string]string{"name": "Test Project"}
	session, err := sm.Create(t.Context(), "chat-1", "user-1", domain.IntentCreateProject, initialState)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if session.ChatID != "chat-1" {
		t.Errorf("expected ChatID chat-1, got %s", session.ChatID)
	}
	if session.UserID != "user-1" {
		t.Errorf("expected UserID user-1, got %s", session.UserID)
	}
	if session.Intent != domain.IntentCreateProject {
		t.Errorf("expected intent create_project, got %s", session.Intent)
	}
	if session.Step != 0 {
		t.Errorf("expected step 0, got %d", session.Step)
	}
	if session.ExpiresAt.Before(time.Now()) {
		t.Error("expected ExpiresAt to be in the future")
	}
	if created == nil {
		t.Fatal("expected repo.Create to be called")
	}

	// Verify state was marshaled correctly.
	var state map[string]string
	if err := json.Unmarshal(session.State, &state); err != nil {
		t.Fatalf("failed to unmarshal state: %v", err)
	}
	if state["name"] != "Test Project" {
		t.Errorf("expected state name 'Test Project', got %s", state["name"])
	}
}

func TestSessionManager_GetActive_Found(t *testing.T) {
	expected := &domain.BotSession{
		ID:        "ses-1",
		ChatID:    "chat-1",
		UserID:    "user-1",
		Intent:    domain.IntentListProjects,
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}
	repo := &mockSessionRepo{
		GetActiveByChatIDFn: func(_ context.Context, chatID string) (*domain.BotSession, error) {
			if chatID == "chat-1" {
				return expected, nil
			}
			return nil, domain.ErrSessionNotFound
		},
	}
	sm := NewSessionManager(repo)

	session, err := sm.GetActive(t.Context(), "chat-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if session.ID != "ses-1" {
		t.Errorf("expected session ID ses-1, got %s", session.ID)
	}
}

func TestSessionManager_GetActive_Expired(t *testing.T) {
	var deletedID string
	repo := &mockSessionRepo{
		GetActiveByChatIDFn: func(_ context.Context, _ string) (*domain.BotSession, error) {
			return &domain.BotSession{
				ID:        "ses-expired",
				ChatID:    "chat-1",
				ExpiresAt: time.Now().Add(-1 * time.Minute), // already expired
			}, nil
		},
		DeleteFn: func(_ context.Context, id string) error {
			deletedID = id
			return nil
		},
	}
	sm := NewSessionManager(repo)

	_, err := sm.GetActive(t.Context(), "chat-1")
	if !errors.Is(err, domain.ErrSessionNotFound) {
		t.Fatalf("expected ErrSessionNotFound, got %v", err)
	}
	if deletedID != "ses-expired" {
		t.Errorf("expected expired session to be deleted, deletedID=%s", deletedID)
	}
}

func TestSessionManager_Advance(t *testing.T) {
	var updated *domain.BotSession
	repo := &mockSessionRepo{
		UpdateFn: func(_ context.Context, s *domain.BotSession) error {
			updated = s
			return nil
		},
	}
	sm := NewSessionManager(repo)

	initialState, _ := json.Marshal(map[string]string{"name": "Project A"})
	session := &domain.BotSession{
		ID:    "ses-1",
		State: initialState,
		Step:  0,
	}

	err := sm.Advance(t.Context(), session, map[string]string{"description": "Desc B"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if session.Step != 1 {
		t.Errorf("expected step 1 after advance, got %d", session.Step)
	}

	state, err := sm.GetState(session)
	if err != nil {
		t.Fatalf("failed to get state: %v", err)
	}
	if state["name"] != "Project A" {
		t.Errorf("expected original key preserved, got name=%s", state["name"])
	}
	if state["description"] != "Desc B" {
		t.Errorf("expected new key merged, got description=%s", state["description"])
	}
	if updated == nil {
		t.Fatal("expected repo.Update to be called")
	}
}

func TestSessionManager_Complete(t *testing.T) {
	var deletedID string
	repo := &mockSessionRepo{
		DeleteFn: func(_ context.Context, id string) error {
			deletedID = id
			return nil
		},
	}
	sm := NewSessionManager(repo)

	err := sm.Complete(t.Context(), "ses-done")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deletedID != "ses-done" {
		t.Errorf("expected session ses-done to be deleted, got %s", deletedID)
	}
}

func TestSessionManager_GetState(t *testing.T) {
	sm := NewSessionManager(&mockSessionRepo{})

	t.Run("valid state", func(t *testing.T) {
		raw, _ := json.Marshal(map[string]string{"key": "value", "foo": "bar"})
		session := &domain.BotSession{State: raw}

		state, err := sm.GetState(session)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if state["key"] != "value" {
			t.Errorf("expected key=value, got %s", state["key"])
		}
		if state["foo"] != "bar" {
			t.Errorf("expected foo=bar, got %s", state["foo"])
		}
	})

	t.Run("empty state", func(t *testing.T) {
		session := &domain.BotSession{}

		state, err := sm.GetState(session)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(state) != 0 {
			t.Errorf("expected empty state, got %v", state)
		}
	})
}
