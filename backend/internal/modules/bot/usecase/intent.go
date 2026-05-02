// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/VDV001/estimate-pro/backend/internal/modules/bot/domain"
)

// IntentExecutor executes parsed intents by delegating to the appropriate managers.
type IntentExecutor struct {
	projects    domain.ProjectManager
	members     domain.MemberManager
	estimations domain.EstimationManager
	documents   domain.DocumentManager
	passwords   domain.PasswordResetManager
}

// NewIntentExecutor creates a new IntentExecutor.
func NewIntentExecutor(
	projects domain.ProjectManager,
	members domain.MemberManager,
	estimations domain.EstimationManager,
	documents domain.DocumentManager,
	passwords domain.PasswordResetManager,
) *IntentExecutor {
	return &IntentExecutor{
		projects:    projects,
		members:     members,
		estimations: estimations,
		documents:   documents,
		passwords:   passwords,
	}
}

// Execute processes an intent and returns a response message, optional keyboard, and error.
func (e *IntentExecutor) Execute(ctx context.Context, intent *domain.Intent, userID string) (string, [][]domain.InlineKeyboardButton, error) {
	slog.InfoContext(ctx, "IntentExecutor.Execute", slog.String("intent", string(intent.Type)), slog.String("user_id", userID), slog.Any("params", intent.Params))
	switch intent.Type {
	case domain.IntentListProjects:
		return e.listProjects(ctx, userID)
	case domain.IntentGetProjectStatus:
		return e.getProjectStatus(ctx, intent, userID)
	case domain.IntentCreateProject:
		return e.createProject(intent)
	case domain.IntentAddMember:
		return e.addMember(intent)
	case domain.IntentRemoveMember:
		return e.removeMember(intent)
	case domain.IntentListMembers:
		return e.listMembers(ctx, intent, userID)
	case domain.IntentGetAggregated:
		return e.getAggregated(ctx, intent, userID)
	case domain.IntentForgotPassword:
		return e.forgotPassword(ctx, intent, userID)
	case domain.IntentHelp:
		return e.help()
	default:
		return e.unknown()
	}
}

func (e *IntentExecutor) listProjects(ctx context.Context, userID string) (string, [][]domain.InlineKeyboardButton, error) {
	slog.DebugContext(ctx, "IntentExecutor.listProjects", slog.String("user_id", userID))
	projects, total, err := e.projects.ListByUser(ctx, userID, 50, 0)
	if err != nil {
		slog.ErrorContext(ctx, "IntentExecutor.listProjects: ListByUser failed", slog.String("error", err.Error()))
		return "", nil, fmt.Errorf("IntentExecutor.Execute: %w", err)
	}
	slog.DebugContext(ctx, "IntentExecutor.listProjects: found", slog.Int("total", total))

	if total == 0 {
		return "У вас пока нет проектов. Создайте первый с помощью команды «создай проект».", nil, nil
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("📋 Ваши проекты (%d):\n\n", total))
	for i, p := range projects {
		emoji := statusEmoji(p.Status)
		b.WriteString(fmt.Sprintf("%d. %s %s", i+1, emoji, p.Name))
		if p.MemberCount > 0 {
			b.WriteString(fmt.Sprintf(" · %d уч.", p.MemberCount))
		}
		b.WriteByte('\n')
	}

	return b.String(), nil, nil
}

func (e *IntentExecutor) getProjectStatus(ctx context.Context, intent *domain.Intent, userID string) (string, [][]domain.InlineKeyboardButton, error) {
	slog.DebugContext(ctx, "IntentExecutor.getProjectStatus", slog.String("project_name", intent.Params["project_name"]))
	projectName := intent.Params["project_name"]
	if projectName == "" {
		return "Укажите название проекта, чтобы получить его статус.", nil, nil
	}

	projects, _, err := e.projects.ListByUser(ctx, userID, 100, 0)
	if err != nil {
		return "", nil, fmt.Errorf("IntentExecutor.Execute: %w", err)
	}

	for _, p := range projects {
		if strings.EqualFold(p.Name, projectName) {
			emoji := statusEmoji(p.Status)
			msg := fmt.Sprintf("%s *%s*\nСтатус: %s\nУчастников: %d",
				emoji, p.Name, p.Status, p.MemberCount)
			return msg, nil, nil
		}
	}

	return fmt.Sprintf("Проект «%s» не найден. Используйте «мои проекты» для просмотра списка.", projectName), nil, nil
}

func (e *IntentExecutor) createProject(intent *domain.Intent) (string, [][]domain.InlineKeyboardButton, error) {
	name := intent.Params["name"]
	description := intent.Params["description"]

	msg := fmt.Sprintf("Создать проект «%s»?", name)
	if description != "" {
		msg += fmt.Sprintf("\nОписание: %s", description)
	}

	keyboard := [][]domain.InlineKeyboardButton{
		{
			{Text: "Подтвердить", CallbackData: "confirm:create_project"},
			{Text: "Отмена", CallbackData: "cancel"},
		},
	}

	return msg, keyboard, nil
}

func (e *IntentExecutor) addMember(intent *domain.Intent) (string, [][]domain.InlineKeyboardButton, error) {
	projectName := intent.Params["project_name"]
	email := intent.Params["email"]

	if projectName == "" || email == "" {
		return "Укажите название проекта и email участника.", nil, nil
	}

	msg := fmt.Sprintf("Добавить %s в проект «%s». Выберите роль:", email, projectName)

	keyboard := [][]domain.InlineKeyboardButton{
		{
			{Text: "Developer", CallbackData: "role:developer"},
			{Text: "Tech Lead", CallbackData: "role:tech_lead"},
		},
		{
			{Text: "PM", CallbackData: "role:pm"},
			{Text: "Observer", CallbackData: "role:observer"},
			{Text: "Admin", CallbackData: "role:admin"},
		},
	}

	return msg, keyboard, nil
}

func (e *IntentExecutor) removeMember(intent *domain.Intent) (string, [][]domain.InlineKeyboardButton, error) {
	projectName := intent.Params["project_name"]
	userName := intent.Params["user_name"]

	if projectName == "" || userName == "" {
		return "Укажите название проекта и имя участника для удаления.", nil, nil
	}

	msg := fmt.Sprintf("Удалить %s из проекта «%s»?", userName, projectName)

	keyboard := [][]domain.InlineKeyboardButton{
		{
			{Text: "Подтвердить", CallbackData: "confirm:remove_member"},
			{Text: "Отмена", CallbackData: "cancel"},
		},
	}

	return msg, keyboard, nil
}

func (e *IntentExecutor) listMembers(ctx context.Context, intent *domain.Intent, userID string) (string, [][]domain.InlineKeyboardButton, error) {
	projectID, userMsg, err := e.resolveProjectID(ctx, intent, userID, "Укажите проект, чтобы просмотреть участников.")
	if err != nil {
		return "", nil, err
	}
	if userMsg != "" {
		return userMsg, nil, nil
	}

	slog.DebugContext(ctx, "IntentExecutor.listMembers", slog.String("project_id", projectID))
	members, err := e.members.List(ctx, projectID)
	if err != nil {
		slog.ErrorContext(ctx, "IntentExecutor.listMembers: List failed", slog.String("project_id", projectID), slog.String("error", err.Error()))
		return "", nil, fmt.Errorf("IntentExecutor.Execute: %w", err)
	}

	if len(members) == 0 {
		return "В проекте пока нет участников.", nil, nil
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("👥 Участники (%d):\n\n", len(members)))
	for _, m := range members {
		b.WriteString(fmt.Sprintf("• %s — [%s]\n", m.UserName, m.Role))
	}

	return b.String(), nil, nil
}

func (e *IntentExecutor) getAggregated(ctx context.Context, intent *domain.Intent, userID string) (string, [][]domain.InlineKeyboardButton, error) {
	projectID, userMsg, err := e.resolveProjectID(ctx, intent, userID, "Укажите проект, чтобы получить агрегированную оценку.")
	if err != nil {
		return "", nil, err
	}
	if userMsg != "" {
		return userMsg, nil, nil
	}

	slog.DebugContext(ctx, "IntentExecutor.getAggregated", slog.String("project_id", projectID))
	result, err := e.estimations.GetAggregated(ctx, projectID)
	if err != nil {
		slog.ErrorContext(ctx, "IntentExecutor.getAggregated: GetAggregated failed", slog.String("project_id", projectID), slog.String("error", err.Error()))
		return "", nil, fmt.Errorf("IntentExecutor.Execute: %w", err)
	}

	return result, nil, nil
}

// resolveProjectID resolves a project ID from intent params using two strategies:
//  1. Direct project_id (callback flow — set by selection inline-button).
//  2. Lookup by project_name through projects.ListByUser (text-message flow —
//     classifier extracts the name from a user phrase).
//
// Return contract: exactly one of (projectID, userMsg, err) is non-zero.
//   - projectID != "" — resolved, proceed.
//   - userMsg != ""   — present this message to the user with no keyboard.
//   - err != nil      — internal error, propagate.
//
// missingMsg is shown when neither project_id nor project_name is provided.
func (e *IntentExecutor) resolveProjectID(
	ctx context.Context,
	intent *domain.Intent,
	userID, missingMsg string,
) (projectID, userMsg string, err error) {
	if id := intent.Params["project_id"]; id != "" {
		return id, "", nil
	}
	name := intent.Params["project_name"]
	if name == "" {
		return "", missingMsg, nil
	}
	projects, _, listErr := e.projects.ListByUser(ctx, userID, 100, 0)
	if listErr != nil {
		slog.ErrorContext(ctx, "IntentExecutor.resolveProjectID: ListByUser failed", slog.String("user_id", userID), slog.String("error", listErr.Error()))
		return "", "", fmt.Errorf("IntentExecutor.resolveProjectID: %w", listErr)
	}
	for _, p := range projects {
		if strings.EqualFold(p.Name, name) {
			return p.ID, "", nil
		}
	}
	return "", fmt.Sprintf("Проект «%s» не найден. Используйте «мои проекты» для просмотра списка.", name), nil
}

func (e *IntentExecutor) forgotPassword(ctx context.Context, _ *domain.Intent, userID string) (string, [][]domain.InlineKeyboardButton, error) {
	slog.DebugContext(ctx, "IntentExecutor.forgotPassword", slog.String("user_id", userID))
	if e.passwords == nil {
		return "Password reset is not configured.", nil, nil
	}
	link, err := e.passwords.RequestReset(ctx, userID)
	if err != nil {
		if errors.Is(err, domain.ErrNoPassword) {
			return "Твой аккаунт использует вход через Google/GitHub. Пароль сбрасывать не нужно! 😊", nil, nil
		}
		return "", nil, fmt.Errorf("forgotPassword: %w", err)
	}
	return fmt.Sprintf("Вот ссылка для сброса пароля:\n%s\n\nДействует 15 минут ⏳", link), nil, nil
}

func (e *IntentExecutor) help() (string, [][]domain.InlineKeyboardButton, error) {
	msg := `🤖 Доступные команды:

• *мои проекты* — список ваших проектов
• *статус проекта [название]* — подробности проекта
• *создай проект [название]* — создать новый проект
• *добавь участника [email] в [проект]* — добавить участника
• *удали участника [имя] из [проект]* — удалить участника
• *участники [проект]* — список участников
• *оценка [проект]* — агрегированная оценка
• *забыл пароль* — сброс пароля
• *помощь* — эта справка

Вы также можете отправлять сообщения в свободной форме — бот постарается понять ваш запрос.`

	return msg, nil, nil
}

func (e *IntentExecutor) unknown() (string, [][]domain.InlineKeyboardButton, error) {
	return "Не удалось распознать команду. Введите «помощь» для списка доступных команд.", nil, nil
}

// NeedsSession returns true if the intent type requires a multi-step session.
func NeedsSession(intentType domain.IntentType) bool {
	switch intentType {
	case domain.IntentCreateProject,
		domain.IntentUpdateProject,
		domain.IntentAddMember,
		domain.IntentRemoveMember:
		return true
	default:
		return false
	}
}

func statusEmoji(status string) string {
	switch strings.ToLower(status) {
	case "active":
		return "✅"
	case "archived":
		return "📦"
	default:
		return "📌"
	}
}
