package errors_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/VDV001/estimate-pro/backend/internal/shared/errors"
)

func TestErrorResponses(t *testing.T) {
	tests := []struct {
		name       string
		fn         func(http.ResponseWriter, string)
		msg        string
		wantStatus int
		wantCode   string
	}{
		{"BadRequest", errors.BadRequest, "bad input", http.StatusBadRequest, "BAD_REQUEST"},
		{"Unauthorized", errors.Unauthorized, "not authed", http.StatusUnauthorized, "UNAUTHORIZED"},
		{"Forbidden", errors.Forbidden, "no access", http.StatusForbidden, "FORBIDDEN"},
		{"NotFound", errors.NotFound, "missing", http.StatusNotFound, "NOT_FOUND"},
		{"Conflict", errors.Conflict, "duplicate", http.StatusConflict, "CONFLICT"},
		{"TooManyRequests", errors.TooManyRequests, "slow down", http.StatusTooManyRequests, "TOO_MANY_REQUESTS"},
		{"InternalError", errors.InternalError, "oops", http.StatusInternalServerError, "INTERNAL_ERROR"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			tc.fn(rec, tc.msg)

			if rec.Code != tc.wantStatus {
				t.Fatalf("status: got %d, want %d", rec.Code, tc.wantStatus)
			}

			ct := rec.Header().Get("Content-Type")
			if ct != "application/json" {
				t.Fatalf("Content-Type: got %q, want application/json", ct)
			}

			var resp errors.ErrorResponse
			if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
				t.Fatalf("decode: %v", err)
			}

			if resp.Error.Code != tc.wantCode {
				t.Errorf("code: got %q, want %q", resp.Error.Code, tc.wantCode)
			}
			if resp.Error.Message != tc.msg {
				t.Errorf("message: got %q, want %q", resp.Error.Message, tc.msg)
			}
		})
	}
}
