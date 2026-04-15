package config

import (
	"testing"
	"time"
)

func TestParseOrigins(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{"empty defaults to localhost", "", []string{"http://localhost:3000"}},
		{"single origin", "https://example.com", []string{"https://example.com"}},
		{"multiple origins", "https://a.com, https://b.com, https://c.com", []string{"https://a.com", "https://b.com", "https://c.com"}},
		{"trims whitespace", "  https://a.com , https://b.com  ", []string{"https://a.com", "https://b.com"}},
		{"skips empty entries", "https://a.com,,https://b.com,", []string{"https://a.com", "https://b.com"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := parseOrigins(tc.input)
			if len(got) != len(tc.want) {
				t.Fatalf("len: got %d, want %d (%v)", len(got), len(tc.want), got)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("[%d]: got %q, want %q", i, got[i], tc.want[i])
				}
			}
		})
	}
}

func TestParseDuration(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		fallback time.Duration
		want     time.Duration
	}{
		{"empty uses fallback", "", 5 * time.Minute, 5 * time.Minute},
		{"valid duration", "30m", time.Hour, 30 * time.Minute},
		{"invalid uses fallback", "not-a-duration", 10 * time.Second, 10 * time.Second},
		{"hours", "2h", time.Minute, 2 * time.Hour},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := parseDuration(tc.input, tc.fallback)
			if got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestLoad_Defaults(t *testing.T) {
	// Clear relevant env vars to test defaults.
	t.Setenv("SERVER_PORT", "")
	t.Setenv("DATABASE_URL", "")
	t.Setenv("REDIS_URL", "")
	t.Setenv("CORS_ALLOWED_ORIGINS", "")
	t.Setenv("S3_ENDPOINT", "")
	t.Setenv("S3_ACCESS_KEY", "")
	t.Setenv("S3_SECRET_KEY", "")
	t.Setenv("S3_BUCKET", "")
	t.Setenv("S3_USE_SSL", "")
	t.Setenv("JWT_SECRET", "")
	t.Setenv("JWT_ACCESS_TTL", "")
	t.Setenv("JWT_REFRESH_TTL", "")
	t.Setenv("OAUTH_REDIRECT_BASE_URL", "")
	t.Setenv("FRONTEND_BASE_URL", "")
	t.Setenv("LLM_PROVIDER", "")
	t.Setenv("LLM_MODEL", "")

	cfg := Load()

	if cfg.ServerPort != "8080" {
		t.Errorf("ServerPort: got %q, want 8080", cfg.ServerPort)
	}
	if cfg.RedisURL != "redis://localhost:6379" {
		t.Errorf("RedisURL: got %q", cfg.RedisURL)
	}
	if len(cfg.AllowedOrigins) != 1 || cfg.AllowedOrigins[0] != "http://localhost:3000" {
		t.Errorf("AllowedOrigins: got %v", cfg.AllowedOrigins)
	}
	if cfg.S3.Endpoint != "localhost:9000" {
		t.Errorf("S3.Endpoint: got %q", cfg.S3.Endpoint)
	}
	if cfg.S3.AccessKey != "minioadmin" {
		t.Errorf("S3.AccessKey: got %q", cfg.S3.AccessKey)
	}
	if cfg.S3.Bucket != "estimatepro" {
		t.Errorf("S3.Bucket: got %q", cfg.S3.Bucket)
	}
	if cfg.S3.UseSSL {
		t.Error("S3.UseSSL should be false by default")
	}
	if cfg.JWT.AccessTTL != 15*time.Minute {
		t.Errorf("JWT.AccessTTL: got %v", cfg.JWT.AccessTTL)
	}
	if cfg.JWT.RefreshTTL != 30*24*time.Hour {
		t.Errorf("JWT.RefreshTTL: got %v", cfg.JWT.RefreshTTL)
	}
	if cfg.OAuth.RedirectBaseURL != "http://localhost:3000" {
		t.Errorf("OAuth.RedirectBaseURL: got %q", cfg.OAuth.RedirectBaseURL)
	}
	if cfg.FrontendBaseURL != "http://localhost:3000" {
		t.Errorf("FrontendBaseURL: got %q", cfg.FrontendBaseURL)
	}
	if cfg.LLM.Provider != "claude" {
		t.Errorf("LLM.Provider: got %q", cfg.LLM.Provider)
	}
}

func TestLoad_CustomValues(t *testing.T) {
	t.Setenv("SERVER_PORT", "9090")
	t.Setenv("DATABASE_URL", "postgres://localhost/test")
	t.Setenv("REDIS_URL", "redis://custom:6380")
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://app.example.com,https://admin.example.com")
	t.Setenv("S3_USE_SSL", "true")
	t.Setenv("JWT_ACCESS_TTL", "30m")
	t.Setenv("JWT_REFRESH_TTL", "168h")
	t.Setenv("LLM_PROVIDER", "openai")
	t.Setenv("LLM_MODEL", "gpt-4")

	cfg := Load()

	if cfg.ServerPort != "9090" {
		t.Errorf("ServerPort: got %q, want 9090", cfg.ServerPort)
	}
	if cfg.DatabaseURL != "postgres://localhost/test" {
		t.Errorf("DatabaseURL: got %q", cfg.DatabaseURL)
	}
	if cfg.RedisURL != "redis://custom:6380" {
		t.Errorf("RedisURL: got %q", cfg.RedisURL)
	}
	if len(cfg.AllowedOrigins) != 2 {
		t.Fatalf("AllowedOrigins: got %v", cfg.AllowedOrigins)
	}
	if cfg.S3.UseSSL != true {
		t.Error("S3.UseSSL should be true")
	}
	if cfg.JWT.AccessTTL != 30*time.Minute {
		t.Errorf("JWT.AccessTTL: got %v", cfg.JWT.AccessTTL)
	}
	if cfg.JWT.RefreshTTL != 168*time.Hour {
		t.Errorf("JWT.RefreshTTL: got %v", cfg.JWT.RefreshTTL)
	}
	if cfg.LLM.Provider != "openai" {
		t.Errorf("LLM.Provider: got %q", cfg.LLM.Provider)
	}
}
