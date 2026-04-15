package response_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/VDV001/estimate-pro/backend/internal/shared/response"
)

func TestWriteJSON(t *testing.T) {
	type payload struct {
		Name  string `json:"name"`
		Count int    `json:"count"`
	}

	tests := []struct {
		name   string
		status int
		body   any
	}{
		{"ok", http.StatusOK, payload{Name: "test", Count: 42}},
		{"created", http.StatusCreated, map[string]string{"id": "abc"}},
		{"empty", http.StatusOK, map[string]any{}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			response.WriteJSON(rec, tc.status, tc.body)

			if rec.Code != tc.status {
				t.Fatalf("status: got %d, want %d", rec.Code, tc.status)
			}

			ct := rec.Header().Get("Content-Type")
			if ct != "application/json" {
				t.Fatalf("Content-Type: got %q, want application/json", ct)
			}

			var decoded map[string]any
			if err := json.NewDecoder(rec.Body).Decode(&decoded); err != nil {
				t.Fatalf("decode: %v", err)
			}
		})
	}
}
