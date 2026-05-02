// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package domain

import (
	"context"
	"io"
)

// SessionRepository manages bot conversation sessions.
type SessionRepository interface {
	Create(ctx context.Context, session *BotSession) error
	GetActiveByChatID(ctx context.Context, chatID string) (*BotSession, error)
	Update(ctx context.Context, session *BotSession) error
	Delete(ctx context.Context, id string) error
	DeleteExpired(ctx context.Context) error
}

// UserLinkRepository manages links between Telegram users and EstimatePro accounts.
type UserLinkRepository interface {
	Link(ctx context.Context, link *BotUserLink) error
	GetByTelegramUserID(ctx context.Context, telegramUserID int64) (*BotUserLink, error)
	GetByUserID(ctx context.Context, userID string) (*BotUserLink, error)
	Delete(ctx context.Context, telegramUserID int64) error
}

// UserResolver finds an EstimatePro user ID by Telegram user ID.
// Used for auto-linking: when a Telegram user writes to the bot for the first time,
// the resolver checks if their telegram_user_id matches a users.telegram_chat_id.
type UserResolver interface {
	ResolveByTelegramID(ctx context.Context, telegramUserID int64) (userID string, err error)
}

// LLMConfigRepository manages LLM provider configurations.
type LLMConfigRepository interface {
	GetSystem(ctx context.Context) (*LLMConfig, error)
	GetByUserID(ctx context.Context, userID string) (*LLMConfig, error)
	Upsert(ctx context.Context, cfg *LLMConfig) error
}

// LLMParser parses user messages into structured intents using an LLM.
type LLMParser interface {
	ParseIntent(ctx context.Context, message string, history []string) (*Intent, error)
}

// MemoryRepository stores and retrieves conversation history for context.
type MemoryRepository interface {
	Save(ctx context.Context, entry *MemoryEntry) error
	GetRecent(ctx context.Context, userID string, limit int) ([]*MemoryEntry, error)
	DeleteOld(ctx context.Context, userID string, keepLast int) error
}

// UserPrefsRepository stores learned user preferences.
type UserPrefsRepository interface {
	Get(ctx context.Context, userID string) (*UserPrefs, error)
	Upsert(ctx context.Context, prefs *UserPrefs) error
}

// TelegramClient provides methods for interacting with the Telegram Bot API.
type TelegramClient interface {
	SendMessage(ctx context.Context, chatID string, text string) error
	SendMarkdown(ctx context.Context, chatID string, text string) error
	SendInlineKeyboard(ctx context.Context, chatID string, text string, keyboard [][]InlineKeyboardButton) error
	AnswerCallbackQuery(ctx context.Context, callbackQueryID string, text string) error
	SetReaction(ctx context.Context, chatID string, messageID int64, emoji string) error
	GetFileURL(ctx context.Context, fileID string) (string, error)
	DownloadFile(ctx context.Context, url string) ([]byte, error)
}

// LLMFormatter formats raw action results into human-like bot responses.
type LLMFormatter interface {
	Format(ctx context.Context, actionResult string, intentType IntentType) (string, error)
}

// ProjectManager provides project operations for the bot module.
type ProjectManager interface {
	Create(ctx context.Context, workspaceID string, name string, description string, userID string) (string, error)
	Update(ctx context.Context, projectID string, name string, description string, userID string) error
	ListByUser(ctx context.Context, userID string, limit int, offset int) ([]ProjectSummary, int, error)
}

// MemberManager provides member operations for the bot module.
type MemberManager interface {
	AddByEmail(ctx context.Context, projectID string, email string, role string, callerID string) error
	Remove(ctx context.Context, projectID string, userID string, callerID string) error
	List(ctx context.Context, projectID string) ([]MemberSummary, error)
}

// EstimationManager provides estimation operations for the bot module.
type EstimationManager interface {
	GetAggregated(ctx context.Context, projectID string) (string, error)
	// SubmitItem creates a single-item estimation for the given task and
	// immediately submits it on behalf of the user. Used by the bot
	// `submit_estimation` intent (e.g. "Оценка для задачи X в проекте Y:
	// min 8, likely 12, max 20").
	//
	// Returns ErrInvalidEstimationHours when min/likely/max violate the
	// domain invariant (min ≤ likely ≤ max, all ≥ 0). Other errors are
	// treated as internal.
	SubmitItem(ctx context.Context, projectID, userID, taskName string, minHours, likelyHours, maxHours float64) error
	// RequestEstimation marks a task as needing estimation in the given
	// project, notifying participants. Used by the bot `request_estimation`
	// intent.
	//
	// May return ErrFeatureNotImplemented if the adapter is a placeholder
	// pending real notify-dispatcher integration (tracked in issue #24).
	// Executor maps this to a "feature in development" user-message.
	RequestEstimation(ctx context.Context, projectID, userID, taskName string) error
}

// DocumentManager provides document operations for the bot module.
type DocumentManager interface {
	Upload(ctx context.Context, projectID string, title string, fileName string, fileSize int64, fileType string, content io.Reader, userID string) error
}
