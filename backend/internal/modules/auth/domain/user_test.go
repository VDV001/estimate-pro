// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package domain_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/VDV001/estimate-pro/backend/internal/modules/auth/domain"
)

func TestNewUser_Valid(t *testing.T) {
	u, err := domain.NewUser("alice@example.com", "hash", "Alice", "")
	if err != nil {
		t.Fatalf("NewUser: %v", err)
	}
	if u.ID == "" {
		t.Error("ID must be auto-generated")
	}
	if u.Email != "alice@example.com" || u.Name != "Alice" {
		t.Errorf("fields wrong: %+v", u)
	}
	if u.PasswordHash != "hash" {
		t.Errorf("PasswordHash = %q", u.PasswordHash)
	}
	if u.PreferredLocale != "ru" {
		t.Errorf("PreferredLocale = %q, want default 'ru'", u.PreferredLocale)
	}
	if u.CreatedAt.IsZero() || u.UpdatedAt.IsZero() {
		t.Error("timestamps must be set")
	}
}

func TestNewUser_OAuthNoPassword(t *testing.T) {
	// OAuth users have empty PasswordHash — allowed.
	u, err := domain.NewUser("bob@example.com", "", "Bob", "https://ex/avatar.png")
	if err != nil {
		t.Fatalf("NewUser: %v", err)
	}
	if u.PasswordHash != "" {
		t.Errorf("OAuth PasswordHash must stay empty, got %q", u.PasswordHash)
	}
	if u.AvatarURL != "https://ex/avatar.png" {
		t.Errorf("AvatarURL = %q", u.AvatarURL)
	}
}

func TestNewUser_Validation(t *testing.T) {
	tests := []struct {
		name     string
		email    string
		userName string
		want     error
	}{
		{"empty email", "", "Alice", domain.ErrInvalidEmail},
		{"whitespace email", "   ", "Alice", domain.ErrInvalidEmail},
		{"empty name", "a@b.com", "", domain.ErrInvalidName},
		{"whitespace name", "a@b.com", "   ", domain.ErrInvalidName},
		{"name too long", "a@b.com", strings.Repeat("x", 256), domain.ErrInvalidName},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := domain.NewUser(tc.email, "hash", tc.userName, "")
			if !errors.Is(err, tc.want) {
				t.Errorf("err = %v, want %v", err, tc.want)
			}
		})
	}
}

func TestUser_UpdateProfile(t *testing.T) {
	u, _ := domain.NewUser("a@b.com", "hash", "Alice", "")
	before := u.UpdatedAt

	if err := u.UpdateProfile("Alice Updated", "url", "chat-id", "notify@x.com"); err != nil {
		t.Fatalf("UpdateProfile: %v", err)
	}
	if u.Name != "Alice Updated" {
		t.Errorf("Name = %q", u.Name)
	}
	if u.AvatarURL != "url" || u.TelegramChatID != "chat-id" || u.NotificationEmail != "notify@x.com" {
		t.Errorf("fields not updated: %+v", u)
	}
	if !u.UpdatedAt.After(before) {
		t.Error("UpdatedAt must advance")
	}
}

func TestUser_UpdateProfile_InvalidName(t *testing.T) {
	u, _ := domain.NewUser("a@b.com", "hash", "Alice", "")

	err := u.UpdateProfile(strings.Repeat("x", 256), "", "", "")
	if !errors.Is(err, domain.ErrInvalidName) {
		t.Errorf("err = %v, want ErrInvalidName", err)
	}
	if u.Name != "Alice" {
		t.Error("Name must not mutate on validation error")
	}
}
