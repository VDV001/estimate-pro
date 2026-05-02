// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/VDV001/estimate-pro/backend/internal/modules/bot/domain"
)

// projectListLimit caps how many projects we fetch via ProjectManager.ListByUser
// when resolving a project by name. 100 covers the vast majority of users; for
// power-users with more projects, lookup-by-name should move into the
// repository as a SQL WHERE clause (separate refactor).
const projectListLimit = 100

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
	case domain.IntentUpdateProject:
		return e.updateProject(ctx, intent, userID)
	case domain.IntentAddMember:
		return e.addMember(intent)
	case domain.IntentRemoveMember:
		return e.removeMember(intent)
	case domain.IntentListMembers:
		return e.listMembers(ctx, intent, userID)
	case domain.IntentGetAggregated:
		return e.getAggregated(ctx, intent, userID)
	case domain.IntentSubmitEstimation:
		return e.submitEstimation(ctx, intent, userID)
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
	projectName := intent.Params["project_name"]
	slog.DebugContext(ctx, "IntentExecutor.getProjectStatus", slog.String("project_name", projectName))
	if projectName == "" {
		return "Укажите название проекта, чтобы получить его статус.", nil, nil
	}

	p, err := e.findProjectByName(ctx, userID, projectName)
	if err != nil {
		if errors.Is(err, domain.ErrProjectNotFound) {
			return projectNotFoundMsg(projectName), nil, nil
		}
		return "", nil, err
	}
	emoji := statusEmoji(p.Status)
	return fmt.Sprintf("%s *%s*\nСтатус: %s\nУчастников: %d", emoji, p.Name, p.Status, p.MemberCount), nil, nil
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
			{Text: "Отмена", CallbackData: "cancel:"},
		},
	}

	return msg, keyboard, nil
}

func (e *IntentExecutor) updateProject(ctx context.Context, intent *domain.Intent, userID string) (string, [][]domain.InlineKeyboardButton, error) {
	projectName := intent.Params["project_name"]
	if projectName == "" {
		return "Укажите проект, который нужно обновить.", nil, nil
	}
	newName := intent.Params["new_name"]
	description := intent.Params["description"]
	if newName == "" && description == "" {
		return "Укажите что обновить: новое имя проекта или описание.", nil, nil
	}

	p, err := e.findProjectByName(ctx, userID, projectName)
	if err != nil {
		if errors.Is(err, domain.ErrProjectNotFound) {
			return projectNotFoundMsg(projectName), nil, nil
		}
		return "", nil, err
	}

	var msg strings.Builder
	fmt.Fprintf(&msg, "Обновить проект «%s»?\n", p.Name)
	if newName != "" {
		fmt.Fprintf(&msg, "Новое имя: %s\n", newName)
	}
	if description != "" {
		fmt.Fprintf(&msg, "Описание: %s\n", description)
	}

	keyboard := [][]domain.InlineKeyboardButton{
		{
			{Text: "Подтвердить", CallbackData: "confirm:update_project"},
			{Text: "Отмена", CallbackData: "cancel:"},
		},
	}
	return msg.String(), keyboard, nil
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
			{Text: "Отмена", CallbackData: "cancel:"},
		},
	}

	return msg, keyboard, nil
}

func (e *IntentExecutor) submitEstimation(ctx context.Context, intent *domain.Intent, userID string) (string, [][]domain.InlineKeyboardButton, error) {
	projectName := intent.Params["project_name"]
	if projectName == "" {
		return "Укажите проект для отправки оценки.", nil, nil
	}
	taskName := intent.Params["task_name"]
	if taskName == "" {
		return "Укажите задачу для оценки.", nil, nil
	}
	minStr := intent.Params["min_hours"]
	likelyStr := intent.Params["likely_hours"]
	maxStr := intent.Params["max_hours"]
	if minStr == "" || likelyStr == "" || maxStr == "" {
		return "Укажите минимальные, ожидаемые и максимальные часы (min, likely, max).", nil, nil
	}
	minH, errMin := strconv.ParseFloat(minStr, 64)
	likelyH, errLikely := strconv.ParseFloat(likelyStr, 64)
	maxH, errMax := strconv.ParseFloat(maxStr, 64)
	if errMin != nil || errLikely != nil || errMax != nil {
		return "Часы должны быть числами (например 8 или 12.5).", nil, nil
	}
	// UX-friendly check that mirrors domain.NewEstimationItem invariants —
	// показываем понятное сообщение до похода в БД (домен всё равно
	// провалидирует на нижнем уровне).
	if minH < 0 || likelyH < 0 || maxH < 0 || minH > likelyH || likelyH > maxH {
		return "Часы должны удовлетворять условию min ≤ likely ≤ max и быть неотрицательными.", nil, nil
	}

	p, err := e.findProjectByName(ctx, userID, projectName)
	if err != nil {
		if errors.Is(err, domain.ErrProjectNotFound) {
			return projectNotFoundMsg(projectName), nil, nil
		}
		return "", nil, err
	}

	if err := e.estimations.SubmitItem(ctx, p.ID, userID, taskName, minH, likelyH, maxH); err != nil {
		slog.ErrorContext(ctx, "IntentExecutor.submitEstimation: SubmitItem failed", slog.String("project_id", p.ID), slog.String("task", taskName), slog.String("error", err.Error()))
		return "", nil, fmt.Errorf("IntentExecutor.submitEstimation: %w", err)
	}

	return fmt.Sprintf("Оценка для задачи «%s» в проекте «%s» отправлена! ✅\n• min: %vч\n• likely: %vч\n• max: %vч",
		taskName, p.Name, minH, likelyH, maxH), nil, nil
}

func (e *IntentExecutor) listMembers(ctx context.Context, intent *domain.Intent, userID string) (string, [][]domain.InlineKeyboardButton, error) {
	projectID, err := e.resolveProjectID(ctx, intent, userID)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrProjectNotIdentified):
			return "Укажите проект, чтобы просмотреть участников.", nil, nil
		case errors.Is(err, domain.ErrProjectNotFound):
			return projectNotFoundMsg(intent.Params["project_name"]), nil, nil
		default:
			return "", nil, err
		}
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
	projectID, err := e.resolveProjectID(ctx, intent, userID)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrProjectNotIdentified):
			return "Укажите проект, чтобы получить агрегированную оценку.", nil, nil
		case errors.Is(err, domain.ErrProjectNotFound):
			return projectNotFoundMsg(intent.Params["project_name"]), nil, nil
		default:
			return "", nil, err
		}
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
//  1. Direct project_id (callback flow — set by inline-button selection).
//  2. Lookup by project_name through findProjectByName (text-message flow —
//     classifier extracts the name from a user phrase).
//
// Returns sentinel errors so callers can map them to UI messages:
//   - domain.ErrProjectNotIdentified — neither project_id nor project_name set.
//   - domain.ErrProjectNotFound       — name does not match any user's project.
//   - other errors are internal and should propagate.
func (e *IntentExecutor) resolveProjectID(ctx context.Context, intent *domain.Intent, userID string) (string, error) {
	if id := intent.Params["project_id"]; id != "" {
		return id, nil
	}
	name := intent.Params["project_name"]
	if name == "" {
		return "", domain.ErrProjectNotIdentified
	}
	p, err := e.findProjectByName(ctx, userID, name)
	if err != nil {
		return "", err
	}
	return p.ID, nil
}

// findProjectByName looks up a project by name (case-insensitive) among the
// user's projects. Returns domain.ErrProjectNotFound if no match.
func (e *IntentExecutor) findProjectByName(ctx context.Context, userID, name string) (*domain.ProjectSummary, error) {
	projects, _, err := e.projects.ListByUser(ctx, userID, projectListLimit, 0)
	if err != nil {
		slog.ErrorContext(ctx, "IntentExecutor.findProjectByName: ListByUser failed", slog.String("user_id", userID), slog.String("error", err.Error()))
		return nil, fmt.Errorf("IntentExecutor.findProjectByName: %w", err)
	}
	for i := range projects {
		if strings.EqualFold(projects[i].Name, name) {
			return &projects[i], nil
		}
	}
	return nil, fmt.Errorf("%w: %s", domain.ErrProjectNotFound, name)
}

// projectNotFoundMsg returns the canonical user-facing message shown when a
// project_name does not match any of the user's projects.
func projectNotFoundMsg(name string) string {
	return fmt.Sprintf("Проект «%s» не найден. Используйте «мои проекты» для просмотра списка.", name)
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
