package jwt

import (
	"testing"
	"time"
)

func TestGeneratePair(t *testing.T) {
	svc := NewService("test-secret-key", 15*time.Minute, 24*time.Hour)

	tests := []struct {
		name   string
		userID string
	}{
		{name: "standard user", userID: "user-123"},
		{name: "uuid user", userID: "550e8400-e29b-41d4-a716-446655440000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pair, err := svc.GeneratePair(tt.userID)
			if err != nil {
				t.Fatalf("GeneratePair() error = %v", err)
			}
			if pair.AccessToken == "" {
				t.Fatal("AccessToken is empty")
			}
			if pair.RefreshToken == "" {
				t.Fatal("RefreshToken is empty")
			}
			if pair.AccessToken == pair.RefreshToken {
				t.Fatal("AccessToken and RefreshToken must be different")
			}
		})
	}
}

func TestValidateAccess_Valid(t *testing.T) {
	tests := []struct {
		name   string
		userID string
	}{
		{name: "simple id", userID: "user-1"},
		{name: "uuid", userID: "550e8400-e29b-41d4-a716-446655440000"},
	}

	svc := NewService("test-secret", 15*time.Minute, 24*time.Hour)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pair, err := svc.GeneratePair(tt.userID)
			if err != nil {
				t.Fatalf("GeneratePair() error = %v", err)
			}

			claims, err := svc.ValidateAccess(pair.AccessToken)
			if err != nil {
				t.Fatalf("ValidateAccess() error = %v", err)
			}
			if claims.UserID != tt.userID {
				t.Fatalf("UserID = %q, want %q", claims.UserID, tt.userID)
			}
		})
	}
}

func TestValidateAccess_RefreshToken(t *testing.T) {
	svc := NewService("test-secret", 15*time.Minute, 24*time.Hour)

	pair, err := svc.GeneratePair("user-1")
	if err != nil {
		t.Fatalf("GeneratePair() error = %v", err)
	}

	_, err = svc.ValidateAccess(pair.RefreshToken)
	if err == nil {
		t.Fatal("ValidateAccess(refreshToken) should return error, got nil")
	}
}

func TestValidateRefresh_Valid(t *testing.T) {
	tests := []struct {
		name   string
		userID string
	}{
		{name: "simple id", userID: "user-42"},
		{name: "uuid", userID: "a1b2c3d4-e5f6-7890-abcd-ef1234567890"},
	}

	svc := NewService("test-secret", 15*time.Minute, 24*time.Hour)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pair, err := svc.GeneratePair(tt.userID)
			if err != nil {
				t.Fatalf("GeneratePair() error = %v", err)
			}

			claims, err := svc.ValidateRefresh(pair.RefreshToken)
			if err != nil {
				t.Fatalf("ValidateRefresh() error = %v", err)
			}
			if claims.UserID != tt.userID {
				t.Fatalf("UserID = %q, want %q", claims.UserID, tt.userID)
			}
		})
	}
}

func TestValidateRefresh_AccessToken(t *testing.T) {
	svc := NewService("test-secret", 15*time.Minute, 24*time.Hour)

	pair, err := svc.GeneratePair("user-1")
	if err != nil {
		t.Fatalf("GeneratePair() error = %v", err)
	}

	_, err = svc.ValidateRefresh(pair.AccessToken)
	if err == nil {
		t.Fatal("ValidateRefresh(accessToken) should return error, got nil")
	}
}

func TestValidateAccess_Expired(t *testing.T) {
	svc := NewService("test-secret", 1*time.Millisecond, 24*time.Hour)

	pair, err := svc.GeneratePair("user-1")
	if err != nil {
		t.Fatalf("GeneratePair() error = %v", err)
	}

	time.Sleep(5 * time.Millisecond)

	_, err = svc.ValidateAccess(pair.AccessToken)
	if err == nil {
		t.Fatal("ValidateAccess() should return error for expired token, got nil")
	}
}

func TestRefreshTTL(t *testing.T) {
	ttl := 7 * 24 * time.Hour
	svc := NewService("secret", 15*time.Minute, ttl)
	if got := svc.RefreshTTL(); got != ttl {
		t.Errorf("RefreshTTL() = %v, want %v", got, ttl)
	}
}

func TestValidateAccess_InvalidSignature(t *testing.T) {
	svc1 := NewService("secret-one", 15*time.Minute, 24*time.Hour)
	svc2 := NewService("secret-two", 15*time.Minute, 24*time.Hour)

	pair, err := svc1.GeneratePair("user-1")
	if err != nil {
		t.Fatalf("GeneratePair() error = %v", err)
	}

	_, err = svc2.ValidateAccess(pair.AccessToken)
	if err == nil {
		t.Fatal("ValidateAccess() should return error for token signed with different secret, got nil")
	}
}
