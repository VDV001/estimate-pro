package config

import (
	"cmp"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	ServerPort      string
	DatabaseURL     string
	RedisURL        string
	AllowedOrigins  []string
	LogLevel        slog.Level
	S3              S3Config
	JWT             JWTConfig
	Composio        ComposioConfig
	OAuth           OAuthConfig
	TelegramBot     TelegramBotConfig
	LLM             LLMDefaultConfig
	Media           MediaConfig
	FrontendBaseURL string
	Extractor       ExtractorConfig
	Generator       GeneratorConfig
}

// MediaConfig holds the dedicated provider keys for the bot's
// photo-OCR (Anthropic Claude Vision) and voice-STT (OpenAI Whisper)
// pipelines (issue #8). Both keys are optional — when missing, the
// bot replies with a "feature unavailable" message rather than
// dereferencing a nil adapter. Models default to current production
// IDs so a bare ANTHROPIC_API_KEY / OPENAI_API_KEY is enough to opt in.
type MediaConfig struct {
	AnthropicAPIKey string
	VisionModel     string
	OpenAIAPIKey    string
	WhisperModel    string
}

// ExtractorConfig gates the document-pipeline module (PR-B series).
// Enabled defaults to false so the feature stays dormant on a fresh
// deploy until ops explicitly opts in via FEATURE_DOCUMENT_PIPELINE_ENABLED.
type ExtractorConfig struct {
	Enabled  bool
	MaxBytes int64
}

// GeneratorConfig holds the document-generator wiring (PR-B4).
// GotenbergURL points at the sidecar (defaults to the docker-compose
// service); GotenbergTimeout caps every conversion round-trip.
// When the URL is empty the converter is left nil — local-dev
// without the sidecar still gets working MD/PDF/DOCX-fill paths
// from shared/generator.Composite.
type GeneratorConfig struct {
	GotenbergURL     string
	GotenbergTimeout time.Duration
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
		LogLevel:       parseLogLevel(os.Getenv("LOG_LEVEL")),
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
		Media: MediaConfig{
			AnthropicAPIKey: os.Getenv("ANTHROPIC_API_KEY"),
			VisionModel:     cmp.Or(os.Getenv("VISION_MODEL"), "claude-sonnet-4-20250514"),
			OpenAIAPIKey:    os.Getenv("OPENAI_API_KEY"),
			WhisperModel:    cmp.Or(os.Getenv("WHISPER_MODEL"), "whisper-1"),
		},
		Extractor: ExtractorConfig{
			Enabled:  os.Getenv("FEATURE_DOCUMENT_PIPELINE_ENABLED") == "true",
			MaxBytes: parseInt64(os.Getenv("EXTRACTOR_MAX_DOCUMENT_BYTES"), 50<<20),
		},
		Generator: GeneratorConfig{
			GotenbergURL:     os.Getenv("GOTENBERG_URL"),
			GotenbergTimeout: parseDuration(os.Getenv("GOTENBERG_TIMEOUT"), 5*time.Minute),
		},
	}
}

func parseInt64(s string, fallback int64) int64 {
	if s == "" {
		return fallback
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return fallback
	}
	return v
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

func parseLogLevel(s string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
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
