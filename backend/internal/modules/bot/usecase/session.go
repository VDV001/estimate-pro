// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/VDV001/estimate-pro/backend/internal/modules/bot/domain"
)

const sessionTTL = 10 * time.Minute

// SessionManager manages bot conversation sessions with TTL-based expiration.
type SessionManager struct {
	repo domain.SessionRepository
}

// NewSessionManager creates a new SessionManager.
func NewSessionManager(repo domain.SessionRepository) *SessionManager {
	return &SessionManager{repo: repo}
}

// GetActive returns the active session for the given chat ID.
// If the session has expired, it is deleted and ErrSessionNotFound is returned.
func (sm *SessionManager) GetActive(ctx context.Context, chatID string) (*domain.BotSession, error) {
	session, err := sm.repo.GetActiveByChatID(ctx, chatID)
	if err != nil {
		return nil, fmt.Errorf("SessionManager.GetActive: %w", err)
	}

	if session.IsExpired() {
		slog.InfoContext(ctx, "SessionManager.GetActive: session expired, deleting", slog.String("session_id", session.ID), slog.String("chat_id", chatID))
		if delErr := sm.repo.Delete(ctx, session.ID); delErr != nil {
			return nil, fmt.Errorf("SessionManager.GetActive: delete expired: %w", delErr)
		}
		return nil, domain.ErrSessionNotFound
	}

	slog.DebugContext(ctx, "SessionManager.GetActive: found", slog.String("session_id", session.ID), slog.String("intent", string(session.Intent)), slog.Int("step", session.Step))
	return session, nil
}

// Create creates a new session for the given chat and user with the specified intent and initial state.
func (sm *SessionManager) Create(
	ctx context.Context,
	chatID string,
	userID string,
	intent domain.IntentType,
	initialState map[string]string,
) (*domain.BotSession, error) {
	stateJSON, err := json.Marshal(initialState)
	if err != nil {
		return nil, fmt.Errorf("SessionManager.Create: marshal state: %w", err)
	}

	session, err := domain.NewBotSession(chatID, userID, intent, stateJSON, sessionTTL)
	if err != nil {
		return nil, err
	}

	if err := sm.repo.Create(ctx, session); err != nil {
		return nil, fmt.Errorf("SessionManager.Create: %w", err)
	}

	slog.InfoContext(ctx, "SessionManager.Create: created", slog.String("session_id", session.ID), slog.String("chat_id", chatID), slog.String("intent", string(intent)))
	return session, nil
}

// Advance merges newData into the session state, increments the step, and persists the update.
func (sm *SessionManager) Advance(ctx context.Context, session *domain.BotSession, newData map[string]string) error {
	slog.DebugContext(ctx, "SessionManager.Advance", slog.String("session_id", session.ID), slog.Int("from_step", session.Step), slog.Any("new_data", newData))
	state, err := sm.GetState(session)
	if err != nil {
		return fmt.Errorf("SessionManager.Advance: %w", err)
	}

	for k, v := range newData {
		state[k] = v
	}

	stateJSON, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("SessionManager.Advance: marshal state: %w", err)
	}

	session.Advance(stateJSON)

	if err := sm.repo.Update(ctx, session); err != nil {
		slog.ErrorContext(ctx, "SessionManager.Advance: Update failed", slog.String("session_id", session.ID), slog.String("error", err.Error()))
		return fmt.Errorf("SessionManager.Advance: %w", err)
	}

	slog.DebugContext(ctx, "SessionManager.Advance: done", slog.String("session_id", session.ID), slog.Int("new_step", session.Step))
	return nil
}

// Complete removes the session after a flow has been completed.
func (sm *SessionManager) Complete(ctx context.Context, sessionID string) error {
	slog.InfoContext(ctx, "SessionManager.Complete", slog.String("session_id", sessionID))
	if err := sm.repo.Delete(ctx, sessionID); err != nil {
		slog.ErrorContext(ctx, "SessionManager.Complete: Delete failed", slog.String("session_id", sessionID), slog.String("error", err.Error()))
		return fmt.Errorf("SessionManager.Complete: %w", err)
	}
	return nil
}

// Cancel removes the session when the user cancels the flow.
func (sm *SessionManager) Cancel(ctx context.Context, sessionID string) error {
	slog.InfoContext(ctx, "SessionManager.Cancel", slog.String("session_id", sessionID))
	if err := sm.repo.Delete(ctx, sessionID); err != nil {
		slog.ErrorContext(ctx, "SessionManager.Cancel: Delete failed", slog.String("session_id", sessionID), slog.String("error", err.Error()))
		return fmt.Errorf("SessionManager.Cancel: %w", err)
	}
	return nil
}

// GetState unmarshals the session's JSON state into a map.
func (sm *SessionManager) GetState(session *domain.BotSession) (map[string]string, error) {
	if len(session.State) == 0 {
		return make(map[string]string), nil
	}

	var state map[string]string
	if err := json.Unmarshal(session.State, &state); err != nil {
		return nil, fmt.Errorf("SessionManager.GetState: %w", err)
	}

	return state, nil
}
