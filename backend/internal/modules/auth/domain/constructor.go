// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package domain

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

const maxNameLen = 255

// NewUser constructs a User enforcing domain invariants: non-empty trimmed
// email and name (1..255 chars). PasswordHash may be empty for OAuth users.
// AvatarURL is optional. ID is auto-generated; PreferredLocale defaults to "ru".
func NewUser(email, passwordHash, name, avatarURL string) (*User, error) {
	trimmedEmail := strings.TrimSpace(email)
	if trimmedEmail == "" {
		return nil, ErrInvalidEmail
	}
	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" || len(trimmedName) > maxNameLen {
		return nil, ErrInvalidName
	}
	now := time.Now()
	return &User{
		ID:              uuid.New().String(),
		Email:           trimmedEmail,
		PasswordHash:    passwordHash,
		Name:            trimmedName,
		AvatarURL:       avatarURL,
		PreferredLocale: "ru",
		CreatedAt:       now,
		UpdatedAt:       now,
	}, nil
}

// ProfileUpdate holds optional profile fields. Nil pointer = "don't touch",
// non-nil = "set to this value" (including empty string to clear).
type ProfileUpdate struct {
	Name              string
	AvatarURL         *string
	TelegramChatID    *string
	NotificationEmail *string
}

// UpdateProfile applies partial updates. Non-empty Name is validated (1..255
// after trim); empty Name means "keep current". Pointer fields use nil =
// "don't touch", non-nil = "set" (allows clearing). If Name validation fails,
// no field is mutated. UpdatedAt advances on success.
func (u *User) UpdateProfile(upd ProfileUpdate) error {
	var newName string
	if upd.Name != "" {
		trimmed := strings.TrimSpace(upd.Name)
		if trimmed == "" || len(trimmed) > maxNameLen {
			return ErrInvalidName
		}
		newName = trimmed
	}
	if newName != "" {
		u.Name = newName
	}
	if upd.AvatarURL != nil {
		u.AvatarURL = *upd.AvatarURL
	}
	if upd.TelegramChatID != nil {
		u.TelegramChatID = *upd.TelegramChatID
	}
	if upd.NotificationEmail != nil {
		u.NotificationEmail = *upd.NotificationEmail
	}
	u.UpdatedAt = time.Now()
	return nil
}

// SetAvatar updates the avatar URL and stamps UpdatedAt.
func (u *User) SetAvatar(url string) {
	u.AvatarURL = url
	u.UpdatedAt = time.Now()
}
