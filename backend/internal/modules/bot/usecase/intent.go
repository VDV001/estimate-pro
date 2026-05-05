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
	"github.com/VDV001/estimate-pro/backend/internal/modules/bot/handler/messages"
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
	extractions domain.Extractor
	reporter    domain.Reporter
}

// NewIntentExecutor creates a new IntentExecutor. extractions and
// reporter may be nil in tests / dev environments where those modules
// are not wired — the corresponding intent handlers short-circuit
// cleanly in that case.
func NewIntentExecutor(
	projects domain.ProjectManager,
	members domain.MemberManager,
	estimations domain.EstimationManager,
	documents domain.DocumentManager,
	passwords domain.PasswordResetManager,
	extractions domain.Extractor,
	reporter domain.Reporter,
) *IntentExecutor {
	return &IntentExecutor{
		projects:    projects,
		members:     members,
		estimations: estimations,
		documents:   documents,
		passwords:   passwords,
		extractions: extractions,
		reporter:    reporter,
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
	case domain.IntentRenderReport:
		return e.renderReport(ctx, intent, userID)
	case domain.IntentSubmitEstimation:
		return e.submitEstimation(ctx, intent, userID)
	case domain.IntentRequestEstimation:
		return e.requestEstimation(ctx, intent, userID)
	case domain.IntentUploadDocument:
		return e.uploadDocumentRequest(ctx, intent, userID)
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
		return messages.NoProjectsCreateFirst, nil, nil
	}

	var b strings.Builder
	b.WriteString(messages.ProjectListHeader(total))
	for i, p := range projects {
		emoji := messages.StatusEmoji(p.Status)
		b.WriteString(fmt.Sprintf("%d. %s %s", i+1, emoji, p.Name))
		if p.MemberCount > 0 {
			b.WriteString(messages.ProjectMemberSuffix(p.MemberCount))
		}
		b.WriteByte('\n')
	}

	return b.String(), nil, nil
}

func (e *IntentExecutor) getProjectStatus(ctx context.Context, intent *domain.Intent, userID string) (string, [][]domain.InlineKeyboardButton, error) {
	projectName := intent.Params["project_name"]
	slog.DebugContext(ctx, "IntentExecutor.getProjectStatus", slog.String("project_name", projectName))
	if projectName == "" {
		return messages.AskProjectForStatus, nil, nil
	}

	p, err := e.findProjectByName(ctx, userID, projectName)
	if err != nil {
		if errors.Is(err, domain.ErrProjectNotFound) {
			return messages.ProjectNotFound(projectName), nil, nil
		}
		return "", nil, err
	}
	emoji := messages.StatusEmoji(p.Status)
	return messages.ProjectStatusDetails(emoji, p.Name, p.Status, p.MemberCount), nil, nil
}

func (e *IntentExecutor) createProject(intent *domain.Intent) (string, [][]domain.InlineKeyboardButton, error) {
	name := intent.Params["name"]
	description := intent.Params["description"]

	msg := messages.ConfirmCreateProject(name, description)

	keyboard := [][]domain.InlineKeyboardButton{
		{
			{Text: messages.BtnConfirm, CallbackData: domain.ConfirmCallback(domain.IntentCreateProject)},
			{Text: messages.BtnCancel, CallbackData: domain.CancelCallback()},
		},
	}

	return msg, keyboard, nil
}

func (e *IntentExecutor) updateProject(ctx context.Context, intent *domain.Intent, userID string) (string, [][]domain.InlineKeyboardButton, error) {
	projectName := intent.Params["project_name"]
	if projectName == "" {
		return messages.AskProjectToUpdate, nil, nil
	}
	newName := intent.Params["new_name"]
	description := intent.Params["description"]
	if newName == "" && description == "" {
		return messages.AskWhatToUpdate, nil, nil
	}

	p, err := e.findProjectByName(ctx, userID, projectName)
	if err != nil {
		if errors.Is(err, domain.ErrProjectNotFound) {
			return messages.ProjectNotFound(projectName), nil, nil
		}
		return "", nil, err
	}

	keyboard := [][]domain.InlineKeyboardButton{
		{
			{Text: messages.BtnConfirm, CallbackData: domain.ConfirmCallback(domain.IntentUpdateProject)},
			{Text: messages.BtnCancel, CallbackData: domain.CancelCallback()},
		},
	}
	return messages.ConfirmUpdateProject(p.Name, newName, description), keyboard, nil
}

func (e *IntentExecutor) addMember(intent *domain.Intent) (string, [][]domain.InlineKeyboardButton, error) {
	projectName := intent.Params["project_name"]
	email := intent.Params["email"]

	if projectName == "" || email == "" {
		return messages.AskProjectAndEmail, nil, nil
	}

	msg := messages.AskMemberRole(email, projectName)

	// SelectCallback emits the select-role form that ProcessCallback's
	// strings.HasPrefix(action, CallbackPrefixSelect) branch advances the
	// active session with — without that prefix the click falls into default.
	keyboard := [][]domain.InlineKeyboardButton{
		{
			{Text: "Developer", CallbackData: domain.SelectCallback(domain.CallbackKeyRole, "developer")},
			{Text: "Tech Lead", CallbackData: domain.SelectCallback(domain.CallbackKeyRole, "tech_lead")},
		},
		{
			{Text: "PM", CallbackData: domain.SelectCallback(domain.CallbackKeyRole, "pm")},
			{Text: "Observer", CallbackData: domain.SelectCallback(domain.CallbackKeyRole, "observer")},
			{Text: "Admin", CallbackData: domain.SelectCallback(domain.CallbackKeyRole, "admin")},
		},
	}

	return msg, keyboard, nil
}

func (e *IntentExecutor) removeMember(intent *domain.Intent) (string, [][]domain.InlineKeyboardButton, error) {
	projectName := intent.Params["project_name"]
	userName := intent.Params["user_name"]

	if projectName == "" || userName == "" {
		return messages.AskProjectAndUserName, nil, nil
	}

	msg := messages.ConfirmRemoveMember(userName, projectName)

	keyboard := [][]domain.InlineKeyboardButton{
		{
			{Text: messages.BtnConfirm, CallbackData: domain.ConfirmCallback(domain.IntentRemoveMember)},
			{Text: messages.BtnCancel, CallbackData: domain.CancelCallback()},
		},
	}

	return msg, keyboard, nil
}

func (e *IntentExecutor) submitEstimation(ctx context.Context, intent *domain.Intent, userID string) (string, [][]domain.InlineKeyboardButton, error) {
	projectName := intent.Params["project_name"]
	if projectName == "" {
		return messages.AskProjectForEstimation, nil, nil
	}
	taskName := intent.Params["task_name"]
	if taskName == "" {
		return messages.AskTaskForEstimation, nil, nil
	}
	minStr := intent.Params["min_hours"]
	likelyStr := intent.Params["likely_hours"]
	maxStr := intent.Params["max_hours"]
	if minStr == "" || likelyStr == "" || maxStr == "" {
		return messages.AskHours, nil, nil
	}
	minH, likelyH, maxH, ok := parseHours(minStr, likelyStr, maxStr)
	if !ok {
		return messages.ErrHoursNotNumbers, nil, nil
	}

	p, err := e.findProjectByName(ctx, userID, projectName)
	if err != nil {
		if errors.Is(err, domain.ErrProjectNotFound) {
			return messages.ProjectNotFound(projectName), nil, nil
		}
		return "", nil, err
	}

	// Domain validates min ≤ likely ≤ max in NewEstimationItem; bot adapter
	// wraps estimation's ErrInvalidHours into domain.ErrInvalidEstimationHours
	// so we map back to UX message here without duplicating the invariant.
	if err := e.estimations.SubmitItem(ctx, p.ID, userID, taskName, minH, likelyH, maxH); err != nil {
		if errors.Is(err, domain.ErrInvalidEstimationHours) {
			return messages.ErrHoursInvariant, nil, nil
		}
		slog.ErrorContext(ctx, "IntentExecutor.submitEstimation: SubmitItem failed", slog.String("project_id", p.ID), slog.String("task", taskName), slog.String("error", err.Error()))
		return "", nil, fmt.Errorf("IntentExecutor.submitEstimation: %w", err)
	}

	return messages.EstimationSubmitted(taskName, p.Name, minH, likelyH, maxH), nil, nil
}

func (e *IntentExecutor) requestEstimation(ctx context.Context, intent *domain.Intent, userID string) (string, [][]domain.InlineKeyboardButton, error) {
	projectName := intent.Params["project_name"]
	if projectName == "" {
		return messages.AskProjectForRequestEstimation, nil, nil
	}
	taskName := intent.Params["task_name"]
	if taskName == "" {
		return messages.AskTaskForRequestEstimation, nil, nil
	}

	p, err := e.findProjectByName(ctx, userID, projectName)
	if err != nil {
		if errors.Is(err, domain.ErrProjectNotFound) {
			return messages.ProjectNotFound(projectName), nil, nil
		}
		return "", nil, err
	}

	if err := e.estimations.RequestEstimation(ctx, p.ID, userID, taskName); err != nil {
		slog.ErrorContext(ctx, "IntentExecutor.requestEstimation: RequestEstimation failed", slog.String("project_id", p.ID), slog.String("task", taskName), slog.String("error", err.Error()))
		return "", nil, fmt.Errorf("IntentExecutor.requestEstimation: %w", err)
	}

	return messages.EstimationRequested(taskName, p.Name), nil, nil
}

// uploadDocumentRequest handles the text version of the upload_document intent
// (e.g. «загрузи документ в Backend»). It resolves the project by name,
// enriches intent.Params with the resolved project_id (so that the
// subsequent NeedsSession-flow in BotUsecase persists it into session state),
// and instructs the user to send the file as the next message. The actual
// file upload is performed by handleFileUpload when the document arrives —
// it sees the active session and routes the file to the resolved project.
func (e *IntentExecutor) uploadDocumentRequest(ctx context.Context, intent *domain.Intent, userID string) (string, [][]domain.InlineKeyboardButton, error) {
	projectName := intent.Params["project_name"]
	if projectName == "" {
		return messages.AskProjectForUpload, nil, nil
	}

	p, err := e.findProjectByName(ctx, userID, projectName)
	if err != nil {
		if errors.Is(err, domain.ErrProjectNotFound) {
			return messages.ProjectNotFound(projectName), nil, nil
		}
		return "", nil, err
	}

	// Enrich intent.Params so that BotUsecase.ProcessMessage's NeedsSession
	// flow stores project_id in session state, making it available to
	// handleFileUpload when the user sends the file.
	if intent.Params == nil {
		intent.Params = make(map[string]string)
	}
	intent.Params["project_id"] = p.ID

	return messages.WaitingForFile(p.Name), nil, nil
}

func (e *IntentExecutor) listMembers(ctx context.Context, intent *domain.Intent, userID string) (string, [][]domain.InlineKeyboardButton, error) {
	projectID, err := e.resolveProjectID(ctx, intent, userID)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrProjectNotIdentified):
			return messages.AskProjectForMembers, nil, nil
		case errors.Is(err, domain.ErrProjectNotFound):
			return messages.ProjectNotFound(intent.Params["project_name"]), nil, nil
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
		return messages.NoMembersInProject, nil, nil
	}

	var b strings.Builder
	b.WriteString(messages.MembersListHeader(len(members)))
	for _, m := range members {
		b.WriteString(messages.MemberLine(m.UserName, m.Role))
	}

	return b.String(), nil, nil
}

func (e *IntentExecutor) getAggregated(ctx context.Context, intent *domain.Intent, userID string) (string, [][]domain.InlineKeyboardButton, error) {
	projectID, err := e.resolveProjectID(ctx, intent, userID)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrProjectNotIdentified):
			return messages.AskProjectForAggregated, nil, nil
		case errors.Is(err, domain.ErrProjectNotFound):
			return messages.ProjectNotFound(intent.Params["project_name"]), nil, nil
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

// renderReport hands the user a deeplink to the frontend's report
// download path. Format defaults to pdf when the LLM-classifier
// did not extract one. Reporter port is optional — when nil
// (dev/test), respond with a friendly fallback so the chat UX
// stays coherent.
func (e *IntentExecutor) renderReport(ctx context.Context, intent *domain.Intent, userID string) (string, [][]domain.InlineKeyboardButton, error) {
	projectID, err := e.resolveProjectID(ctx, intent, userID)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrProjectNotIdentified):
			return messages.AskProjectForReport, nil, nil
		case errors.Is(err, domain.ErrProjectNotFound):
			return messages.ProjectNotFound(intent.Params["project_name"]), nil, nil
		default:
			return "", nil, err
		}
	}

	if e.reporter == nil {
		return messages.ReportUnavailable, nil, nil
	}

	format := strings.ToLower(intent.Params["format"])
	if format == "" {
		format = "pdf"
	}
	url, err := e.reporter.BuildReportURL(ctx, projectID, format)
	if err != nil {
		slog.ErrorContext(ctx, "IntentExecutor.renderReport: BuildReportURL failed",
			slog.String("project_id", projectID), slog.String("format", format),
			slog.String("error", err.Error()))
		return "", nil, fmt.Errorf("IntentExecutor.Execute: %w", err)
	}
	return messages.ReportReady(url, format), nil, nil
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

// findMemberByName looks up a member of the given project by user name
// (case-insensitive). Returns domain.ErrMemberNotFound if no match.
// Symmetric with findProjectByName.
func (e *IntentExecutor) findMemberByName(ctx context.Context, projectID, name string) (*domain.MemberSummary, error) {
	members, err := e.members.List(ctx, projectID)
	if err != nil {
		slog.ErrorContext(ctx, "IntentExecutor.findMemberByName: List failed", slog.String("project_id", projectID), slog.String("error", err.Error()))
		return nil, fmt.Errorf("IntentExecutor.findMemberByName: %w", err)
	}
	for i := range members {
		if strings.EqualFold(members[i].UserName, name) {
			return &members[i], nil
		}
	}
	return nil, fmt.Errorf("%w: %s", domain.ErrMemberNotFound, name)
}

// parseHours parses min/likely/max hours from string params. Returns ok=false
// if any value fails strconv.ParseFloat. Domain validates the invariant
// min ≤ likely ≤ max in NewEstimationItem — caller should not duplicate.
func parseHours(minStr, likelyStr, maxStr string) (minH, likelyH, maxH float64, ok bool) {
	minH, errMin := strconv.ParseFloat(minStr, 64)
	likelyH, errLikely := strconv.ParseFloat(likelyStr, 64)
	maxH, errMax := strconv.ParseFloat(maxStr, 64)
	ok = errMin == nil && errLikely == nil && errMax == nil
	return
}

func (e *IntentExecutor) forgotPassword(ctx context.Context, _ *domain.Intent, userID string) (string, [][]domain.InlineKeyboardButton, error) {
	slog.DebugContext(ctx, "IntentExecutor.forgotPassword", slog.String("user_id", userID))
	if e.passwords == nil {
		return messages.PasswordResetUnavailable, nil, nil
	}
	link, err := e.passwords.RequestReset(ctx, userID)
	if err != nil {
		if errors.Is(err, domain.ErrNoPassword) {
			return messages.PasswordResetOAuth, nil, nil
		}
		return "", nil, fmt.Errorf("forgotPassword: %w", err)
	}
	return messages.PasswordResetLink(link), nil, nil
}

func (e *IntentExecutor) help() (string, [][]domain.InlineKeyboardButton, error) {
	return messages.Help, nil, nil
}

func (e *IntentExecutor) unknown() (string, [][]domain.InlineKeyboardButton, error) {
	return messages.UnknownCommand, nil, nil
}

// NeedsSession returns true if the intent type requires a multi-step session
// to be persisted between messages.
//
// IntentUploadDocument (text-flow) needs a session so that handleFileUpload
// can look up the resolved project_id when the user's file arrives in the
// following message.
func NeedsSession(intentType domain.IntentType) bool {
	switch intentType {
	case domain.IntentCreateProject,
		domain.IntentUpdateProject,
		domain.IntentAddMember,
		domain.IntentRemoveMember,
		domain.IntentUploadDocument:
		return true
	default:
		return false
	}
}
