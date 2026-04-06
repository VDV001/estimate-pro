// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package domain

import (
	"context"
	"time"
)

// UserSearchResult is a safe projection of User without sensitive fields.
type UserSearchResult struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url,omitempty"`
}

type UserRepository interface {
	Create(ctx context.Context, user *User) error
	GetByID(ctx context.Context, id string) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	Update(ctx context.Context, user *User) error
	Search(ctx context.Context, query string, excludeUserID string, limit int) ([]*UserSearchResult, error)
	ListColleagues(ctx context.Context, userID string, limit int) ([]*UserSearchResult, error)
}

// WorkspaceCreator creates a personal workspace for a newly registered user.
// Defined here to avoid cross-module import of project domain.
type WorkspaceCreator interface {
	CreatePersonalWorkspace(ctx context.Context, userID, name string) error
}

// MembershipChecker checks if two users share at least one project.
type MembershipChecker interface {
	ShareProject(ctx context.Context, userA, userB string) (bool, error)
}

// AvatarStorage uploads and serves avatar images.
type AvatarStorage interface {
	Upload(ctx context.Context, key string, data []byte, contentType string) (url string, err error)
	Download(ctx context.Context, key string) ([]byte, string, error) // data, contentType, error
}

// TokenStore manages refresh token persistence (Redis-backed).
type TokenStore interface {
	Save(ctx context.Context, userID, tokenID string, ttl time.Duration) error
	Exists(ctx context.Context, userID, tokenID string) (bool, error)
	Delete(ctx context.Context, userID, tokenID string) error
	DeleteAll(ctx context.Context, userID string) error
}
