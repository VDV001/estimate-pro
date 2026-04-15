package config

import (
	"cmp"
	"os"
	"strings"
	"time"
)

type Config struct {
	ServerPort     string
	DatabaseURL    string
	RedisURL       string
	AllowedOrigins []string
	S3             S3Config
	JWT            JWTConfig
	Composio       ComposioConfig
	OAuth          OAuthConfig
	TelegramBot    TelegramBotConfig
	LLM            LLMDefaultConfig
	FrontendBaseURL string
}

type S3Config struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	UseSSL    bool
}

type JWTConfig struct {
	Secret     string
	AccessTTL  time.Duration
	RefreshTTL time.Duration
}

type ComposioConfig struct {
	APIKey            string
	GmailAccountID    string
	TelegramAccountID string
}

type OAuthConfig struct {
	GoogleClientID     string
	GoogleClientSecret string
	GitHubClientID     string
	GitHubClientSecret string
	RedirectBaseURL    string
}

type TelegramBotConfig struct {
	Token         string
	WebhookSecret string
	BotUsername   string
}

type LLMDefaultConfig struct {
	Provider string
	APIKey   string
	Model    string
	BaseURL  string
}

func Load() Config {
	return Config{
		ServerPort:     cmp.Or(os.Getenv("SERVER_PORT"), "8080"),
		DatabaseURL:    os.Getenv("DATABASE_URL"),
		RedisURL:       cmp.Or(os.Getenv("REDIS_URL"), "redis://localhost:6379"),
		AllowedOrigins: parseOrigins(os.Getenv("CORS_ALLOWED_ORIGINS")),
		S3: S3Config{
			Endpoint:  cmp.Or(os.Getenv("S3_ENDPOINT"), "localhost:9000"),
			AccessKey: cmp.Or(os.Getenv("S3_ACCESS_KEY"), "minioadmin"),
			SecretKey: cmp.Or(os.Getenv("S3_SECRET_KEY"), "minioadmin"),
			Bucket:    cmp.Or(os.Getenv("S3_BUCKET"), "estimatepro"),
			UseSSL:    os.Getenv("S3_USE_SSL") == "true",
		},
		JWT: JWTConfig{
			Secret:     os.Getenv("JWT_SECRET"),
			AccessTTL:  parseDuration(os.Getenv("JWT_ACCESS_TTL"), 15*time.Minute),
			RefreshTTL: parseDuration(os.Getenv("JWT_REFRESH_TTL"), 30*24*time.Hour),
		},
		Composio: ComposioConfig{
			APIKey:            os.Getenv("COMPOSIO_API_KEY"),
			GmailAccountID:    os.Getenv("COMPOSIO_GMAIL_ACCOUNT_ID"),
			TelegramAccountID: os.Getenv("COMPOSIO_TELEGRAM_ACCOUNT_ID"),
		},
		OAuth: OAuthConfig{
			GoogleClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
			GoogleClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
			GitHubClientID:     os.Getenv("GITHUB_CLIENT_ID"),
			GitHubClientSecret: os.Getenv("GITHUB_CLIENT_SECRET"),
			RedirectBaseURL:    cmp.Or(os.Getenv("OAUTH_REDIRECT_BASE_URL"), "http://localhost:3000"),
		},
		TelegramBot: TelegramBotConfig{
			Token:         os.Getenv("TELEGRAM_BOT_TOKEN"),
			WebhookSecret: os.Getenv("TELEGRAM_WEBHOOK_SECRET"),
			BotUsername:   os.Getenv("TELEGRAM_BOT_USERNAME"),
		},
		FrontendBaseURL: cmp.Or(os.Getenv("FRONTEND_BASE_URL"), "http://localhost:3000"),
		LLM: LLMDefaultConfig{
			Provider: cmp.Or(os.Getenv("LLM_PROVIDER"), "claude"),
			APIKey:   os.Getenv("LLM_API_KEY"),
			Model:    cmp.Or(os.Getenv("LLM_MODEL"), "claude-sonnet-4-20250514"),
			BaseURL:  os.Getenv("LLM_BASE_URL"),
		},
	}
}

func parseOrigins(s string) []string {
	if s == "" {
		return []string{"http://localhost:3000"}
	}
	var origins []string
	for _, o := range strings.Split(s, ",") {
		o = strings.TrimSpace(o)
		if o != "" {
			origins = append(origins, o)
		}
	}
	return origins
}

func parseDuration(s string, fallback time.Duration) time.Duration {
	if s == "" {
		return fallback
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return fallback
	}
	return d
}
