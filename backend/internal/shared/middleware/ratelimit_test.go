package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/VDV001/estimate-pro/backend/internal/shared/errors"
)

func TestRateLimit(t *testing.T) {
	t.Run("under limit passes", func(t *testing.T) {
		handler := RateLimit(t.Context(), 5, time.Minute)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		for i := range 5 {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.RemoteAddr = "10.0.0.1:1234"
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("request %d: status = %d, want %d", i+1, rec.Code, http.StatusOK)
			}
		}
	})

	t.Run("exceeds limit returns 429", func(t *testing.T) {
		handler := RateLimit(t.Context(), 3, time.Minute)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		// Send 3 allowed requests.
		for range 3 {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.RemoteAddr = "10.0.0.2:1234"
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
		}

		// 4th should be blocked.
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "10.0.0.2:1234"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusTooManyRequests {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusTooManyRequests)
		}

		var errResp errors.ErrorResponse
		if err := json.NewDecoder(rec.Body).Decode(&errResp); err != nil {
			t.Fatalf("decoding error response: %v", err)
		}
		if errResp.Error.Code != "TOO_MANY_REQUESTS" {
			t.Errorf("error code = %q, want %q", errResp.Error.Code, "TOO_MANY_REQUESTS")
		}
	})

	t.Run("different IPs have independent counters", func(t *testing.T) {
		handler := RateLimit(t.Context(), 2, time.Minute)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		// Exhaust limit for IP-A.
		for range 2 {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.RemoteAddr = "10.0.0.3:1234"
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
		}

		// IP-A should be blocked.
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "10.0.0.3:1234"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusTooManyRequests {
			t.Errorf("IP-A status = %d, want %d", rec.Code, http.StatusTooManyRequests)
		}

		// IP-B should still pass.
		req = httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "10.0.0.4:1234"
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("IP-B status = %d, want %d", rec.Code, http.StatusOK)
		}
	})

	t.Run("window reset allows new requests", func(t *testing.T) {
		handler := RateLimit(t.Context(), 1, 50*time.Millisecond)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		// First request passes.
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "10.0.0.5:1234"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("first request: status = %d, want %d", rec.Code, http.StatusOK)
		}

		// Second is blocked.
		req = httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "10.0.0.5:1234"
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusTooManyRequests {
			t.Errorf("second request: status = %d, want %d", rec.Code, http.StatusTooManyRequests)
		}

		// Wait for window to expire.
		time.Sleep(100 * time.Millisecond)

		// Should pass again after window reset.
		req = httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "10.0.0.5:1234"
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("after reset: status = %d, want %d", rec.Code, http.StatusOK)
		}
	})
}
