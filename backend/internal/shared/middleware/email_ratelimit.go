// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package middleware

import (
	"context"
	"sync"
	"time"
)

// EmailRateLimiter limits per-email request frequency within a sliding window.
// Cleanup goroutine stops when the provided context is cancelled.
type EmailRateLimiter struct {
	mu      sync.Mutex
	entries map[string]*emailRateEntry
	limit   int
	window  time.Duration
}

type emailRateEntry struct {
	count   int
	resetAt time.Time
}

// NewEmailRateLimiter constructs an EmailRateLimiter and starts a cleanup
// goroutine that exits when ctx is cancelled.
func NewEmailRateLimiter(ctx context.Context, limit int, window time.Duration) *EmailRateLimiter {
	rl := &EmailRateLimiter{
		entries: make(map[string]*emailRateEntry),
		limit:   limit,
		window:  window,
	}
	go rl.cleanupLoop(ctx)
	return rl
}

// Allow reports whether the given email can proceed. Returns false if over limit.
func (rl *EmailRateLimiter) Allow(email string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	entry, exists := rl.entries[email]
	if !exists || now.After(entry.resetAt) {
		rl.entries[email] = &emailRateEntry{count: 1, resetAt: now.Add(rl.window)}
		return true
	}
	if entry.count >= rl.limit {
		return false
	}
	entry.count++
	return true
}

func (rl *EmailRateLimiter) cleanupLoop(ctx context.Context) {
	ticker := time.NewTicker(rl.window)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			rl.mu.Lock()
			now := time.Now()
			for email, entry := range rl.entries {
				if now.After(entry.resetAt) {
					delete(rl.entries, email)
				}
			}
			rl.mu.Unlock()
		}
	}
}
