// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/VDV001/estimate-pro/backend/internal/modules/auth/domain"
)

// --- Mock ResetTokenStore ---

type mockResetTokenStore struct {
	tokens map[string]string // token → userID
}

func newMockResetTokenStore() *mockResetTokenStore {
	return &mockResetTokenStore{tokens: make(map[string]string)}
}

func (m *mockResetTokenStore) Save(_ context.Context, token, userID string, _ time.Duration) error {
	m.tokens[token] = userID
	return nil
}

func (m *mockResetTokenStore) Consume(_ context.Context, token string) (string, error) {
	userID, ok := m.tokens[token]
	if !ok {
		return "", domain.ErrResetTokenNotFound
	}
	delete(m.tokens, token)
	return userID, nil
}

// --- Helpers ---

func newResetUsecase(userRepo *mockUserRepo, resetStore *mockResetTokenStore) *AuthUsecase {
	tokenStore := newMockTokenStore()
	uc := New(userRepo, &mockWorkspaceCreator{}, newTestJWT(), tokenStore, &mockAvatarStorage{}, &mockMembershipChecker{})
	uc.SetResetConfig(resetStore, "https://app.example.com", 30*time.Minute)
	return uc
}

// --- Tests ---

func TestForgotPassword_ExistingUser(t *testing.T) {
	resetStore := newMockResetTokenStore()
	userRepo := &mockUserRepo{
		getByEmailFn: func(_ context.Context, _ string) (*domain.User, error) {
			return &domain.User{
				ID:           "user-1",
				Email:        "alice@example.com",
				PasswordHash: hashPassword(t, "secret123"),
			}, nil
		},
	}
	uc := newResetUsecase(userRepo, resetStore)

	out, err := uc.ForgotPassword(t.Context(), "alice@example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Token == "" {
		t.Fatal("expected non-empty token")
	}
	if out.UserID != "user-1" {
		t.Fatalf("expected UserID=user-1, got %q", out.UserID)
	}
	// Token should be saved in the store
	if _, ok := resetStore.tokens[out.Token]; !ok {
		t.Fatal("token not found in reset store")
	}
}

func TestForgotPassword_NonExistentUser(t *testing.T) {
	resetStore := newMockResetTokenStore()
	userRepo := &mockUserRepo{
		getByEmailFn: func(_ context.Context, _ string) (*domain.User, error) {
			return nil, domain.ErrUserNotFound
		},
	}
	uc := newResetUsecase(userRepo, resetStore)

	out, err := uc.ForgotPassword(t.Context(), "nobody@example.com")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if out.Token != "" {
		t.Fatalf("expected empty token, got %q", out.Token)
	}
	if out.UserID != "" {
		t.Fatalf("expected empty UserID, got %q", out.UserID)
	}
}

func TestForgotPassword_OAuthUser(t *testing.T) {
	resetStore := newMockResetTokenStore()
	userRepo := &mockUserRepo{
		getByEmailFn: func(_ context.Context, _ string) (*domain.User, error) {
			return &domain.User{
				ID:           "oauth-user",
				Email:        "oauth@example.com",
				PasswordHash: "", // OAuth — no password
			}, nil
		},
	}
	uc := newResetUsecase(userRepo, resetStore)

	out, err := uc.ForgotPassword(t.Context(), "oauth@example.com")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if out.Token != "" {
		t.Fatalf("expected empty token for OAuth user, got %q", out.Token)
	}
	if out.UserID != "" {
		t.Fatalf("expected empty UserID for OAuth user, got %q", out.UserID)
	}
}

func TestResetPassword_ValidToken(t *testing.T) {
	resetStore := newMockResetTokenStore()
	const oldPassword = "old-password"
	var updatedHash string

	userRepo := &mockUserRepo{
		getByIDFn: func(_ context.Context, id string) (*domain.User, error) {
			if id == "user-1" {
				return &domain.User{
					ID:           "user-1",
					Email:        "alice@example.com",
					PasswordHash: hashPassword(t, oldPassword),
				}, nil
			}
			return nil, domain.ErrUserNotFound
		},
		updateFn: func(_ context.Context, user *domain.User) error {
			updatedHash = user.PasswordHash
			return nil
		},
	}
	tokenStore := newMockTokenStore()
	uc := New(userRepo, &mockWorkspaceCreator{}, newTestJWT(), tokenStore, &mockAvatarStorage{}, &mockMembershipChecker{})
	uc.SetResetConfig(resetStore, "https://app.example.com", 30*time.Minute)

	// Pre-save a reset token
	resetStore.tokens["reset-token-abc"] = "user-1"

	// Pre-save some refresh tokens that should be deleted
	tokenStore.Save(t.Context(), "user-1", "refresh-1", 7*24*time.Hour)
	tokenStore.Save(t.Context(), "user-1", "refresh-2", 7*24*time.Hour)

	err := uc.ResetPassword(t.Context(), ResetPasswordInput{
		Token:       "reset-token-abc",
		NewPassword: "new-secure-password",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Token should be consumed (deleted)
	if _, ok := resetStore.tokens["reset-token-abc"]; ok {
		t.Error("reset token should be consumed after use")
	}

	// Password should be changed
	if updatedHash == "" {
		t.Fatal("expected password hash to be updated")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(updatedHash), []byte("new-secure-password")); err != nil {
		t.Error("new password hash does not match")
	}

	// Refresh tokens should be deleted
	exists1, _ := tokenStore.Exists(t.Context(), "user-1", "refresh-1")
	exists2, _ := tokenStore.Exists(t.Context(), "user-1", "refresh-2")
	if exists1 || exists2 {
		t.Error("all refresh tokens should be deleted after password reset")
	}
}

func TestResetPassword_InvalidToken(t *testing.T) {
	resetStore := newMockResetTokenStore() // empty store
	uc := newResetUsecase(&mockUserRepo{}, resetStore)

	err := uc.ResetPassword(t.Context(), ResetPasswordInput{
		Token:       "nonexistent-token",
		NewPassword: "anything",
	})
	if err == nil {
		t.Fatal("expected error for invalid token")
	}
	if !errors.Is(err, domain.ErrResetTokenNotFound) {
		t.Fatalf("expected ErrResetTokenNotFound, got: %v", err)
	}
}

func TestForgotPasswordByUserID_Success(t *testing.T) {
	resetStore := newMockResetTokenStore()
	userRepo := &mockUserRepo{
		getByIDFn: func(_ context.Context, id string) (*domain.User, error) {
			return &domain.User{
				ID:           id,
				Email:        "alice@example.com",
				PasswordHash: hashPassword(t, "secret"),
			}, nil
		},
	}
	uc := newResetUsecase(userRepo, resetStore)

	link, err := uc.ForgotPasswordByUserID(t.Context(), "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if link == "" {
		t.Fatal("expected non-empty reset link")
	}
	// Link should contain frontend base URL
	if len(link) < len("https://app.example.com/reset-password?token=") {
		t.Fatalf("link too short: %q", link)
	}
	// A token should be saved in the store
	if len(resetStore.tokens) != 1 {
		t.Fatalf("expected 1 token in store, got %d", len(resetStore.tokens))
	}
}

func TestForgotPasswordByUserID_OAuthUser(t *testing.T) {
	userRepo := &mockUserRepo{
		getByIDFn: func(_ context.Context, id string) (*domain.User, error) {
			return &domain.User{
				ID:           id,
				Email:        "oauth@example.com",
				PasswordHash: "", // OAuth — no password
			}, nil
		},
	}
	uc := newResetUsecase(userRepo, newMockResetTokenStore())

	_, err := uc.ForgotPasswordByUserID(t.Context(), "oauth-user")
	if err == nil {
		t.Fatal("expected error for OAuth user")
	}
	if !errors.Is(err, domain.ErrNoPassword) {
		t.Fatalf("expected ErrNoPassword, got: %v", err)
	}
}
