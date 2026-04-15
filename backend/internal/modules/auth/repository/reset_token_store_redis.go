// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/VDV001/estimate-pro/backend/internal/modules/auth/domain"
)

// RedisResetTokenStore stores password reset tokens in Redis with TTL-based auto-expiry.
// Each token maps to a userID and is consumed atomically (one-time use).
type RedisResetTokenStore struct {
	client *redis.Client
}

func NewRedisResetTokenStore(client *redis.Client) *RedisResetTokenStore {
	return &RedisResetTokenStore{client: client}
}

func (s *RedisResetTokenStore) key(token string) string {
	return fmt.Sprintf("password_reset:%s", token)
}

func (s *RedisResetTokenStore) Save(ctx context.Context, token, userID string, ttl time.Duration) error {
	if err := s.client.Set(ctx, s.key(token), userID, ttl).Err(); err != nil {
		return fmt.Errorf("ResetTokenStore.Save: %w", err)
	}
	return nil
}

// Consume retrieves the userID for the token and deletes it atomically.
func (s *RedisResetTokenStore) Consume(ctx context.Context, token string) (string, error) {
	key := s.key(token)
	// GetDel is atomic: get + delete in one round trip.
	userID, err := s.client.GetDel(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return "", domain.ErrResetTokenNotFound
		}
		return "", fmt.Errorf("ResetTokenStore.Consume: %w", err)
	}
	return userID, nil
}
