// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

// Package messages holds all user-facing strings (Russian) shown by the
// Telegram bot — buttons, prompts, errors, success notifications, help.
//
// Architecture role: this is presentation infrastructure. usecase imports
// it; the package itself depends only on stdlib so there is no cycle and
// no leak of usecase/domain types into UI text.
//
// Naming:
//   - Btn*       — inline keyboard button labels.
//   - Toast*     — short text shown via Telegram callback-query answer.
//   - Ask*       — prompt asking the user for input.
//   - Confirm*   — confirmation question for an inline-keyboard flow.
//   - Err*       — user-facing error message (NOT a Go error).
//   - Memory*    — strings persisted to bot conversation memory.
//   - Static const for fixed text; func for dynamic format-strings.
package messages

import (
	"fmt"
	"strings"
)

// Inline keyboard button labels (reused across confirm/cancel flows).
const (
	BtnConfirm = "Подтвердить"
	BtnCancel  = "Отмена"
)

// Callback-query answer toasts.
const (
	ToastCancelled = "Отменено"
	ToastUploading = "Загружаю..."
)

// Generic session-flow notifications.
const (
	SessionCancelled = "Отменено."
	SessionDone      = "Готово!"
	NoActiveSession  = "Нет активной сессии."
	ErrSessionAction = "Ошибка при выполнении действия."
)

// Account linkage.
const AccountNotLinked = "Привяжите аккаунт в настройках EstimatePro."

// File-upload flow.
const (
	UnsupportedFileFormat  = "Этот формат я пока не поддерживаю. Скинь PDF, DOCX, XLSX, MD, TXT или CSV!"
	FileTooLarge           = "Файл слишком большой (макс 50MB)"
	NoProjectsForUpload    = "У тебя пока нет проектов. Сначала создай проект!"
	MemoryEstiFileUploaded = "Файл загружен"
)

// FileReceivedAskProject is shown after a file arrives but no project context
// is set — asks the user to pick a project from the inline keyboard.
func FileReceivedAskProject(fileName string) string {
	return fmt.Sprintf("Файл *%s* получен! В какой проект загрузить?", fileName)
}

// FileUploadedToProject is the success notification after successful upload.
func FileUploadedToProject(fileName string) string {
	return fmt.Sprintf("Файл *%s* загружен в проект! 📎", fileName)
}

// MemoryUserUploadedFile records a file upload as a user-side memory entry.
func MemoryUserUploadedFile(fileName string) string {
	return "Загрузил файл: " + fileName
}

// Project lifecycle text — list / status / create / update / upload-target.
const (
	NoProjectsCreateFirst = "У вас пока нет проектов. Создайте первый с помощью команды «создай проект»."
	AskProjectForStatus   = "Укажите название проекта, чтобы получить его статус."
	AskProjectDescription = "Отлично! Теперь введите описание проекта (или 'пропустить')."
	SkipKeyword           = "пропустить"
	ErrCreateProject      = "Ошибка при создании проекта."
	AskProjectToUpdate    = "Укажите проект, который нужно обновить."
	AskWhatToUpdate       = "Укажите что обновить: новое имя проекта или описание."
	AskProjectForUpload   = "Укажите проект, в который нужно загрузить документ."
)

// ProjectListHeader formats the header of the «мои проекты» response.
func ProjectListHeader(total int) string {
	return fmt.Sprintf("📋 Ваши проекты (%d):\n\n", total)
}

// ProjectMemberSuffix formats the « · N уч.» suffix on each project line.
func ProjectMemberSuffix(memberCount int) string {
	return fmt.Sprintf(" · %d уч.", memberCount)
}

// ProjectStatusDetails formats the «статус проекта» response body.
func ProjectStatusDetails(emoji, name, status string, memberCount int) string {
	return fmt.Sprintf("%s *%s*\nСтатус: %s\nУчастников: %d", emoji, name, status, memberCount)
}

// ConfirmCreateProject builds the confirm-question for create_project.
// Description line is appended only when non-empty.
func ConfirmCreateProject(name, description string) string {
	msg := fmt.Sprintf("Создать проект «%s»?", name)
	if description != "" {
		msg += fmt.Sprintf("\nОписание: %s", description)
	}
	return msg
}

// ConfirmUpdateProject builds the confirm-question for update_project.
// Both new-name and description lines are conditional.
func ConfirmUpdateProject(currentName, newName, description string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Обновить проект «%s»?\n", currentName)
	if newName != "" {
		fmt.Fprintf(&b, "Новое имя: %s\n", newName)
	}
	if description != "" {
		fmt.Fprintf(&b, "Описание: %s\n", description)
	}
	return b.String()
}

// ProjectCreated is the success notification with the new project's id.
// The «\!» escape is intentional — Telegram MarkdownV2 requires it.
func ProjectCreated(name, id string) string {
	return fmt.Sprintf("Проект *%s* создан\\! ID: `%s`", name, id)
}

// ProjectNotFound is the canonical user-facing message when a project_name
// does not match any of the user's projects. Used by both text and session
// flows so the UX stays consistent across surfaces.
func ProjectNotFound(name string) string {
	return fmt.Sprintf("Проект «%s» не найден. Используйте «мои проекты» для просмотра списка.", name)
}

// WaitingForFile is shown after upload_document (text-flow) resolves a
// project, instructing the user to send the file in the next message.
func WaitingForFile(projectName string) string {
	return fmt.Sprintf("Жду файл для проекта «%s» 📎\nПоддерживаемые форматы: PDF, DOCX, XLSX, MD, TXT, CSV (до 50MB).", projectName)
}

// Member-management text — add / remove / list.
const (
	AskProjectAndEmail    = "Укажите название проекта и email участника."
	AskProjectAndUserName = "Укажите название проекта и имя участника для удаления."
	AskMemberEmail        = "Введите email участника."
	AskProjectForMembers  = "Укажите проект, чтобы просмотреть участников."
	NoMembersInProject    = "В проекте пока нет участников."
	MemberAdded           = "Участник добавлен!"
	ErrAddMember          = "Ошибка при добавлении участника."
)

// AskMemberRole prompts the user to pick a role for a new member.
func AskMemberRole(email, projectName string) string {
	return fmt.Sprintf("Добавить %s в проект «%s». Выберите роль:", email, projectName)
}

// ConfirmRemoveMember builds the confirm-question for remove_member.
func ConfirmRemoveMember(userName, projectName string) string {
	return fmt.Sprintf("Удалить %s из проекта «%s»?", userName, projectName)
}

// MembersListHeader formats the header of the «участники» response.
func MembersListHeader(count int) string {
	return fmt.Sprintf("👥 Участники (%d):\n\n", count)
}

// MemberLine formats one «• name — [role]» line in the members list.
func MemberLine(userName, role string) string {
	return fmt.Sprintf("• %s — [%s]\n", userName, role)
}

// MemberNotFound is the canonical user-facing message when a user_name does
// not match any member of a resolved project. Symmetric with ProjectNotFound.
func MemberNotFound(name string) string {
	return fmt.Sprintf("Участник «%s» не найден. Используйте «участники <проект>» для просмотра списка.", name)
}

// Estimation flow text — submit / request / aggregated.
const (
	AskProjectForEstimation        = "Укажите проект для отправки оценки."
	AskTaskForEstimation           = "Укажите задачу для оценки."
	AskHours                       = "Укажите минимальные, ожидаемые и максимальные часы (min, likely, max)."
	ErrHoursNotNumbers             = "Часы должны быть числами (например 8 или 12.5)."
	ErrHoursInvariant              = "Часы должны удовлетворять условию min ≤ likely ≤ max и быть неотрицательными."
	AskProjectForRequestEstimation = "Укажите проект, для которого нужна оценка."
	AskTaskForRequestEstimation    = "Укажите задачу, для которой нужна оценка."
	AskProjectForAggregated        = "Укажите проект, чтобы получить агрегированную оценку."
)

// EstimationSubmitted is the success notification after submit_estimation.
func EstimationSubmitted(task, project string, minH, likelyH, maxH float64) string {
	return fmt.Sprintf("Оценка для задачи «%s» в проекте «%s» отправлена! ✅\n• min: %vч\n• likely: %vч\n• max: %vч",
		task, project, minH, likelyH, maxH)
}

// EstimationRequested is the success notification after request_estimation.
func EstimationRequested(task, project string) string {
	return fmt.Sprintf("Запрос оценки задачи «%s» в проекте «%s» отправлен команде. 📨", task, project)
}

// Password-reset flow text.
const (
	PasswordResetUnavailable = "Password reset is not configured."
	PasswordResetOAuth       = "Твой аккаунт использует вход через Google/GitHub. Пароль сбрасывать не нужно! 😊"
)

// PasswordResetLink renders the reset-link response with TTL hint.
func PasswordResetLink(link string) string {
	return fmt.Sprintf("Вот ссылка для сброса пароля:\n%s\n\nДействует 15 минут ⏳", link)
}

// Help and unknown-command fallback.
const (
	Help = `🤖 Доступные команды:

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

	UnknownCommand = "Не удалось распознать команду. Введите «помощь» для списка доступных команд."
)

// StatusEmoji maps a project status string to its UI indicator. Lives in
// presentation because the emoji is a UI element, not domain data — domain
// owns Status as a string, presentation decides how to render it.
func StatusEmoji(status string) string {
	switch strings.ToLower(status) {
	case "active":
		return "✅"
	case "archived":
		return "📦"
	default:
		return "📌"
	}
}
