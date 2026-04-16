// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package middleware_test

import (
	"context"
	"testing"
	"time"

	"github.com/VDV001/estimate-pro/backend/internal/shared/middleware"
)

func TestEmailRateLimiter_AllowWithinLimit(t *testing.T) {
	rl := middleware.NewEmailRateLimiter(t.Context(), 3, time.Hour)

	for i := 1; i <= 3; i++ {
		if !rl.Allow("a@b.com") {
			t.Errorf("Allow call %d: want true (under limit)", i)
		}
	}
}

func TestEmailRateLimiter_RejectOverLimit(t *testing.T) {
	rl := middleware.NewEmailRateLimiter(t.Context(), 2, time.Hour)

	rl.Allow("a@b.com")
	rl.Allow("a@b.com")

	if rl.Allow("a@b.com") {
		t.Error("third Allow: want false (over limit)")
	}
}

func TestEmailRateLimiter_DifferentEmailsIndependent(t *testing.T) {
	rl := middleware.NewEmailRateLimiter(t.Context(), 1, time.Hour)

	if !rl.Allow("a@b.com") {
		t.Error("first email: want true")
	}
	if !rl.Allow("c@d.com") {
		t.Error("second email: want true (independent counter)")
	}
	if rl.Allow("a@b.com") {
		t.Error("first email second call: want false (over limit)")
	}
}

func TestEmailRateLimiter_ResetAfterWindow(t *testing.T) {
	rl := middleware.NewEmailRateLimiter(t.Context(), 1, 10*time.Millisecond)

	rl.Allow("a@b.com")
	if rl.Allow("a@b.com") {
		t.Error("immediate retry: want false")
	}

	time.Sleep(20 * time.Millisecond)

	if !rl.Allow("a@b.com") {
		t.Error("after window: want true (counter reset)")
	}
}

func TestEmailRateLimiter_ContextCancelStopsGoroutine(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	rl := middleware.NewEmailRateLimiter(ctx, 1, time.Millisecond)

	rl.Allow("a@b.com")
	cancel()

	// After cancel, cleanup goroutine must exit. We can't directly observe it,
	// but if it leaks, -race / goleak will catch it. Here we just ensure the
	// limiter still functions synchronously after cancel.
	time.Sleep(5 * time.Millisecond)
	_ = rl.Allow("a@b.com") // must not panic/deadlock
}
