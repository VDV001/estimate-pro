// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package usecase

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

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
	sessions     *SessionManager
	links        domain.UserLinkRepository
	userResolver domain.UserResolver
	llmConfigs   domain.LLMConfigRepository
	memory       domain.MemoryRepository
	prefs        domain.UserPrefsRepository
	telegram     domain.TelegramClient
	executor     *IntentExecutor
	llmFactory   func(domain.LLMProviderType, string, string, string) (domain.LLMParser, error)
	envLLM       EnvLLMConfig
	botUsername  string
	formatter    *llm.Formatter // LLM #2 — personality formatter
}

const memoryLimit = 20 // last N messages to keep per user

// New creates a new BotUsecase with all dependencies.
func New(
	sessionRepo domain.SessionRepository,
	links domain.UserLinkRepository,
	userResolver domain.UserResolver,
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
		sessions:     NewSessionManager(sessionRepo),
		links:        links,
		userResolver: userResolver,
		llmConfigs:   llmConfigs,
		memory:       memoryRepo,
		prefs:        prefsRepo,
		telegram:     tg,
		executor:     NewIntentExecutor(projects, members, estimations, documents, passwords),
		llmFactory:   llmFactory,
		envLLM:       envLLM,
		botUsername:  botUsername,
		formatter:    llm.NewFormatter(domain.LLMProviderType(envLLM.Provider), envLLM.APIKey, envLLM.Model, envLLM.BaseURL),
	}
}

// ProcessMessage handles an incoming Telegram message update.
func (uc *BotUsecase) ProcessMessage(ctx context.Context, update *telegram.Update) error {
	if update.Message == nil {
		return nil
	}

	msg := update.Message
	chatID := strconv.FormatInt(msg.Chat.ID, 10)

	slog.InfoContext(ctx, "BotUsecase.ProcessMessage: incoming",
		slog.String("chat_id", chatID),
		slog.String("chat_type", msg.Chat.Type),
		slog.Int64("from_id", msg.From.ID),
		slog.String("from_username", msg.From.Username),
		slog.Int("text_len", len(msg.Text)),
		slog.Bool("has_document", msg.Document != nil),
	)

	// Group chat: only process if bot is mentioned or replied to.
	if msg.Chat.Type == "group" || msg.Chat.Type == "supergroup" {
		if !uc.isBotMentioned(msg) {
			slog.DebugContext(ctx, "BotUsecase.ProcessMessage: bot not mentioned in group, skipping", slog.String("chat_id", chatID))
			return nil
		}
		slog.DebugContext(ctx, "BotUsecase.ProcessMessage: bot mentioned in group", slog.String("chat_id", chatID))
	}

	text := uc.stripBotMention(msg.Text)
	msgID := msg.MessageID

	// Input filter — detect prompt injection attempts.
	if isPromptInjection(text) {
		slog.WarnContext(ctx, "BotUsecase.ProcessMessage: prompt injection detected", slog.String("chat_id", chatID), slog.Int64("from_id", msg.From.ID))
		_ = uc.telegram.SetReaction(ctx, chatID, msgID, "🤔")
		_ = uc.telegram.SendMessage(ctx, chatID, deflectionResponse())
		return nil
	}

	// Look up linked user, auto-link if possible.
	link, err := uc.links.GetByTelegramUserID(ctx, msg.From.ID)
	if errors.Is(err, domain.ErrUserNotLinked) {
		link, err = uc.tryAutoLink(ctx, msg.From)
	}
	if err != nil {
		slog.WarnContext(ctx, "BotUsecase.ProcessMessage: user not linked", slog.Int64("telegram_user_id", msg.From.ID), slog.String("error", err.Error()))
		_ = uc.telegram.SendMessage(ctx, chatID, llm.UnlinkedUser.Pick())
		return nil //nolint:nilerr // unlinked user is not an error
	}
	slog.InfoContext(ctx, "BotUsecase.ProcessMessage: user linked", slog.String("user_id", link.UserID), slog.Int64("telegram_user_id", msg.From.ID))

	// Handle file attachments (PDF, DOCX, XLSX, MD, TXT, CSV).
	if msg.Document != nil {
		slog.InfoContext(ctx, "BotUsecase.ProcessMessage: file attachment", slog.String("file_name", msg.Document.FileName), slog.String("mime_type", msg.Document.MimeType), slog.Int64("file_size", msg.Document.FileSize))
		return uc.handleFileUpload(ctx, msg, chatID, link.UserID)
	}

	// Check for active session — continue the flow.
	session, err := uc.sessions.GetActive(ctx, chatID)
	if err == nil {
		slog.InfoContext(ctx, "BotUsecase.ProcessMessage: continuing active session", slog.String("session_id", session.ID), slog.String("intent", string(session.Intent)), slog.Int("step", session.Step))
		return uc.handleSessionMessage(ctx, session, text, link.UserID)
	}

	// Resolve LLM configuration.
	slog.DebugContext(ctx, "BotUsecase.ProcessMessage: resolving LLM parser", slog.String("user_id", link.UserID))
	parser, err := uc.resolveLLMParser(ctx, link.UserID)
	if err != nil {
		slog.ErrorContext(ctx, "BotUsecase.ProcessMessage: failed to resolve LLM parser", slog.String("user_id", link.UserID), slog.String("error", err.Error()))
		_ = uc.telegram.SendMessage(ctx, chatID, llm.LLMConfigError.Pick())
		return nil
	}

	// Load conversation history for context.
	var history []string
	if memories, err := uc.memory.GetRecent(ctx, link.UserID, 10); err == nil {
		for _, m := range memories {
			history = append(history, string(m.Role)+": "+m.Content)
		}
		slog.DebugContext(ctx, "BotUsecase.ProcessMessage: loaded memory", slog.Int("entries", len(memories)))
	} else {
		slog.WarnContext(ctx, "BotUsecase.ProcessMessage: failed to load memory", slog.String("user_id", link.UserID), slog.String("error", err.Error()))
	}

	// Parse intent (LLM #1 — classifier, no personality).
	slog.InfoContext(ctx, "BotUsecase.ProcessMessage: calling ParseIntent", slog.String("text", text))
	intent, err := parser.ParseIntent(ctx, text, history)
	if err != nil {
		slog.ErrorContext(ctx, "BotUsecase.ProcessMessage: ParseIntent failed", slog.String("chat_id", chatID), slog.String("text", text), slog.String("error", err.Error()))
		_ = uc.telegram.SetReaction(ctx, chatID, msgID, "🤔")
		_ = uc.telegram.SendMessage(ctx, chatID, llm.LLMError.Pick())
		return nil
	}

	slog.InfoContext(ctx, "BotUsecase.ProcessMessage: intent parsed", slog.String("intent", string(intent.Type)), slog.Float64("confidence", intent.Confidence))

	// Low confidence — react and ask to rephrase.
	if intent.Confidence < 0.5 {
		slog.InfoContext(ctx, "BotUsecase.ProcessMessage: low confidence, asking to rephrase", slog.Float64("confidence", intent.Confidence))
		_ = uc.telegram.SetReaction(ctx, chatID, msgID, "🤔")
		_ = uc.telegram.SendMessage(ctx, chatID, llm.LowConfidence.Pick())
		return nil
	}

	// Set reaction on the original message.
	if reaction := llm.FormatReaction(intent.Type); reaction != "" {
		_ = uc.telegram.SetReaction(ctx, chatID, msgID, reaction)
	}

	// Execute intent.
	slog.InfoContext(ctx, "BotUsecase.ProcessMessage: executing intent", slog.String("intent", string(intent.Type)), slog.String("user_id", link.UserID))
	result, keyboard, err := uc.executor.Execute(ctx, intent, link.UserID)
	if err != nil {
		slog.ErrorContext(ctx, "BotUsecase.ProcessMessage: Execute failed", slog.String("intent", string(intent.Type)), slog.String("user_id", link.UserID), slog.String("error", err.Error()))
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
	slog.InfoContext(ctx, "BotUsecase.ProcessMessage: sending response", slog.String("chat_id", chatID), slog.Bool("has_keyboard", keyboard != nil), slog.Int("result_len", len(result)))
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
	slog.InfoContext(ctx, "BotUsecase.handleFileUpload: start", slog.String("chat_id", chatID), slog.String("user_id", userID), slog.String("file_name", msg.Document.FileName))
	doc := msg.Document
	msgID := msg.MessageID

	// Determine file type.
	fileType := supportedFileTypes[doc.MimeType]
	if fileType == "" {
		fileType = fileTypeFromName(doc.FileName)
	}
	if fileType == "" {
		slog.WarnContext(ctx, "BotUsecase.handleFileUpload: unsupported file type", slog.String("file_name", doc.FileName), slog.String("mime_type", doc.MimeType))
		_ = uc.telegram.SetReaction(ctx, chatID, msgID, "🤔")
		_ = uc.telegram.SendMessage(ctx, chatID, "Этот формат я пока не поддерживаю. Скинь PDF, DOCX, XLSX, MD, TXT или CSV!")
		return nil
	}

	// Check file size before downloading.
	const maxBotFileSize = 50 << 20 // 50MB, same as document module

	if doc.FileSize > maxBotFileSize {
		slog.WarnContext(ctx, "BotUsecase.handleFileUpload: file too large", slog.String("file_name", doc.FileName), slog.Int64("file_size", doc.FileSize))
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
	if err != nil {
		slog.ErrorContext(ctx, "BotUsecase.handleFileUpload: ListByUser failed", slog.String("user_id", userID), slog.String("error", err.Error()))
		_ = uc.telegram.SendMessage(ctx, chatID, "У тебя пока нет проектов. Сначала создай проект!")
		return nil
	}
	if len(projects) == 0 {
		slog.InfoContext(ctx, "BotUsecase.handleFileUpload: no projects for user", slog.String("user_id", userID))
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
			{Text: p.Name, CallbackData: domain.SelectCallback(domain.CallbackKeyProject, p.ID)},
		})
	}
	keyboard = append(keyboard, []domain.InlineKeyboardButton{
		{Text: "Отмена", CallbackData: domain.CancelCallback()},
	})

	_ = uc.telegram.SendInlineKeyboard(ctx, chatID, fmt.Sprintf("Файл *%s* получен! В какой проект загрузить?", doc.FileName), keyboard)
	return nil
}

// uploadFile downloads the file from Telegram and uploads it to the project.
func (uc *BotUsecase) uploadFile(ctx context.Context, chatID, userID, projectID string, doc *telegram.Document, fileType string) error {
	slog.InfoContext(ctx, "BotUsecase.uploadFile: start", slog.String("chat_id", chatID), slog.String("project_id", projectID), slog.String("file_name", doc.FileName), slog.String("file_type", fileType))
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

	slog.InfoContext(ctx, "BotUsecase.uploadFile: success", slog.String("file_name", doc.FileName), slog.String("project_id", projectID))
	_ = uc.telegram.SendMarkdown(ctx, chatID, fmt.Sprintf("Файл *%s* загружен в проект! 📎", doc.FileName))

	// Save to memory.
	go uc.saveMemory(context.WithoutCancel(ctx), userID, chatID, "Загрузил файл: "+doc.FileName, "Файл загружен", string(domain.IntentUploadDocument))

	return nil
}

// saveMemory stores user message and bot response in conversation history.
func (uc *BotUsecase) saveMemory(ctx context.Context, userID, chatID, userMsg, botResponse, intent string) {
	slog.DebugContext(ctx, "BotUsecase.saveMemory: start", slog.String("user_id", userID), slog.String("chat_id", chatID), slog.String("intent", intent))
	if userEntry, err := domain.NewMemoryEntry(userID, chatID, domain.MemoryRoleUser, userMsg, intent); err == nil {
		if err := uc.memory.Save(ctx, userEntry); err != nil {
			slog.ErrorContext(ctx, "BotUsecase.saveMemory: failed to save user entry", slog.String("user_id", userID), slog.String("error", err.Error()))
		}
	}
	if estiEntry, err := domain.NewMemoryEntry(userID, chatID, domain.MemoryRoleEsti, botResponse, intent); err == nil {
		if err := uc.memory.Save(ctx, estiEntry); err != nil {
			slog.ErrorContext(ctx, "BotUsecase.saveMemory: failed to save esti entry", slog.String("user_id", userID), slog.String("error", err.Error()))
		}
	}
	// Trim old memories.
	if err := uc.memory.DeleteOld(ctx, userID, memoryLimit); err != nil {
		slog.WarnContext(ctx, "BotUsecase.saveMemory: failed to trim old memories", slog.String("user_id", userID), slog.String("error", err.Error()))
	}
	slog.DebugContext(ctx, "BotUsecase.saveMemory: done", slog.String("user_id", userID))
}

// ProcessCallback handles an incoming Telegram callback query update.
func (uc *BotUsecase) ProcessCallback(ctx context.Context, update *telegram.Update) error {
	cb := update.CallbackQuery
	if cb == nil {
		return nil
	}

	chatID := strconv.FormatInt(cb.Message.Chat.ID, 10)

	action, payload := domain.ParseCallback(cb.Data)

	slog.InfoContext(ctx, "BotUsecase.ProcessCallback: incoming",
		slog.String("chat_id", chatID),
		slog.String("action", string(action)),
		slog.String("payload", payload),
		slog.Int64("from_id", cb.From.ID),
	)

	// Look up linked user, auto-link if possible.
	link, err := uc.links.GetByTelegramUserID(ctx, cb.From.ID)
	if errors.Is(err, domain.ErrUserNotLinked) {
		link, err = uc.tryAutoLink(ctx, cb.From)
	}
	if err != nil {
		slog.WarnContext(ctx, "BotUsecase.ProcessCallback: user not linked", slog.Int64("telegram_user_id", cb.From.ID), slog.String("error", err.Error()))
		_ = uc.telegram.SendMessage(ctx, chatID, "Привяжите аккаунт в настройках EstimatePro.")
		_ = uc.telegram.AnswerCallbackQuery(ctx, cb.ID, "")
		return nil //nolint:nilerr // unlinked user is not an error
	}

	switch {
	case action.IsCancel():
		slog.InfoContext(ctx, "BotUsecase.ProcessCallback: cancel action", slog.String("chat_id", chatID))
		session, sErr := uc.sessions.GetActive(ctx, chatID)
		if sErr == nil {
			_ = uc.sessions.Cancel(ctx, session.ID)
		}
		_ = uc.telegram.SendMessage(ctx, chatID, "Отменено.")
		_ = uc.telegram.AnswerCallbackQuery(ctx, cb.ID, "Отменено")

	case action.IsConfirm():
		slog.InfoContext(ctx, "BotUsecase.ProcessCallback: confirm action", slog.String("chat_id", chatID))
		session, sErr := uc.sessions.GetActive(ctx, chatID)
		if sErr != nil {
			slog.WarnContext(ctx, "BotUsecase.ProcessCallback: confirm but no active session", slog.String("chat_id", chatID), slog.String("error", sErr.Error()))
			_ = uc.telegram.SendMessage(ctx, chatID, "Нет активной сессии.")
			_ = uc.telegram.AnswerCallbackQuery(ctx, cb.ID, "")
			return nil
		}

		if err := uc.executeSessionAction(ctx, session, link.UserID); err != nil {
			slog.ErrorContext(ctx, "BotUsecase.ProcessCallback: executeSessionAction failed", slog.String("session_id", session.ID), slog.String("error", err.Error()))
			state, _ := uc.sessions.GetState(session)
			_ = uc.telegram.SendMessage(ctx, chatID, sessionActionErrorMessage(err, state))
		} else {
			slog.InfoContext(ctx, "BotUsecase.ProcessCallback: session action completed", slog.String("session_id", session.ID))
			_ = uc.telegram.SendMessage(ctx, chatID, "Готово!") // TODO(#37): extract UI string
		}
		_ = uc.sessions.Complete(ctx, session.ID)
		_ = uc.telegram.AnswerCallbackQuery(ctx, cb.ID, "")

	case action.IsSelect():
		slog.InfoContext(ctx, "BotUsecase.ProcessCallback: selection action", slog.String("action", string(action)), slog.String("payload", payload))
		session, sErr := uc.sessions.GetActive(ctx, chatID)
		if sErr != nil {
			slog.WarnContext(ctx, "BotUsecase.ProcessCallback: selection but no active session", slog.String("chat_id", chatID), slog.String("error", sErr.Error()))
			_ = uc.telegram.AnswerCallbackQuery(ctx, cb.ID, "")
			return nil
		}

		// Handle file upload project selection.
		if session.Intent == domain.IntentUploadDocument && action == domain.SelectAction(domain.CallbackKeyProject) {
			slog.InfoContext(ctx, "BotUsecase.ProcessCallback: file upload project selected", slog.String("project_id", payload))
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

		selKey := action.SelectKey()
		if !selKey.IsKnown() {
			slog.WarnContext(ctx, "BotUsecase.ProcessCallback: select with unknown key, ignoring",
				slog.String("selection_key", string(selKey)),
				slog.String("payload", payload),
			)
			_ = uc.telegram.AnswerCallbackQuery(ctx, cb.ID, "")
			return nil
		}
		slog.DebugContext(ctx, "BotUsecase.ProcessCallback: advancing session", slog.String("selection_key", string(selKey)), slog.String("payload", payload))
		_ = uc.sessions.Advance(ctx, session, map[string]string{string(selKey): payload})

		// AddMember has no Confirm step — role-selection is the last user input.
		// Without auto-execute the flow stalls until 10-min TTL (issue #27).
		if session.Intent == domain.IntentAddMember && selKey == domain.CallbackKeyRole {
			if execErr := uc.executeSessionAction(ctx, session, link.UserID); execErr != nil {
				slog.ErrorContext(ctx, "BotUsecase.ProcessCallback: AddMember auto-execute failed", slog.String("session_id", session.ID), slog.String("error", execErr.Error()))
				state, _ := uc.sessions.GetState(session)
				_ = uc.telegram.SendMessage(ctx, chatID, sessionActionErrorMessage(execErr, state))
			} else {
				_ = uc.telegram.SendMessage(ctx, chatID, "Готово!") // TODO(#37): extract UI string
			}
			_ = uc.sessions.Complete(ctx, session.ID)
		}
		_ = uc.telegram.AnswerCallbackQuery(ctx, cb.ID, "")

	default:
		slog.DebugContext(ctx, "BotUsecase.ProcessCallback: unknown action", slog.String("action", string(action)))
		_ = uc.telegram.AnswerCallbackQuery(ctx, cb.ID, "")
	}

	return nil
}

// handleSessionMessage processes a text message within an active session flow.
func (uc *BotUsecase) handleSessionMessage(ctx context.Context, session *domain.BotSession, text string, userID string) error {
	chatID := session.ChatID
	slog.InfoContext(ctx, "BotUsecase.handleSessionMessage", slog.String("session_id", session.ID), slog.String("intent", string(session.Intent)), slog.Int("step", session.Step), slog.String("text", text))

	switch session.Intent {
	case domain.IntentCreateProject:
		return uc.handleCreateProjectSession(ctx, session, text, userID, chatID)
	case domain.IntentAddMember:
		return uc.handleAddMemberSession(ctx, session, text, userID, chatID)
	default:
		slog.DebugContext(ctx, "BotUsecase.handleSessionMessage: unhandled intent, advancing with raw text", slog.String("intent", string(session.Intent)))
		return uc.sessions.Advance(ctx, session, map[string]string{"input": text})
	}
}

func (uc *BotUsecase) handleCreateProjectSession(ctx context.Context, session *domain.BotSession, text, userID, chatID string) error {
	slog.DebugContext(ctx, "BotUsecase.handleCreateProjectSession", slog.Int("step", session.Step), slog.String("text", text))
	switch session.Step {
	case 0:
		// Step 0: we got the project name.
		if err := uc.sessions.Advance(ctx, session, map[string]string{"name": text}); err != nil {
			slog.ErrorContext(ctx, "BotUsecase.handleCreateProjectSession: Advance failed", slog.String("error", err.Error()))
			return fmt.Errorf("BotUsecase.handleCreateProjectSession: %w", err)
		}
		_ = uc.telegram.SendMessage(ctx, chatID, "Отлично! Теперь введите описание проекта (или 'пропустить').")
		return nil
	case 1:
		// Step 1: we got the description — create the project.
		state, err := uc.sessions.GetState(session)
		if err != nil {
			slog.ErrorContext(ctx, "BotUsecase.handleCreateProjectSession: GetState failed", slog.String("error", err.Error()))
			return fmt.Errorf("BotUsecase.handleCreateProjectSession: %w", err)
		}
		description := text
		if strings.EqualFold(description, "пропустить") {
			description = ""
		}

		slog.InfoContext(ctx, "BotUsecase.handleCreateProjectSession: creating project", slog.String("name", state["name"]), slog.String("user_id", userID))
		projectID, err := uc.executor.projects.Create(ctx, "", state["name"], description, userID)
		if err != nil {
			slog.ErrorContext(ctx, "BotUsecase.handleCreateProjectSession: Create failed", slog.String("name", state["name"]), slog.String("error", err.Error()))
			_ = uc.telegram.SendMessage(ctx, chatID, "Ошибка при создании проекта.")
			return fmt.Errorf("BotUsecase.handleCreateProjectSession: %w", err)
		}

		slog.InfoContext(ctx, "BotUsecase.handleCreateProjectSession: project created", slog.String("project_id", projectID), slog.String("name", state["name"]))
		_ = uc.telegram.SendMarkdown(ctx, chatID, fmt.Sprintf("Проект *%s* создан\\! ID: `%s`", state["name"], projectID))
		return uc.sessions.Complete(ctx, session.ID)
	default:
		slog.WarnContext(ctx, "BotUsecase.handleCreateProjectSession: unexpected step, completing", slog.Int("step", session.Step))
		return uc.sessions.Complete(ctx, session.ID)
	}
}

func (uc *BotUsecase) handleAddMemberSession(ctx context.Context, session *domain.BotSession, text, userID, chatID string) error {
	slog.DebugContext(ctx, "BotUsecase.handleAddMemberSession", slog.Int("step", session.Step), slog.String("text", text))
	switch session.Step {
	case 0:
		// Step 0: got project ID.
		if err := uc.sessions.Advance(ctx, session, map[string]string{"project_id": text}); err != nil {
			slog.ErrorContext(ctx, "BotUsecase.handleAddMemberSession: Advance failed", slog.String("error", err.Error()))
			return fmt.Errorf("BotUsecase.handleAddMemberSession: %w", err)
		}
		_ = uc.telegram.SendMessage(ctx, chatID, "Введите email участника.")
		return nil
	case 1:
		// Step 1: got email — add member.
		state, err := uc.sessions.GetState(session)
		if err != nil {
			slog.ErrorContext(ctx, "BotUsecase.handleAddMemberSession: GetState failed", slog.String("error", err.Error()))
			return fmt.Errorf("BotUsecase.handleAddMemberSession: %w", err)
		}

		slog.InfoContext(ctx, "BotUsecase.handleAddMemberSession: adding member", slog.String("project_id", state["project_id"]), slog.String("email", text))
		if err := uc.executor.members.AddByEmail(ctx, state["project_id"], text, "developer", userID); err != nil {
			slog.ErrorContext(ctx, "BotUsecase.handleAddMemberSession: AddByEmail failed", slog.String("project_id", state["project_id"]), slog.String("email", text), slog.String("error", err.Error()))
			_ = uc.telegram.SendMessage(ctx, chatID, "Ошибка при добавлении участника.")
			return fmt.Errorf("BotUsecase.handleAddMemberSession: %w", err)
		}

		slog.InfoContext(ctx, "BotUsecase.handleAddMemberSession: member added", slog.String("email", text), slog.String("project_id", state["project_id"]))
		_ = uc.telegram.SendMessage(ctx, chatID, "Участник добавлен!")
		return uc.sessions.Complete(ctx, session.ID)
	default:
		slog.WarnContext(ctx, "BotUsecase.handleAddMemberSession: unexpected step, completing", slog.Int("step", session.Step))
		return uc.sessions.Complete(ctx, session.ID)
	}
}

// sessionActionErrorMessage maps an executeSessionAction error to the
// user-facing message, prioritising domain sentinels (Project / Member
// not found) over a generic fallback. Mirrors the sentinel mapping in
// IntentExecutor.Execute paths so session-flow UX stays consistent with
// text-flow UX.
//
// TODO(#37): UI literals belong in bot/handler/messages — kept inline
// here while the broader extraction is tracked separately.
func sessionActionErrorMessage(err error, state map[string]string) string {
	switch {
	case errors.Is(err, domain.ErrProjectNotFound):
		return projectNotFoundMsg(state["project_name"])
	case errors.Is(err, domain.ErrMemberNotFound):
		return memberNotFoundMsg(state["user_name"])
	default:
		return "Ошибка при выполнении действия."
	}
}

// executeSessionAction runs the final action for a confirmed session.
func (uc *BotUsecase) executeSessionAction(ctx context.Context, session *domain.BotSession, userID string) error {
	slog.InfoContext(ctx, "BotUsecase.executeSessionAction", slog.String("session_id", session.ID), slog.String("intent", string(session.Intent)), slog.String("user_id", userID))
	state, err := uc.sessions.GetState(session)
	if err != nil {
		slog.ErrorContext(ctx, "BotUsecase.executeSessionAction: GetState failed", slog.String("session_id", session.ID), slog.String("error", err.Error()))
		return fmt.Errorf("BotUsecase.executeSessionAction: %w", err)
	}

	switch session.Intent {
	case domain.IntentCreateProject:
		slog.InfoContext(ctx, "BotUsecase.executeSessionAction: creating project", slog.String("name", state["name"]))
		_, err = uc.executor.projects.Create(ctx, "", state["name"], state["description"], userID)
	case domain.IntentUpdateProject:
		slog.InfoContext(ctx, "BotUsecase.executeSessionAction: updating project", slog.String("project_name", state["project_name"]))
		var p *domain.ProjectSummary
		p, err = uc.executor.findProjectByName(ctx, userID, state["project_name"])
		if err == nil {
			err = uc.executor.projects.Update(ctx, p.ID, state["new_name"], state["description"], userID)
		}
	case domain.IntentAddMember:
		slog.InfoContext(ctx, "BotUsecase.executeSessionAction: adding member", slog.String("project_name", state["project_name"]), slog.String("email", state["email"]))
		var p *domain.ProjectSummary
		p, err = uc.executor.findProjectByName(ctx, userID, state["project_name"])
		if err == nil {
			err = uc.executor.members.AddByEmail(ctx, p.ID, state["email"], state["role"], userID)
		}
	case domain.IntentRemoveMember:
		slog.InfoContext(ctx, "BotUsecase.executeSessionAction: removing member", slog.String("project_name", state["project_name"]), slog.String("user_name", state["user_name"]))
		var p *domain.ProjectSummary
		p, err = uc.executor.findProjectByName(ctx, userID, state["project_name"])
		if err == nil {
			var m *domain.MemberSummary
			m, err = uc.executor.findMemberByName(ctx, p.ID, state["user_name"])
			if err == nil {
				err = uc.executor.members.Remove(ctx, p.ID, m.UserID, userID)
			}
		}
	default:
		slog.WarnContext(ctx, "BotUsecase.executeSessionAction: unknown intent", slog.String("intent", string(session.Intent)))
		return nil
	}

	if err != nil {
		slog.ErrorContext(ctx, "BotUsecase.executeSessionAction: action failed", slog.String("intent", string(session.Intent)), slog.String("error", err.Error()))
		return fmt.Errorf("BotUsecase.executeSessionAction: %w", err)
	}
	slog.InfoContext(ctx, "BotUsecase.executeSessionAction: success", slog.String("intent", string(session.Intent)))
	return nil
}

// isBotMentioned checks if the bot is mentioned in a group message.
// tryAutoLink attempts to auto-link a Telegram user to their EstimatePro account
// by matching the Telegram user ID against users.telegram_chat_id in the database.
func (uc *BotUsecase) tryAutoLink(ctx context.Context, from *telegram.User) (*domain.BotUserLink, error) {
	slog.InfoContext(ctx, "BotUsecase.tryAutoLink: attempting auto-link",
		slog.Int64("telegram_user_id", from.ID),
		slog.String("telegram_username", from.Username),
	)

	userID, err := uc.userResolver.ResolveByTelegramID(ctx, from.ID)
	if err != nil {
		slog.DebugContext(ctx, "BotUsecase.tryAutoLink: resolver failed",
			slog.Int64("telegram_user_id", from.ID),
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("BotUsecase.tryAutoLink: %w", err)
	}

	link := &domain.BotUserLink{
		TelegramUserID:   from.ID,
		UserID:           userID,
		TelegramUsername: from.Username,
		LinkedAt:         time.Now(),
	}
	if err := uc.links.Link(ctx, link); err != nil {
		slog.ErrorContext(ctx, "BotUsecase.tryAutoLink: failed to persist link",
			slog.Int64("telegram_user_id", from.ID),
			slog.String("user_id", userID),
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("BotUsecase.tryAutoLink: %w", err)
	}

	slog.InfoContext(ctx, "BotUsecase.tryAutoLink: auto-linked user",
		slog.Int64("telegram_user_id", from.ID),
		slog.String("user_id", userID),
		slog.String("telegram_username", from.Username),
	)
	return link, nil
}

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
		slog.InfoContext(ctx, "BotUsecase.resolveLLMParser: using user config", slog.String("user_id", userID), slog.String("provider", string(cfg.Provider)), slog.String("model", cfg.Model))
		return uc.llmFactory(cfg.Provider, cfg.APIKey, cfg.Model, cfg.BaseURL)
	}

	// Try system-level config.
	cfg, err = uc.llmConfigs.GetSystem(ctx)
	if err == nil {
		slog.InfoContext(ctx, "BotUsecase.resolveLLMParser: using system config", slog.String("provider", string(cfg.Provider)), slog.String("model", cfg.Model))
		return uc.llmFactory(cfg.Provider, cfg.APIKey, cfg.Model, cfg.BaseURL)
	}

	// Fall back to environment config.
	if uc.envLLM.Provider == "" {
		slog.ErrorContext(ctx, "BotUsecase.resolveLLMParser: no LLM config found", slog.String("user_id", userID))
		return nil, domain.ErrNoLLMConfig
	}

	slog.InfoContext(ctx, "BotUsecase.resolveLLMParser: using env config", slog.String("provider", uc.envLLM.Provider), slog.String("model", uc.envLLM.Model))
	return uc.llmFactory(
		domain.LLMProviderType(uc.envLLM.Provider),
		uc.envLLM.APIKey,
		uc.envLLM.Model,
		uc.envLLM.BaseURL,
	)
}
