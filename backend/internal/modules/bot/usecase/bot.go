// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package usecase

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/VDV001/estimate-pro/backend/internal/modules/bot/domain"
	"github.com/VDV001/estimate-pro/backend/internal/modules/bot/llm"
	"github.com/VDV001/estimate-pro/backend/internal/modules/bot/telegram"
)

// EnvLLMConfig holds LLM configuration from environment variables (fallback).
type EnvLLMConfig struct {
	Provider string
	APIKey   string
	Model    string
	BaseURL  string
}

// botNameAliases are informal names users might call the bot in group chats.
var botNameAliases = []string{
	"эсти", "эстя", "эстик", "эстюша",
	"esti", "esty",
}

// BotUsecase orchestrates message processing, intent resolution, and session management.
type BotUsecase struct {
	sessions    *SessionManager
	links       domain.UserLinkRepository
	llmConfigs  domain.LLMConfigRepository
	memory      domain.MemoryRepository
	prefs       domain.UserPrefsRepository
	telegram    domain.TelegramClient
	executor    *IntentExecutor
	llmFactory  func(domain.LLMProviderType, string, string, string) (domain.LLMParser, error)
	envLLM      EnvLLMConfig
	botUsername string
	formatter   *llm.Formatter // LLM #2 — personality formatter
}

const memoryLimit = 20 // last N messages to keep per user

// New creates a new BotUsecase with all dependencies.
func New(
	sessionRepo domain.SessionRepository,
	links domain.UserLinkRepository,
	llmConfigs domain.LLMConfigRepository,
	memoryRepo domain.MemoryRepository,
	prefsRepo domain.UserPrefsRepository,
	tg domain.TelegramClient,
	llmFactory func(domain.LLMProviderType, string, string, string) (domain.LLMParser, error),
	envLLM EnvLLMConfig,
	botUsername string,
	projects domain.ProjectManager,
	members domain.MemberManager,
	estimations domain.EstimationManager,
	documents domain.DocumentManager,
	passwords domain.PasswordResetManager,
) *BotUsecase {
	return &BotUsecase{
		sessions:    NewSessionManager(sessionRepo),
		links:       links,
		llmConfigs:  llmConfigs,
		memory:      memoryRepo,
		prefs:       prefsRepo,
		telegram:    tg,
		executor:    NewIntentExecutor(projects, members, estimations, documents, passwords),
		llmFactory:  llmFactory,
		envLLM:      envLLM,
		botUsername: botUsername,
		formatter:   llm.NewFormatter(domain.LLMProviderType(envLLM.Provider), envLLM.APIKey, envLLM.Model, envLLM.BaseURL),
	}
}

// ProcessMessage handles an incoming Telegram message update.
func (uc *BotUsecase) ProcessMessage(ctx context.Context, update *telegram.Update) error {
	if update.Message == nil {
		return nil
	}

	msg := update.Message
	chatID := strconv.FormatInt(msg.Chat.ID, 10)

	// Group chat: only process if bot is mentioned or replied to.
	if msg.Chat.Type == "group" || msg.Chat.Type == "supergroup" {
		if !uc.isBotMentioned(msg) {
			return nil
		}
	}

	text := uc.stripBotMention(msg.Text)
	msgID := msg.MessageID

	// Input filter — detect prompt injection attempts.
	if isPromptInjection(text) {
		_ = uc.telegram.SetReaction(ctx, chatID, msgID, "🤔")
		_ = uc.telegram.SendMessage(ctx, chatID, deflectionResponse())
		return nil
	}

	// Look up linked user.
	link, err := uc.links.GetByTelegramUserID(ctx, msg.From.ID)
	if err != nil {
		_ = uc.telegram.SendMessage(ctx, chatID, llm.UnlinkedUser.Pick())
		return nil //nolint:nilerr // unlinked user is not an error
	}

	// Handle file attachments (PDF, DOCX, XLSX, MD, TXT, CSV).
	if msg.Document != nil {
		return uc.handleFileUpload(ctx, msg, chatID, link.UserID)
	}

	// Check for active session — continue the flow.
	session, err := uc.sessions.GetActive(ctx, chatID)
	if err == nil {
		return uc.handleSessionMessage(ctx, session, text, link.UserID)
	}

	// Resolve LLM configuration.
	parser, err := uc.resolveLLMParser(ctx, link.UserID)
	if err != nil {
		slog.ErrorContext(ctx, "BotUsecase.ProcessMessage: failed to resolve LLM parser", slog.String("error", err.Error()))
		_ = uc.telegram.SendMessage(ctx, chatID, llm.LLMConfigError.Pick())
		return nil
	}

	// Load conversation history for context.
	var history []string
	if memories, err := uc.memory.GetRecent(ctx, link.UserID, 10); err == nil {
		for _, m := range memories {
			history = append(history, m.Role+": "+m.Content)
		}
	}

	// Parse intent (LLM #1 — classifier, no personality).
	intent, err := parser.ParseIntent(ctx, text, history)
	if err != nil {
		slog.ErrorContext(ctx, "BotUsecase.ProcessMessage: ParseIntent failed", slog.String("error", err.Error()))
		_ = uc.telegram.SetReaction(ctx, chatID, msgID, "🤔")
		_ = uc.telegram.SendMessage(ctx, chatID, llm.LLMError.Pick())
		return nil
	}

	// Low confidence — react and ask to rephrase.
	if intent.Confidence < 0.5 {
		_ = uc.telegram.SetReaction(ctx, chatID, msgID, "🤔")
		_ = uc.telegram.SendMessage(ctx, chatID, llm.LowConfidence.Pick())
		return nil
	}

	// Set reaction on the original message.
	if reaction := llm.FormatReaction(intent.Type); reaction != "" {
		_ = uc.telegram.SetReaction(ctx, chatID, msgID, reaction)
	}

	// Execute intent.
	result, keyboard, err := uc.executor.Execute(ctx, intent, link.UserID)
	if err != nil {
		slog.ErrorContext(ctx, "BotUsecase.ProcessMessage: Execute failed", slog.String("error", err.Error()))
		_ = uc.telegram.SendMessage(ctx, chatID, llm.ExecuteError.Pick())
		return nil
	}

	// Format result through LLM #2 (Esti personality) — only for text responses.
	if keyboard == nil && uc.formatter != nil {
		if formatted, err := uc.formatter.Format(ctx, result, intent.Type); err == nil {
			result = formatted
		}
	}

	// Send response.
	if keyboard != nil {
		_ = uc.telegram.SendInlineKeyboard(ctx, chatID, result, keyboard)
	} else {
		_ = uc.telegram.SendMarkdown(ctx, chatID, result)
	}

	// Save conversation to memory (async, non-blocking).
	// Use context.WithoutCancel so the goroutine is not cancelled when the request ends.
	go uc.saveMemory(context.WithoutCancel(ctx), link.UserID, chatID, text, result, string(intent.Type))

	// Start session if the intent requires one.
	if NeedsSession(intent.Type) {
		_, err = uc.sessions.Create(ctx, chatID, link.UserID, intent.Type, intent.Params)
		if err != nil {
			slog.ErrorContext(ctx, "BotUsecase.ProcessMessage: failed to create session", slog.String("error", err.Error()))
		}
	}

	return nil
}

// supportedFileTypes maps MIME types to EstimatePro file type strings.
var supportedFileTypes = map[string]string{
	"application/pdf":                                                          "pdf",
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document":  "docx",
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":        "xlsx",
	"text/markdown":    "md",
	"text/plain":       "txt",
	"text/csv":         "csv",
	"application/csv":  "csv",
}

// fileTypeFromName guesses file type from extension when MIME is missing.
func fileTypeFromName(name string) string {
	ext := strings.ToLower(name)
	switch {
	case strings.HasSuffix(ext, ".pdf"):
		return "pdf"
	case strings.HasSuffix(ext, ".docx"):
		return "docx"
	case strings.HasSuffix(ext, ".xlsx"):
		return "xlsx"
	case strings.HasSuffix(ext, ".md"):
		return "md"
	case strings.HasSuffix(ext, ".txt"):
		return "txt"
	case strings.HasSuffix(ext, ".csv"):
		return "csv"
	default:
		return ""
	}
}

// handleFileUpload downloads a file from Telegram and starts a session to upload it to a project.
func (uc *BotUsecase) handleFileUpload(ctx context.Context, msg *telegram.Message, chatID, userID string) error {
	doc := msg.Document
	msgID := msg.MessageID

	// Determine file type.
	fileType := supportedFileTypes[doc.MimeType]
	if fileType == "" {
		fileType = fileTypeFromName(doc.FileName)
	}
	if fileType == "" {
		_ = uc.telegram.SetReaction(ctx, chatID, msgID, "🤔")
		_ = uc.telegram.SendMessage(ctx, chatID, "Этот формат я пока не поддерживаю. Скинь PDF, DOCX, XLSX, MD, TXT или CSV!")
		return nil
	}

	// Check file size before downloading.
	const maxBotFileSize = 50 << 20 // 50MB, same as document module

	if doc.FileSize > maxBotFileSize {
		_ = uc.telegram.SendMessage(ctx, chatID, "Файл слишком большой (макс 50MB)")
		return nil
	}

	_ = uc.telegram.SetReaction(ctx, chatID, msgID, "👀")

	// Check if there's an active session with project context.
	session, err := uc.sessions.GetActive(ctx, chatID)
	if err == nil {
		state, _ := uc.sessions.GetState(session)
		if projectID := state["project_id"]; projectID != "" {
			return uc.uploadFile(ctx, chatID, userID, projectID, doc, fileType)
		}
	}

	// No project context — ask which project to upload to.
	projects, _, err := uc.executor.projects.ListByUser(ctx, userID, 20, 0)
	if err != nil || len(projects) == 0 {
		_ = uc.telegram.SendMessage(ctx, chatID, "У тебя пока нет проектов. Сначала создай проект!")
		return nil
	}

	// Save file info in session, show project selection.
	state := map[string]string{
		"file_id":   doc.FileID,
		"file_name": doc.FileName,
		"file_type": fileType,
		"file_size": strconv.FormatInt(doc.FileSize, 10),
	}
	_, err = uc.sessions.Create(ctx, chatID, userID, domain.IntentUploadDocument, state)
	if err != nil {
		slog.ErrorContext(ctx, "handleFileUpload: session create failed", slog.String("error", err.Error()))
	}

	// Build project selection keyboard.
	var keyboard [][]domain.InlineKeyboardButton
	for _, p := range projects {
		keyboard = append(keyboard, []domain.InlineKeyboardButton{
			{Text: p.Name, CallbackData: "sel_proj:" + p.ID},
		})
	}
	keyboard = append(keyboard, []domain.InlineKeyboardButton{
		{Text: "Отмена", CallbackData: "cancel:"},
	})

	_ = uc.telegram.SendInlineKeyboard(ctx, chatID, fmt.Sprintf("Файл *%s* получен! В какой проект загрузить?", doc.FileName), keyboard)
	return nil
}

// uploadFile downloads the file from Telegram and uploads it to the project.
func (uc *BotUsecase) uploadFile(ctx context.Context, chatID, userID, projectID string, doc *telegram.Document, fileType string) error {
	// Get download URL.
	fileURL, err := uc.telegram.GetFileURL(ctx, doc.FileID)
	if err != nil {
		slog.ErrorContext(ctx, "uploadFile: GetFileURL failed", slog.String("error", err.Error()))
		_ = uc.telegram.SendMessage(ctx, chatID, llm.LLMError.Pick())
		return nil
	}

	// Download file content.
	data, err := uc.telegram.DownloadFile(ctx, fileURL)
	if err != nil {
		slog.ErrorContext(ctx, "uploadFile: DownloadFile failed", slog.String("error", err.Error()))
		_ = uc.telegram.SendMessage(ctx, chatID, llm.ExecuteError.Pick())
		return nil
	}

	// Upload to EstimatePro.
	title := doc.FileName
	if idx := strings.LastIndex(title, "."); idx > 0 {
		title = title[:idx] // strip extension for title
	}

	reader := bytes.NewReader(data)
	err = uc.executor.documents.Upload(ctx, projectID, title, doc.FileName, doc.FileSize, fileType, reader, userID)
	if err != nil {
		slog.ErrorContext(ctx, "uploadFile: Upload failed", slog.String("error", err.Error()))
		_ = uc.telegram.SendMessage(ctx, chatID, llm.ExecuteError.Pick())
		return nil
	}

	_ = uc.telegram.SendMarkdown(ctx, chatID, fmt.Sprintf("Файл *%s* загружен в проект! 📎", doc.FileName))

	// Save to memory.
	go uc.saveMemory(context.WithoutCancel(ctx), userID, chatID, "Загрузил файл: "+doc.FileName, "Файл загружен", string(domain.IntentUploadDocument))

	return nil
}

// saveMemory stores user message and bot response in conversation history.
func (uc *BotUsecase) saveMemory(ctx context.Context, userID, chatID, userMsg, botResponse, intent string) {
	if userEntry, err := domain.NewMemoryEntry(userID, chatID, "user", userMsg, intent); err == nil {
		_ = uc.memory.Save(ctx, userEntry)
	}
	if estiEntry, err := domain.NewMemoryEntry(userID, chatID, "esti", botResponse, intent); err == nil {
		_ = uc.memory.Save(ctx, estiEntry)
	}
	// Trim old memories.
	_ = uc.memory.DeleteOld(ctx, userID, memoryLimit)
}

// ProcessCallback handles an incoming Telegram callback query update.
func (uc *BotUsecase) ProcessCallback(ctx context.Context, update *telegram.Update) error {
	cb := update.CallbackQuery
	if cb == nil {
		return nil
	}

	chatID := strconv.FormatInt(cb.Message.Chat.ID, 10)

	// Parse callback data: "action:payload"
	parts := strings.SplitN(cb.Data, ":", 2)
	action := parts[0]
	payload := ""
	if len(parts) > 1 {
		payload = parts[1]
	}

	// Look up linked user.
	link, err := uc.links.GetByTelegramUserID(ctx, cb.From.ID)
	if err != nil {
		_ = uc.telegram.SendMessage(ctx, chatID, "Привяжите аккаунт в настройках EstimatePro.")
		_ = uc.telegram.AnswerCallbackQuery(ctx, cb.ID, "")
		return nil //nolint:nilerr // unlinked user is not an error
	}

	switch {
	case action == "cancel":
		session, sErr := uc.sessions.GetActive(ctx, chatID)
		if sErr == nil {
			_ = uc.sessions.Cancel(ctx, session.ID)
		}
		_ = uc.telegram.SendMessage(ctx, chatID, "Отменено.")
		_ = uc.telegram.AnswerCallbackQuery(ctx, cb.ID, "Отменено")

	case action == "confirm":
		session, sErr := uc.sessions.GetActive(ctx, chatID)
		if sErr != nil {
			_ = uc.telegram.SendMessage(ctx, chatID, "Нет активной сессии.")
			_ = uc.telegram.AnswerCallbackQuery(ctx, cb.ID, "")
			return nil
		}

		if err := uc.executeSessionAction(ctx, session, link.UserID); err != nil {
			slog.ErrorContext(ctx, "BotUsecase.ProcessCallback: executeSessionAction failed", slog.String("error", err.Error()))
			_ = uc.telegram.SendMessage(ctx, chatID, "Ошибка при выполнении действия.")
		} else {
			_ = uc.telegram.SendMessage(ctx, chatID, "Готово!")
		}
		_ = uc.sessions.Complete(ctx, session.ID)
		_ = uc.telegram.AnswerCallbackQuery(ctx, cb.ID, "")

	case strings.HasPrefix(action, "sel_"):
		session, sErr := uc.sessions.GetActive(ctx, chatID)
		if sErr != nil {
			_ = uc.telegram.AnswerCallbackQuery(ctx, cb.ID, "")
			return nil
		}

		// Handle file upload project selection.
		if session.Intent == domain.IntentUploadDocument && action == "sel_proj" {
			state, _ := uc.sessions.GetState(session)
			fileSize, _ := strconv.ParseInt(state["file_size"], 10, 64)
			doc := &telegram.Document{
				FileID:   state["file_id"],
				FileName: state["file_name"],
				FileSize: fileSize,
			}
			_ = uc.sessions.Complete(ctx, session.ID)
			_ = uc.telegram.AnswerCallbackQuery(ctx, cb.ID, "Загружаю...")
			return uc.uploadFile(ctx, chatID, link.UserID, payload, doc, state["file_type"])
		}

		selKey := strings.TrimPrefix(action, "sel_")
		_ = uc.sessions.Advance(ctx, session, map[string]string{selKey: payload})
		_ = uc.telegram.AnswerCallbackQuery(ctx, cb.ID, "")

	default:
		_ = uc.telegram.AnswerCallbackQuery(ctx, cb.ID, "")
		_ = payload // suppress unused warning for default case
	}

	return nil
}

// handleSessionMessage processes a text message within an active session flow.
func (uc *BotUsecase) handleSessionMessage(ctx context.Context, session *domain.BotSession, text string, userID string) error {
	chatID := session.ChatID

	switch session.Intent {
	case domain.IntentCreateProject:
		return uc.handleCreateProjectSession(ctx, session, text, userID, chatID)
	case domain.IntentAddMember:
		return uc.handleAddMemberSession(ctx, session, text, userID, chatID)
	default:
		// For unhandled session intents, advance with raw text.
		return uc.sessions.Advance(ctx, session, map[string]string{"input": text})
	}
}

func (uc *BotUsecase) handleCreateProjectSession(ctx context.Context, session *domain.BotSession, text, userID, chatID string) error {
	switch session.Step {
	case 0:
		// Step 0: we got the project name.
		if err := uc.sessions.Advance(ctx, session, map[string]string{"name": text}); err != nil {
			return fmt.Errorf("BotUsecase.handleCreateProjectSession: %w", err)
		}
		_ = uc.telegram.SendMessage(ctx, chatID, "Отлично! Теперь введите описание проекта (или 'пропустить').")
		return nil
	case 1:
		// Step 1: we got the description — create the project.
		state, err := uc.sessions.GetState(session)
		if err != nil {
			return fmt.Errorf("BotUsecase.handleCreateProjectSession: %w", err)
		}
		description := text
		if strings.EqualFold(description, "пропустить") {
			description = ""
		}

		projectID, err := uc.executor.projects.Create(ctx, "", state["name"], description, userID)
		if err != nil {
			_ = uc.telegram.SendMessage(ctx, chatID, "Ошибка при создании проекта.")
			return fmt.Errorf("BotUsecase.handleCreateProjectSession: %w", err)
		}

		_ = uc.telegram.SendMarkdown(ctx, chatID, fmt.Sprintf("Проект *%s* создан\\! ID: `%s`", state["name"], projectID))
		return uc.sessions.Complete(ctx, session.ID)
	default:
		return uc.sessions.Complete(ctx, session.ID)
	}
}

func (uc *BotUsecase) handleAddMemberSession(ctx context.Context, session *domain.BotSession, text, userID, chatID string) error {
	switch session.Step {
	case 0:
		// Step 0: got project ID.
		if err := uc.sessions.Advance(ctx, session, map[string]string{"project_id": text}); err != nil {
			return fmt.Errorf("BotUsecase.handleAddMemberSession: %w", err)
		}
		_ = uc.telegram.SendMessage(ctx, chatID, "Введите email участника.")
		return nil
	case 1:
		// Step 1: got email — add member.
		state, err := uc.sessions.GetState(session)
		if err != nil {
			return fmt.Errorf("BotUsecase.handleAddMemberSession: %w", err)
		}

		if err := uc.executor.members.AddByEmail(ctx, state["project_id"], text, "developer", userID); err != nil {
			_ = uc.telegram.SendMessage(ctx, chatID, "Ошибка при добавлении участника.")
			return fmt.Errorf("BotUsecase.handleAddMemberSession: %w", err)
		}

		_ = uc.telegram.SendMessage(ctx, chatID, "Участник добавлен!")
		return uc.sessions.Complete(ctx, session.ID)
	default:
		return uc.sessions.Complete(ctx, session.ID)
	}
}

// executeSessionAction runs the final action for a confirmed session.
func (uc *BotUsecase) executeSessionAction(ctx context.Context, session *domain.BotSession, userID string) error {
	state, err := uc.sessions.GetState(session)
	if err != nil {
		return fmt.Errorf("BotUsecase.executeSessionAction: %w", err)
	}

	switch session.Intent {
	case domain.IntentCreateProject:
		_, err = uc.executor.projects.Create(ctx, "", state["name"], state["description"], userID)
	case domain.IntentAddMember:
		err = uc.executor.members.AddByEmail(ctx, state["project_id"], state["email"], state["role"], userID)
	case domain.IntentRemoveMember:
		err = uc.executor.members.Remove(ctx, state["project_id"], state["user_id"], userID)
	default:
		return nil
	}

	if err != nil {
		return fmt.Errorf("BotUsecase.executeSessionAction: %w", err)
	}
	return nil
}

// isBotMentioned checks if the bot is mentioned in a group message.
func (uc *BotUsecase) isBotMentioned(msg *telegram.Message) bool {
	lower := strings.ToLower(msg.Text)

	// Check @username mention.
	if uc.botUsername != "" && strings.Contains(lower, "@"+strings.ToLower(uc.botUsername)) {
		return true
	}

	// Check informal name aliases (Эсти, Эстя, Эстик, etc.).
	for _, alias := range botNameAliases {
		if strings.Contains(lower, alias) {
			return true
		}
	}

	// Check if this is a reply to the bot's own message.
	if msg.ReplyToMessage != nil && msg.ReplyToMessage.From != nil && msg.ReplyToMessage.From.IsBot {
		return true
	}

	return false
}

// stripBotMention removes @botUsername and name aliases from the message text.
func (uc *BotUsecase) stripBotMention(text string) string {
	// Strip @username.
	if uc.botUsername != "" {
		text = strings.ReplaceAll(text, "@"+uc.botUsername, "")
	}

	// Strip name aliases (case-insensitive).
	lower := strings.ToLower(text)
	for _, alias := range botNameAliases {
		if idx := strings.Index(lower, alias); idx != -1 {
			text = text[:idx] + text[idx+len(alias):]
			lower = strings.ToLower(text)
		}
	}

	// Clean up: remove leading comma/colon after stripped name, trim spaces.
	text = strings.TrimLeft(text, " ,:")
	return strings.TrimSpace(text)
}

// resolveLLMParser resolves the LLM parser using user config, system config, or env fallback.
func (uc *BotUsecase) resolveLLMParser(ctx context.Context, userID string) (domain.LLMParser, error) {
	// Try user-specific config first.
	cfg, err := uc.llmConfigs.GetByUserID(ctx, userID)
	if err == nil {
		return uc.llmFactory(cfg.Provider, cfg.APIKey, cfg.Model, cfg.BaseURL)
	}

	// Try system-level config.
	cfg, err = uc.llmConfigs.GetSystem(ctx)
	if err == nil {
		return uc.llmFactory(cfg.Provider, cfg.APIKey, cfg.Model, cfg.BaseURL)
	}

	// Fall back to environment config.
	if uc.envLLM.Provider == "" {
		return nil, domain.ErrNoLLMConfig
	}

	return uc.llmFactory(
		domain.LLMProviderType(uc.envLLM.Provider),
		uc.envLLM.APIKey,
		uc.envLLM.Model,
		uc.envLLM.BaseURL,
	)
}
