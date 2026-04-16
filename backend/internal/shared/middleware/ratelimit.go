// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package middleware

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/VDV001/estimate-pro/backend/internal/shared/errors"
)

// RateLimit provides per-IP rate limiting with a sliding window.
// Cleanup goroutine exits when ctx is cancelled.
func RateLimit(ctx context.Context, maxRequests int, window time.Duration) func(http.Handler) http.Handler {
	type entry struct {
		count   int
		resetAt time.Time
	}

	var (
		mu      sync.Mutex
		clients = make(map[string]*entry)
	)

	// Cleanup goroutine — remove expired entries every window period.
	go func() {
		ticker := time.NewTicker(window)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				now := time.Now()
				mu.Lock()
				for ip, e := range clients {
					if now.After(e.resetAt) {
						delete(clients, ip)
					}
				}
				mu.Unlock()
			}
		}
	}()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := r.RemoteAddr // chi RealIP middleware already sets this from X-Forwarded-For

			mu.Lock()
			e, ok := clients[ip]
			now := time.Now()

			if !ok || now.After(e.resetAt) {
				e = &entry{count: 0, resetAt: now.Add(window)}
				clients[ip] = e
			}

			e.count++
			count := e.count
			mu.Unlock()

			if count > maxRequests {
				errors.TooManyRequests(w, "rate limit exceeded, try again later")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
