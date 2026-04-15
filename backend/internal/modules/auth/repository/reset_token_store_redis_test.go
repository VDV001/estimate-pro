// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/VDV001/estimate-pro/backend/internal/modules/auth/domain"
)

// newTestRedisClient creates a Redis client for testing.
// Skips the test if Redis is not available at localhost:6379.
func newTestRedisClient(t *testing.T) *redis.Client {
	t.Helper()

	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   15, // use DB 15 to avoid collisions with dev data
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		t.Skipf("Redis not available at localhost:6379: %v", err)
	}

	t.Cleanup(func() {
		_ = client.FlushDB(context.Background())
		_ = client.Close()
	})

	return client
}

func TestResetTokenStore_SaveAndConsume(t *testing.T) {
	client := newTestRedisClient(t)
	store := NewRedisResetTokenStore(client)
	ctx := t.Context()

	token := "test-token-abc123"
	userID := "user-42"

	// Save token.
	if err := store.Save(ctx, token, userID, 5*time.Minute); err != nil {
		t.Fatalf("Save: unexpected error: %v", err)
	}

	// First consume should return the userID.
	got, err := store.Consume(ctx, token)
	if err != nil {
		t.Fatalf("Consume: unexpected error: %v", err)
	}
	if got != userID {
		t.Errorf("Consume: got userID %q, want %q", got, userID)
	}

	// Second consume must fail — token was already consumed.
	_, err = store.Consume(ctx, token)
	if !errors.Is(err, domain.ErrResetTokenNotFound) {
		t.Errorf("second Consume: got error %v, want %v", err, domain.ErrResetTokenNotFound)
	}
}

func TestResetTokenStore_ConsumeExpired(t *testing.T) {
	client := newTestRedisClient(t)
	store := NewRedisResetTokenStore(client)
	ctx := t.Context()

	token := "test-token-expired"
	userID := "user-99"

	// Save with minimal TTL.
	if err := store.Save(ctx, token, userID, 1*time.Millisecond); err != nil {
		t.Fatalf("Save: unexpected error: %v", err)
	}

	// Wait for expiration.
	time.Sleep(10 * time.Millisecond)

	// Consume must fail — token expired.
	_, err := store.Consume(ctx, token)
	if !errors.Is(err, domain.ErrResetTokenNotFound) {
		t.Errorf("Consume expired: got error %v, want %v", err, domain.ErrResetTokenNotFound)
	}
}

func TestResetTokenStore_ConsumeNonExistent(t *testing.T) {
	client := newTestRedisClient(t)
	store := NewRedisResetTokenStore(client)
	ctx := t.Context()

	// Consume a token that was never saved.
	_, err := store.Consume(ctx, "totally-unknown-token")
	if !errors.Is(err, domain.ErrResetTokenNotFound) {
		t.Errorf("Consume non-existent: got error %v, want %v", err, domain.ErrResetTokenNotFound)
	}
}
