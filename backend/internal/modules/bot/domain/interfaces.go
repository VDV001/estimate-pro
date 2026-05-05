// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package domain

import (
	"context"
	"io"

	sharedllm "github.com/VDV001/estimate-pro/backend/internal/shared/llm"
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

// LLMConfigRepository manages LLM provider configurations. Aliased to
// [sharedllm.LLMConfigRepository] — bot's postgres adapter still owns the
// table but the port is canonical.
type LLMConfigRepository = sharedllm.LLMConfigRepository

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
	// RequestEstimation notifies all project participants (excluding the
	// actor) that an estimation has been requested for the given task.
	// Used by the bot `request_estimation` intent.
	RequestEstimation(ctx context.Context, projectID, userID, taskName string) error
}

// DocumentManager provides document operations for the bot module.
// Upload returns the freshly-created document ID and document
// version ID so the bot can hand them to the ExtractionTrigger
// (PR-B5) without re-querying the document repository.
type DocumentManager interface {
	Upload(ctx context.Context, projectID string, title string, fileName string, fileSize int64, fileType string, content io.Reader, userID string) (documentID string, documentVersionID string, err error)
}

// ExtractionTrigger kicks off an async extraction job for a freshly
// uploaded document version. The bot calls it after a successful
// upload; the actual processing happens out-of-band on the river
// queue (see modules/extractor/worker). Returned extractionID lets
// the caller poll status via Extractor.GetExtraction.
type ExtractionTrigger interface {
	RequestExtraction(ctx context.Context, documentID, documentVersionID string, fileSize int64, actor string) (extractionID string, err error)
}

// Extractor is the composite contract the bot consumes — a single
// dependency satisfying both write (RequestExtraction) and read
// (GetExtraction) sides. Composition root in main.go provides one
// adapter that wraps the extractor module's use case.
type Extractor interface {
	ExtractionTrigger
	ExtractionStatusReader
}

// ExtractionStatus mirrors the extractor module's lifecycle states
// without leaking that module's domain types into bot. The bot
// polls until the status reaches a terminal value (completed /
// failed / cancelled).
type ExtractionStatus string

const (
	ExtractionStatusPending    ExtractionStatus = "pending"
	ExtractionStatusProcessing ExtractionStatus = "processing"
	ExtractionStatusCompleted  ExtractionStatus = "completed"
	ExtractionStatusFailed     ExtractionStatus = "failed"
	ExtractionStatusCancelled  ExtractionStatus = "cancelled"
)

// IsTerminal reports whether the status will not change without a
// new explicit transition (retry / cancel by user). The polling
// loop in handleFileUpload exits as soon as it sees a terminal
// status — no point polling past completed / failed / cancelled.
func (s ExtractionStatus) IsTerminal() bool {
	switch s {
	case ExtractionStatusCompleted, ExtractionStatusFailed, ExtractionStatusCancelled:
		return true
	}
	return false
}

// ExtractedTaskSummary is the bot-side projection of an extracted
// task. Kept as a DTO with public fields — no invariants enforced
// here, the producer (extractor module) already validated them.
type ExtractedTaskSummary struct {
	Name         string
	EstimateHint string
}

// ExtractionResult is what the bot needs to render a follow-up
// message after polling: the current status, the extracted tasks
// (empty unless completed), and a failure reason (empty unless
// failed). Adapters in main.go map extractor module's domain types
// onto this shape so the bot stays free of cross-module imports.
type ExtractionResult struct {
	Status        ExtractionStatus
	Tasks         []ExtractedTaskSummary
	FailureReason string
}

// ExtractionStatusReader fetches the current state of a previously
// requested extraction. Distinct from ExtractionTrigger so each
// interface stays focused (Interface Segregation) — composition
// root in main.go provides one adapter satisfying both via the
// extractor use case.
type ExtractionStatusReader interface {
	GetExtraction(ctx context.Context, extractionID string) (ExtractionResult, error)
}

// Reporter builds a download URL for the rendered report so the
// bot can hand it to the user as a clickable link. Format
// negotiation lives on the web side — Telegram bots can't host
// large binaries inline without extending the telegram client API,
// so the bot intent flow defers actual byte delivery to the
// frontend's existing download path.
type Reporter interface {
	BuildReportURL(ctx context.Context, projectID, format string) (string, error)
}
