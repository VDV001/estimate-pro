// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package usecase

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/VDV001/estimate-pro/backend/internal/modules/bot/domain"
	tg "github.com/VDV001/estimate-pro/backend/internal/modules/bot/telegram"
)

// --- Mock repositories for BotUsecase tests ---

type mockUserLinkRepo struct {
	GetByTelegramUserIDFn func(ctx context.Context, telegramUserID int64) (*domain.BotUserLink, error)
	GetByUserIDFn         func(ctx context.Context, userID string) (*domain.BotUserLink, error)
	LinkFn                func(ctx context.Context, link *domain.BotUserLink) error
	DeleteFn              func(ctx context.Context, telegramUserID int64) error
}

func (m *mockUserLinkRepo) GetByTelegramUserID(ctx context.Context, id int64) (*domain.BotUserLink, error) {
	if m.GetByTelegramUserIDFn != nil {
		return m.GetByTelegramUserIDFn(ctx, id)
	}
	return nil, domain.ErrUserNotLinked
}

func (m *mockUserLinkRepo) GetByUserID(ctx context.Context, id string) (*domain.BotUserLink, error) {
	if m.GetByUserIDFn != nil {
		return m.GetByUserIDFn(ctx, id)
	}
	return nil, domain.ErrUserNotLinked
}

func (m *mockUserLinkRepo) Link(ctx context.Context, link *domain.BotUserLink) error {
	if m.LinkFn != nil {
		return m.LinkFn(ctx, link)
	}
	return nil
}

func (m *mockUserLinkRepo) Delete(ctx context.Context, id int64) error {
	if m.DeleteFn != nil {
		return m.DeleteFn(ctx, id)
	}
	return nil
}

type mockLLMConfigRepo struct {
	GetSystemFn   func(ctx context.Context) (*domain.LLMConfig, error)
	GetByUserIDFn func(ctx context.Context, userID string) (*domain.LLMConfig, error)
	UpsertFn      func(ctx context.Context, cfg *domain.LLMConfig) error
}

func (m *mockLLMConfigRepo) GetSystem(ctx context.Context) (*domain.LLMConfig, error) {
	if m.GetSystemFn != nil {
		return m.GetSystemFn(ctx)
	}
	return nil, domain.ErrNoLLMConfig
}

func (m *mockLLMConfigRepo) GetByUserID(ctx context.Context, userID string) (*domain.LLMConfig, error) {
	if m.GetByUserIDFn != nil {
		return m.GetByUserIDFn(ctx, userID)
	}
	return nil, domain.ErrNoLLMConfig
}

func (m *mockLLMConfigRepo) Upsert(ctx context.Context, cfg *domain.LLMConfig) error {
	if m.UpsertFn != nil {
		return m.UpsertFn(ctx, cfg)
	}
	return nil
}

type mockTelegramClient struct {
	SendMessageFn         func(ctx context.Context, chatID string, text string) error
	SendMarkdownFn        func(ctx context.Context, chatID string, text string) error
	SendInlineKeyboardFn  func(ctx context.Context, chatID string, text string, keyboard [][]domain.InlineKeyboardButton) error
	AnswerCallbackQueryFn func(ctx context.Context, callbackQueryID string, text string) error
	SetReactionFn         func(ctx context.Context, chatID string, messageID int64, emoji string) error
	GetFileURLFn          func(ctx context.Context, fileID string) (string, error)
	DownloadFileFn        func(ctx context.Context, url string) ([]byte, error)
}

func (m *mockTelegramClient) SendMessage(ctx context.Context, chatID, text string) error {
	if m.SendMessageFn != nil {
		return m.SendMessageFn(ctx, chatID, text)
	}
	return nil
}

func (m *mockTelegramClient) SendMarkdown(ctx context.Context, chatID, text string) error {
	if m.SendMarkdownFn != nil {
		return m.SendMarkdownFn(ctx, chatID, text)
	}
	return nil
}

func (m *mockTelegramClient) SendInlineKeyboard(ctx context.Context, chatID, text string, keyboard [][]domain.InlineKeyboardButton) error {
	if m.SendInlineKeyboardFn != nil {
		return m.SendInlineKeyboardFn(ctx, chatID, text, keyboard)
	}
	return nil
}

func (m *mockTelegramClient) AnswerCallbackQuery(ctx context.Context, callbackQueryID, text string) error {
	if m.AnswerCallbackQueryFn != nil {
		return m.AnswerCallbackQueryFn(ctx, callbackQueryID, text)
	}
	return nil
}

func (m *mockTelegramClient) SetReaction(ctx context.Context, chatID string, messageID int64, emoji string) error {
	if m.SetReactionFn != nil {
		return m.SetReactionFn(ctx, chatID, messageID, emoji)
	}
	return nil
}

func (m *mockTelegramClient) GetFileURL(ctx context.Context, fileID string) (string, error) {
	if m.GetFileURLFn != nil {
		return m.GetFileURLFn(ctx, fileID)
	}
	return "", nil
}

func (m *mockTelegramClient) DownloadFile(ctx context.Context, url string) ([]byte, error) {
	if m.DownloadFileFn != nil {
		return m.DownloadFileFn(ctx, url)
	}
	return nil, nil
}

type mockLLMParser struct {
	ParseIntentFn func(ctx context.Context, message string, history []string) (*domain.Intent, error)
}

func (m *mockLLMParser) ParseIntent(ctx context.Context, message string, history []string) (*domain.Intent, error) {
	if m.ParseIntentFn != nil {
		return m.ParseIntentFn(ctx, message, history)
	}
	return &domain.Intent{Type: domain.IntentUnknown, Confidence: 0.0}, nil
}

type mockProjectManager struct {
	CreateFn     func(ctx context.Context, workspaceID, name, description, userID string) (string, error)
	UpdateFn     func(ctx context.Context, projectID, name, description, userID string) error
	ListByUserFn func(ctx context.Context, userID string, limit, offset int) ([]domain.ProjectSummary, int, error)
}

func (m *mockProjectManager) Create(ctx context.Context, workspaceID, name, description, userID string) (string, error) {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, workspaceID, name, description, userID)
	}
	return "proj-1", nil
}

func (m *mockProjectManager) Update(ctx context.Context, projectID, name, description, userID string) error {
	if m.UpdateFn != nil {
		return m.UpdateFn(ctx, projectID, name, description, userID)
	}
	return nil
}

func (m *mockProjectManager) ListByUser(ctx context.Context, userID string, limit, offset int) ([]domain.ProjectSummary, int, error) {
	if m.ListByUserFn != nil {
		return m.ListByUserFn(ctx, userID, limit, offset)
	}
	return nil, 0, nil
}

type mockMemberManager struct {
	AddByEmailFn func(ctx context.Context, projectID, email, role, callerID string) error
	RemoveFn     func(ctx context.Context, projectID, userID, callerID string) error
	ListFn       func(ctx context.Context, projectID string) ([]domain.MemberSummary, error)
}

func (m *mockMemberManager) AddByEmail(ctx context.Context, projectID, email, role, callerID string) error {
	if m.AddByEmailFn != nil {
		return m.AddByEmailFn(ctx, projectID, email, role, callerID)
	}
	return nil
}

func (m *mockMemberManager) Remove(ctx context.Context, projectID, userID, callerID string) error {
	if m.RemoveFn != nil {
		return m.RemoveFn(ctx, projectID, userID, callerID)
	}
	return nil
}

func (m *mockMemberManager) List(ctx context.Context, projectID string) ([]domain.MemberSummary, error) {
	if m.ListFn != nil {
		return m.ListFn(ctx, projectID)
	}
	return nil, nil
}

type mockEstimationManager struct {
	GetAggregatedFn func(ctx context.Context, projectID string) (string, error)
}

func (m *mockEstimationManager) GetAggregated(ctx context.Context, projectID string) (string, error) {
	if m.GetAggregatedFn != nil {
		return m.GetAggregatedFn(ctx, projectID)
	}
	return "", nil
}

type mockDocumentManager struct {
	UploadFn func(ctx context.Context, projectID, title, fileName string, fileSize int64, fileType string, content io.Reader, userID string) error
}

func (m *mockDocumentManager) Upload(ctx context.Context, projectID, title, fileName string, fileSize int64, fileType string, content io.Reader, userID string) error {
	if m.UploadFn != nil {
		return m.UploadFn(ctx, projectID, title, fileName, fileSize, fileType, content, userID)
	}
	return nil
}

// --- Helper to build a test BotUsecase ---

type mockMemoryRepo struct{}

func (m *mockMemoryRepo) Save(_ context.Context, _ *domain.MemoryEntry) error            { return nil }
func (m *mockMemoryRepo) GetRecent(_ context.Context, _ string, _ int) ([]*domain.MemoryEntry, error) {
	return nil, nil
}
func (m *mockMemoryRepo) DeleteOld(_ context.Context, _ string, _ int) error { return nil }

type mockUserPrefsRepo struct{}

func (m *mockUserPrefsRepo) Get(_ context.Context, _ string) (*domain.UserPrefs, error) {
	return &domain.UserPrefs{Style: domain.StyleCasual, Language: "ru"}, nil
}
func (m *mockUserPrefsRepo) Upsert(_ context.Context, _ *domain.UserPrefs) error { return nil }

type testBotDeps struct {
	sessionRepo *mockSessionRepo
	linkRepo    *mockUserLinkRepo
	llmCfgRepo  *mockLLMConfigRepo
	memoryRepo  *mockMemoryRepo
	prefsRepo   *mockUserPrefsRepo
	tgClient    *mockTelegramClient
	parser      *mockLLMParser
	projects    *mockProjectManager
	members     *mockMemberManager
	estimations *mockEstimationManager
	documents   *mockDocumentManager
}

func newTestBotDeps() *testBotDeps {
	return &testBotDeps{
		sessionRepo: &mockSessionRepo{},
		linkRepo:    &mockUserLinkRepo{},
		llmCfgRepo:  &mockLLMConfigRepo{},
		memoryRepo:  &mockMemoryRepo{},
		prefsRepo:   &mockUserPrefsRepo{},
		tgClient:    &mockTelegramClient{},
		parser:      &mockLLMParser{},
		projects:    &mockProjectManager{},
		members:     &mockMemberManager{},
		estimations: &mockEstimationManager{},
		documents:   &mockDocumentManager{},
	}
}

func (d *testBotDeps) build() *BotUsecase {
	parser := d.parser
	return New(
		d.sessionRepo,
		d.linkRepo,
		d.llmCfgRepo,
		d.memoryRepo,
		d.prefsRepo,
		d.tgClient,
		func(_ domain.LLMProviderType, _, _, _ string) (domain.LLMParser, error) {
			return parser, nil
		},
		EnvLLMConfig{Provider: "claude", APIKey: "test-key", Model: "test-model"},
		"estimate_pro_bot",
		d.projects,
		d.members,
		d.estimations,
		d.documents,
	)
}

// --- Tests ---

func TestProcessMessage_NilMessage(t *testing.T) {
	deps := newTestBotDeps()
	uc := deps.build()

	err := uc.ProcessMessage(t.Context(), &tg.Update{Message: nil})
	if err != nil {
		t.Fatalf("expected nil error for nil message, got: %v", err)
	}
}

func TestProcessMessage_UnlinkedUser(t *testing.T) {
	deps := newTestBotDeps()

	var sentText string
	deps.tgClient.SendMessageFn = func(_ context.Context, _ string, text string) error {
		sentText = text
		return nil
	}
	deps.linkRepo.GetByTelegramUserIDFn = func(_ context.Context, _ int64) (*domain.BotUserLink, error) {
		return nil, domain.ErrUserNotLinked
	}

	uc := deps.build()

	err := uc.ProcessMessage(t.Context(), &tg.Update{
		Message: &tg.Message{
			From: &tg.User{ID: 12345},
			Chat: &tg.Chat{ID: 100, Type: "private"},
			Text: "привет",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(sentText, "привяжи") && !strings.Contains(sentText, "Привяжи") && !strings.Contains(sentText, "EstimatePro") {
		t.Errorf("expected link prompt, got: %s", sentText)
	}
}

func TestProcessMessage_GroupChat_NotMentioned(t *testing.T) {
	deps := newTestBotDeps()

	sendCalled := false
	deps.tgClient.SendMessageFn = func(_ context.Context, _ string, _ string) error {
		sendCalled = true
		return nil
	}

	uc := deps.build()

	err := uc.ProcessMessage(t.Context(), &tg.Update{
		Message: &tg.Message{
			From: &tg.User{ID: 12345},
			Chat: &tg.Chat{ID: 200, Type: "supergroup"},
			Text: "обычное сообщение без упоминания",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sendCalled {
		t.Error("expected no message to be sent for non-mentioned group message")
	}
}

func TestProcessMessage_GroupChat_Mentioned(t *testing.T) {
	deps := newTestBotDeps()

	deps.linkRepo.GetByTelegramUserIDFn = func(_ context.Context, _ int64) (*domain.BotUserLink, error) {
		return &domain.BotUserLink{TelegramUserID: 12345, UserID: "user-1"}, nil
	}

	var parsedMsg string
	deps.parser.ParseIntentFn = func(_ context.Context, message string, _ []string) (*domain.Intent, error) {
		parsedMsg = message
		return &domain.Intent{
			Type:       domain.IntentHelp,
			Confidence: 0.95,
			Params:     map[string]string{},
		}, nil
	}

	var sentMarkdown string
	deps.tgClient.SendMarkdownFn = func(_ context.Context, _ string, text string) error {
		sentMarkdown = text
		return nil
	}

	uc := deps.build()

	err := uc.ProcessMessage(t.Context(), &tg.Update{
		Message: &tg.Message{
			From: &tg.User{ID: 12345},
			Chat: &tg.Chat{ID: 200, Type: "group"},
			Text: "@estimate_pro_bot помощь",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if parsedMsg != "помощь" {
		t.Errorf("expected bot mention stripped, got parsed message: %q", parsedMsg)
	}
	if sentMarkdown == "" {
		t.Error("expected markdown response to be sent")
	}
}

func TestProcessMessage_ParsesIntent_SendsResult(t *testing.T) {
	deps := newTestBotDeps()

	deps.linkRepo.GetByTelegramUserIDFn = func(_ context.Context, _ int64) (*domain.BotUserLink, error) {
		return &domain.BotUserLink{TelegramUserID: 12345, UserID: "user-1"}, nil
	}

	deps.parser.ParseIntentFn = func(_ context.Context, _ string, _ []string) (*domain.Intent, error) {
		return &domain.Intent{
			Type:       domain.IntentListProjects,
			Confidence: 0.9,
			Params:     map[string]string{},
		}, nil
	}

	deps.projects.ListByUserFn = func(_ context.Context, _ string, _, _ int) ([]domain.ProjectSummary, int, error) {
		return []domain.ProjectSummary{
			{ID: "p-1", Name: "Alpha", Status: "active", MemberCount: 3},
			{ID: "p-2", Name: "Beta", Status: "draft", MemberCount: 1},
		}, 2, nil
	}

	var sentMarkdown string
	deps.tgClient.SendMarkdownFn = func(_ context.Context, _ string, text string) error {
		sentMarkdown = text
		return nil
	}

	uc := deps.build()

	err := uc.ProcessMessage(t.Context(), &tg.Update{
		Message: &tg.Message{
			From: &tg.User{ID: 12345},
			Chat: &tg.Chat{ID: 100, Type: "private"},
			Text: "покажи мои проекты",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sentMarkdown == "" {
		t.Fatal("expected markdown message to be sent")
	}
	if sentMarkdown == "" {
		t.Error("expected project list in response")
	}
}

func TestProcessMessage_LowConfidence(t *testing.T) {
	deps := newTestBotDeps()

	deps.linkRepo.GetByTelegramUserIDFn = func(_ context.Context, _ int64) (*domain.BotUserLink, error) {
		return &domain.BotUserLink{TelegramUserID: 12345, UserID: "user-1"}, nil
	}

	deps.parser.ParseIntentFn = func(_ context.Context, _ string, _ []string) (*domain.Intent, error) {
		return &domain.Intent{
			Type:       domain.IntentUnknown,
			Confidence: 0.3,
		}, nil
	}

	var sentText string
	deps.tgClient.SendMessageFn = func(_ context.Context, _ string, text string) error {
		sentText = text
		return nil
	}

	uc := deps.build()

	err := uc.ProcessMessage(t.Context(), &tg.Update{
		Message: &tg.Message{
			From: &tg.User{ID: 12345},
			Chat: &tg.Chat{ID: 100, Type: "private"},
			Text: "абракадабра",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(sentText, "понял") && !strings.Contains(sentText, "переформулир") && !strings.Contains(sentText, "помощь") && !strings.Contains(sentText, "туплю") && !strings.Contains(sentText, "распарсил") && !strings.Contains(sentText, "догнал") {
		t.Errorf("expected low confidence message, got: %s", sentText)
	}
}

func TestProcessMessage_ActiveSession_AdvancesFlow(t *testing.T) {
	deps := newTestBotDeps()

	deps.linkRepo.GetByTelegramUserIDFn = func(_ context.Context, _ int64) (*domain.BotUserLink, error) {
		return &domain.BotUserLink{TelegramUserID: 12345, UserID: "user-1"}, nil
	}

	initialState, _ := json.Marshal(map[string]string{})
	deps.sessionRepo.GetActiveByChatIDFn = func(_ context.Context, _ string) (*domain.BotSession, error) {
		return &domain.BotSession{
			ID:        "ses-1",
			ChatID:    "100",
			UserID:    "user-1",
			Intent:    domain.IntentCreateProject,
			State:     initialState,
			Step:      0,
			ExpiresAt: time.Now().Add(5 * time.Minute),
		}, nil
	}

	var updatedSession *domain.BotSession
	deps.sessionRepo.UpdateFn = func(_ context.Context, s *domain.BotSession) error {
		updatedSession = s
		return nil
	}

	var sentText string
	deps.tgClient.SendMessageFn = func(_ context.Context, _ string, text string) error {
		sentText = text
		return nil
	}

	uc := deps.build()

	err := uc.ProcessMessage(t.Context(), &tg.Update{
		Message: &tg.Message{
			From: &tg.User{ID: 12345},
			Chat: &tg.Chat{ID: 100, Type: "private"},
			Text: "Мой новый проект",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if updatedSession == nil {
		t.Fatal("expected session to be updated (advanced)")
	}
	if updatedSession.Step != 1 {
		t.Errorf("expected step to be 1, got %d", updatedSession.Step)
	}
	if sentText == "" {
		t.Error("expected a prompt for next step to be sent")
	}
}

func TestProcessCallback_Cancel(t *testing.T) {
	deps := newTestBotDeps()

	deps.linkRepo.GetByTelegramUserIDFn = func(_ context.Context, _ int64) (*domain.BotUserLink, error) {
		return &domain.BotUserLink{TelegramUserID: 12345, UserID: "user-1"}, nil
	}

	deps.sessionRepo.GetActiveByChatIDFn = func(_ context.Context, _ string) (*domain.BotSession, error) {
		return &domain.BotSession{
			ID:        "ses-1",
			ChatID:    "100",
			ExpiresAt: time.Now().Add(5 * time.Minute),
		}, nil
	}

	var deletedID string
	deps.sessionRepo.DeleteFn = func(_ context.Context, id string) error {
		deletedID = id
		return nil
	}

	var sentText string
	deps.tgClient.SendMessageFn = func(_ context.Context, _ string, text string) error {
		sentText = text
		return nil
	}

	var answeredCBID string
	deps.tgClient.AnswerCallbackQueryFn = func(_ context.Context, cbID, _ string) error {
		answeredCBID = cbID
		return nil
	}

	uc := deps.build()

	err := uc.ProcessCallback(t.Context(), &tg.Update{
		CallbackQuery: &tg.CallbackQuery{
			ID:   "cb-1",
			From: &tg.User{ID: 12345},
			Message: &tg.Message{
				Chat: &tg.Chat{ID: 100, Type: "private"},
			},
			Data: "cancel",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deletedID != "ses-1" {
		t.Errorf("expected session ses-1 to be deleted, got %s", deletedID)
	}
	if sentText != "Отменено." {
		t.Errorf("expected 'Отменено.' message, got: %s", sentText)
	}
	if answeredCBID != "cb-1" {
		t.Errorf("expected callback query cb-1 to be answered, got %s", answeredCBID)
	}
}

func TestProcessCallback_UnlinkedUser(t *testing.T) {
	deps := newTestBotDeps()

	deps.linkRepo.GetByTelegramUserIDFn = func(_ context.Context, _ int64) (*domain.BotUserLink, error) {
		return nil, domain.ErrUserNotLinked
	}

	var sentText string
	deps.tgClient.SendMessageFn = func(_ context.Context, _ string, text string) error {
		sentText = text
		return nil
	}

	var answeredCB bool
	deps.tgClient.AnswerCallbackQueryFn = func(_ context.Context, _, _ string) error {
		answeredCB = true
		return nil
	}

	uc := deps.build()

	err := uc.ProcessCallback(t.Context(), &tg.Update{
		CallbackQuery: &tg.CallbackQuery{
			ID:   "cb-2",
			From: &tg.User{ID: 99999},
			Message: &tg.Message{
				Chat: &tg.Chat{ID: 100, Type: "private"},
			},
			Data: "confirm",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(sentText, "привяжи") && !strings.Contains(sentText, "Привяжи") && !strings.Contains(sentText, "EstimatePro") {
		t.Errorf("expected link prompt, got: %s", sentText)
	}
	if !answeredCB {
		t.Error("expected callback query to be answered")
	}
}
