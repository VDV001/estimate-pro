// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package domain

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

// ErrNoPassword indicates the user has no password (OAuth account).
var ErrNoPassword = errors.New("no password to reset")

// IntentType represents the type of user intent parsed from a message.
type IntentType string

const (
	IntentCreateProject    IntentType = "create_project"
	IntentUpdateProject    IntentType = "update_project"
	IntentListProjects     IntentType = "list_projects"
	IntentGetProjectStatus IntentType = "get_project_status"
	IntentAddMember        IntentType = "add_member"
	IntentRemoveMember     IntentType = "remove_member"
	IntentListMembers      IntentType = "list_members"
	IntentRequestEstimation IntentType = "request_estimation"
	IntentSubmitEstimation IntentType = "submit_estimation"
	IntentGetAggregated    IntentType = "get_aggregated"
	IntentUploadDocument   IntentType = "upload_document"
	IntentForgotPassword   IntentType = "forgot_password"
	IntentHelp             IntentType = "help"
	IntentUnknown          IntentType = "unknown"
)

// IsValid returns true if the intent type is one of the known types.
func (t IntentType) IsValid() bool {
	switch t {
	case IntentCreateProject,
		IntentUpdateProject,
		IntentListProjects,
		IntentGetProjectStatus,
		IntentAddMember,
		IntentRemoveMember,
		IntentListMembers,
		IntentRequestEstimation,
		IntentSubmitEstimation,
		IntentGetAggregated,
		IntentUploadDocument,
		IntentForgotPassword,
		IntentHelp,
		IntentUnknown:
		return true
	default:
		return false
	}
}

// Intent represents a parsed user intent from a chat message.
type Intent struct {
	Type       IntentType        `json:"type"`
	Params     map[string]string `json:"params"`
	Confidence float64           `json:"confidence"`
	RawText    string            `json:"raw_text"`
}

// BotSession represents an active conversation session with the bot.
type BotSession struct {
	ID        string          `json:"id"`
	ChatID    string          `json:"chat_id"`
	UserID    string          `json:"user_id"`
	Intent    IntentType      `json:"intent"`
	State     json.RawMessage `json:"state"`
	Step      int             `json:"step"`
	ExpiresAt time.Time       `json:"expires_at,omitzero"`
	CreatedAt time.Time       `json:"created_at,omitzero"`
	UpdatedAt time.Time       `json:"updated_at,omitzero"`
}

// IsExpired returns true if the session has expired.
func (s *BotSession) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// BotUserLink represents a link between a Telegram user and an EstimatePro user.
type BotUserLink struct {
	TelegramUserID   int64     `json:"telegram_user_id"`
	UserID           string    `json:"user_id"`
	TelegramUsername string    `json:"telegram_username"`
	LinkedAt         time.Time `json:"linked_at"`
}

// LLMProviderType represents a supported LLM provider.
type LLMProviderType string

const (
	ProviderClaude LLMProviderType = "claude"
	ProviderOpenAI LLMProviderType = "openai"
	ProviderGrok   LLMProviderType = "grok"
	ProviderOllama LLMProviderType = "ollama"
)

// IsValid returns true if the provider type is one of the known providers.
func (p LLMProviderType) IsValid() bool {
	switch p {
	case ProviderClaude,
		ProviderOpenAI,
		ProviderGrok,
		ProviderOllama:
		return true
	default:
		return false
	}
}

// LLMConfig represents configuration for an LLM provider.
type LLMConfig struct {
	ID        string          `json:"id"`
	UserID    string          `json:"user_id,omitempty"`
	Provider  LLMProviderType `json:"provider"`
	APIKey    string          `json:"-"`
	Model     string          `json:"model"`
	BaseURL   string          `json:"base_url,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

// ProjectSummary is a lightweight project representation for bot responses.
type ProjectSummary struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Status      string `json:"status"`
	MemberCount int    `json:"member_count"`
}

// MemberSummary is a lightweight member representation for bot responses.
type MemberSummary struct {
	UserID    string `json:"user_id"`
	UserName  string `json:"user_name"`
	UserEmail string `json:"user_email"`
	Role      string `json:"role"`
}

// InlineKeyboardButton represents a Telegram inline keyboard button.
type InlineKeyboardButton struct {
	Text         string `json:"text"`
	CallbackData string `json:"callback_data"`
}

// MemoryEntry is a single message in conversation history.
type MemoryEntry struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	ChatID    string    `json:"chat_id"`
	Role      string    `json:"role"` // "user" or "esti"
	Content   string    `json:"content"`
	Intent    string    `json:"intent,omitempty"`
	CreatedAt time.Time `json:"created_at,omitzero"`
}

// CommunicationStyle represents how the user prefers to interact.
type CommunicationStyle string

const (
	StyleCasual CommunicationStyle = "casual"
	StyleFormal CommunicationStyle = "formal"
	StyleBrief  CommunicationStyle = "brief"
)

// UserPrefs stores learned preferences about a user.
type UserPrefs struct {
	UserID    string             `json:"user_id"`
	Style     CommunicationStyle `json:"style"`
	Language  string             `json:"language"`
	Notes     string             `json:"notes"` // LLM-generated observations
	UpdatedAt time.Time          `json:"updated_at,omitzero"`
}

// PasswordResetManager generates password reset links.
type PasswordResetManager interface {
	RequestReset(ctx context.Context, userID string) (resetLink string, err error)
}
