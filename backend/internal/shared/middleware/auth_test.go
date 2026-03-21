package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/VDV001/estimate-pro/backend/internal/shared/errors"
	"github.com/VDV001/estimate-pro/backend/pkg/jwt"
)

func TestAuth(t *testing.T) {
	const (
		secret = "test-secret-key-for-auth-middleware"
		userID = "usr_abc123"
	)

	jwtService := jwt.NewService(secret, 15*time.Minute, 24*time.Hour)

	validToken := func(t *testing.T) string {
		t.Helper()
		pair, err := jwtService.GeneratePair(userID)
		if err != nil {
			t.Fatalf("generating token pair: %v", err)
		}
		return pair.AccessToken
	}

	expiredToken := func(t *testing.T) string {
		t.Helper()
		svc := jwt.NewService(secret, -1*time.Second, 24*time.Hour)
		pair, err := svc.GeneratePair(userID)
		if err != nil {
			t.Fatalf("generating expired token: %v", err)
		}
		return pair.AccessToken
	}

	wrongSecretToken := func(t *testing.T) string {
		t.Helper()
		svc := jwt.NewService("wrong-secret-key", 15*time.Minute, 24*time.Hour)
		pair, err := svc.GeneratePair(userID)
		if err != nil {
			t.Fatalf("generating wrong-secret token: %v", err)
		}
		return pair.AccessToken
	}

	tests := []struct {
		name           string
		authHeader     string
		wantStatus     int
		wantNextCalled bool
		wantUserID     string
		wantErrCode    string
	}{
		{
			name:           "valid bearer token",
			authHeader:     "Bearer " + validToken(t),
			wantStatus:     http.StatusOK,
			wantNextCalled: true,
			wantUserID:     userID,
		},
		{
			name:        "missing authorization header",
			authHeader:  "",
			wantStatus:  http.StatusUnauthorized,
			wantErrCode: "UNAUTHORIZED",
		},
		{
			name:        "no Bearer prefix",
			authHeader:  "Token " + validToken(t),
			wantStatus:  http.StatusUnauthorized,
			wantErrCode: "UNAUTHORIZED",
		},
		{
			name:        "expired token",
			authHeader:  "Bearer " + expiredToken(t),
			wantStatus:  http.StatusUnauthorized,
			wantErrCode: "UNAUTHORIZED",
		},
		{
			name:        "wrong secret",
			authHeader:  "Bearer " + wrongSecretToken(t),
			wantStatus:  http.StatusUnauthorized,
			wantErrCode: "UNAUTHORIZED",
		},
		{
			name:        "empty bearer",
			authHeader:  "Bearer ",
			wantStatus:  http.StatusUnauthorized,
			wantErrCode: "UNAUTHORIZED",
		},
		{
			name:        "malformed JWT",
			authHeader:  "Bearer not.valid.jwt",
			wantStatus:  http.StatusUnauthorized,
			wantErrCode: "UNAUTHORIZED",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				nextCalled  bool
				capturedUID string
			)

			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				nextCalled = true
				uid, _ := UserIDFromContext(r.Context())
				capturedUID = uid
				w.WriteHeader(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			rec := httptest.NewRecorder()

			handler := Auth(jwtService)(next)
			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if nextCalled != tt.wantNextCalled {
				t.Errorf("nextCalled = %v, want %v", nextCalled, tt.wantNextCalled)
			}
			if tt.wantNextCalled && capturedUID != tt.wantUserID {
				t.Errorf("userID = %q, want %q", capturedUID, tt.wantUserID)
			}
			if tt.wantErrCode != "" {
				var errResp errors.ErrorResponse
				if err := json.NewDecoder(rec.Body).Decode(&errResp); err != nil {
					t.Fatalf("decoding error response: %v", err)
				}
				if errResp.Error.Code != tt.wantErrCode {
					t.Errorf("error code = %q, want %q", errResp.Error.Code, tt.wantErrCode)
				}
			}
		})
	}
}

func TestUserIDFromContext(t *testing.T) {
	tests := []struct {
		name   string
		ctx    context.Context
		wantID string
		wantOK bool
	}{
		{
			name:   "value set",
			ctx:    context.WithValue(context.Background(), UserIDKey, "usr_123"),
			wantID: "usr_123",
			wantOK: true,
		},
		{
			name:   "empty context",
			ctx:    context.Background(),
			wantID: "",
			wantOK: false,
		},
		{
			name:   "wrong type",
			ctx:    context.WithValue(context.Background(), UserIDKey, 12345),
			wantID: "",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, ok := UserIDFromContext(tt.ctx)
			if id != tt.wantID {
				t.Errorf("id = %q, want %q", id, tt.wantID)
			}
			if ok != tt.wantOK {
				t.Errorf("ok = %v, want %v", ok, tt.wantOK)
			}
		})
	}
}
