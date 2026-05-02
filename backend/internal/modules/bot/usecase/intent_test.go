// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package usecase_test

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/VDV001/estimate-pro/backend/internal/modules/bot/domain"
	"github.com/VDV001/estimate-pro/backend/internal/modules/bot/usecase"
)

// --- Mocks ---

type mockProjectManager struct {
	createFn func(ctx context.Context, workspaceID, name, description, userID string) (string, error)
	updateFn func(ctx context.Context, projectID, name, description, userID string) error
	listFn   func(ctx context.Context, userID string, limit, offset int) ([]domain.ProjectSummary, int, error)
}

func (m *mockProjectManager) Create(ctx context.Context, workspaceID, name, description, userID string) (string, error) {
	if m.createFn != nil {
		return m.createFn(ctx, workspaceID, name, description, userID)
	}
	return "", nil
}

func (m *mockProjectManager) Update(ctx context.Context, projectID, name, description, userID string) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, projectID, name, description, userID)
	}
	return nil
}

func (m *mockProjectManager) ListByUser(ctx context.Context, userID string, limit, offset int) ([]domain.ProjectSummary, int, error) {
	if m.listFn != nil {
		return m.listFn(ctx, userID, limit, offset)
	}
	return nil, 0, nil
}

type mockMemberManager struct {
	addByEmailFn func(ctx context.Context, projectID, email, role, callerID string) error
	removeFn     func(ctx context.Context, projectID, userID, callerID string) error
	listFn       func(ctx context.Context, projectID string) ([]domain.MemberSummary, error)
}

func (m *mockMemberManager) AddByEmail(ctx context.Context, projectID, email, role, callerID string) error {
	if m.addByEmailFn != nil {
		return m.addByEmailFn(ctx, projectID, email, role, callerID)
	}
	return nil
}

func (m *mockMemberManager) Remove(ctx context.Context, projectID, userID, callerID string) error {
	if m.removeFn != nil {
		return m.removeFn(ctx, projectID, userID, callerID)
	}
	return nil
}

func (m *mockMemberManager) List(ctx context.Context, projectID string) ([]domain.MemberSummary, error) {
	if m.listFn != nil {
		return m.listFn(ctx, projectID)
	}
	return nil, nil
}

type mockEstimationManager struct {
	getAggregatedFn     func(ctx context.Context, projectID string) (string, error)
	submitItemFn        func(ctx context.Context, projectID, userID, taskName string, minHours, likelyHours, maxHours float64) error
	requestEstimationFn func(ctx context.Context, projectID, userID, taskName string) error
}

func (m *mockEstimationManager) GetAggregated(ctx context.Context, projectID string) (string, error) {
	if m.getAggregatedFn != nil {
		return m.getAggregatedFn(ctx, projectID)
	}
	return "", nil
}

func (m *mockEstimationManager) SubmitItem(ctx context.Context, projectID, userID, taskName string, minHours, likelyHours, maxHours float64) error {
	if m.submitItemFn != nil {
		return m.submitItemFn(ctx, projectID, userID, taskName, minHours, likelyHours, maxHours)
	}
	return nil
}

func (m *mockEstimationManager) RequestEstimation(ctx context.Context, projectID, userID, taskName string) error {
	if m.requestEstimationFn != nil {
		return m.requestEstimationFn(ctx, projectID, userID, taskName)
	}
	return nil
}

type mockDocumentManager struct {
	uploadFn func(ctx context.Context, projectID, title, fileName string, fileSize int64, fileType string, content io.Reader, userID string) error
}

func (m *mockDocumentManager) Upload(ctx context.Context, projectID, title, fileName string, fileSize int64, fileType string, content io.Reader, userID string) error {
	if m.uploadFn != nil {
		return m.uploadFn(ctx, projectID, title, fileName, fileSize, fileType, content, userID)
	}
	return nil
}

type mockPasswordResetManager struct {
	requestResetFn func(ctx context.Context, userID string) (string, error)
}

func (m *mockPasswordResetManager) RequestReset(ctx context.Context, userID string) (string, error) {
	if m.requestResetFn != nil {
		return m.requestResetFn(ctx, userID)
	}
	return "", nil
}

// --- Tests ---

func TestExecute(t *testing.T) {
	tests := []struct {
		name              string
		intent            *domain.Intent
		userID            string
		projects          *mockProjectManager
		members           *mockMemberManager
		estimations       *mockEstimationManager
		passwords         *mockPasswordResetManager
		wantContains      []string
		wantNotContains   []string
		wantKeyboard      bool
		wantKeyboardTexts []string
		wantErr           bool
	}{
		{
			name:   "ListProjects_Success",
			intent: &domain.Intent{Type: domain.IntentListProjects},
			userID: "user-1",
			projects: &mockProjectManager{
				listFn: func(_ context.Context, _ string, _, _ int) ([]domain.ProjectSummary, int, error) {
					return []domain.ProjectSummary{
						{ID: "p1", Name: "Alpha", Status: "active", MemberCount: 3},
						{ID: "p2", Name: "Beta", Status: "archived", MemberCount: 1},
					}, 2, nil
				},
			},
			wantContains: []string{"Alpha", "Beta", "✅", "📦", "2"},
		},
		{
			name:   "ListProjects_Empty",
			intent: &domain.Intent{Type: domain.IntentListProjects},
			userID: "user-1",
			projects: &mockProjectManager{
				listFn: func(_ context.Context, _ string, _, _ int) ([]domain.ProjectSummary, int, error) {
					return nil, 0, nil
				},
			},
			wantContains: []string{"нет проектов"},
		},
		{
			name:   "ListProjects_Error",
			intent: &domain.Intent{Type: domain.IntentListProjects},
			userID: "user-1",
			projects: &mockProjectManager{
				listFn: func(_ context.Context, _ string, _, _ int) ([]domain.ProjectSummary, int, error) {
					return nil, 0, errors.New("db error")
				},
			},
			wantErr: true,
		},
		{
			name:   "Help",
			intent: &domain.Intent{Type: domain.IntentHelp},
			userID: "user-1",
			wantContains: []string{
				"мои проекты",
				"статус проекта",
				"создай проект",
				"помощь",
			},
		},
		{
			name:         "Unknown",
			intent:       &domain.Intent{Type: domain.IntentUnknown},
			userID:       "user-1",
			wantContains: []string{"Не удалось распознать", "помощь"},
		},
		{
			name:   "GetAggregated_Success",
			intent: &domain.Intent{Type: domain.IntentGetAggregated, Params: map[string]string{"project_id": "p1"}},
			userID: "user-1",
			estimations: &mockEstimationManager{
				getAggregatedFn: func(_ context.Context, _ string) (string, error) {
					return "Общая оценка: 120ч (min: 80ч, max: 160ч)", nil
				},
			},
			wantContains: []string{"120ч", "80ч", "160ч"},
		},
		{
			name:   "GetAggregated_NoIdentifier",
			intent: &domain.Intent{Type: domain.IntentGetAggregated, Params: map[string]string{}},
			userID: "user-1",
			wantContains: []string{"Укажите проект"},
		},
		{
			name:   "GetAggregated_ByName_Success",
			intent: &domain.Intent{Type: domain.IntentGetAggregated, Params: map[string]string{"project_name": "Alpha"}},
			userID: "user-1",
			projects: &mockProjectManager{
				listFn: func(_ context.Context, _ string, _, _ int) ([]domain.ProjectSummary, int, error) {
					return []domain.ProjectSummary{
						{ID: "p1", Name: "Alpha", Status: "active"},
					}, 1, nil
				},
			},
			estimations: &mockEstimationManager{
				getAggregatedFn: func(_ context.Context, projectID string) (string, error) {
					if projectID != "p1" {
						return "", errors.New("wrong project_id passed: " + projectID)
					}
					return "Общая оценка: 120ч", nil
				},
			},
			wantContains: []string{"120ч"},
		},
		{
			name:   "GetAggregated_ByName_NotFound",
			intent: &domain.Intent{Type: domain.IntentGetAggregated, Params: map[string]string{"project_name": "Ghost"}},
			userID: "user-1",
			projects: &mockProjectManager{
				listFn: func(_ context.Context, _ string, _, _ int) ([]domain.ProjectSummary, int, error) {
					return []domain.ProjectSummary{}, 0, nil
				},
			},
			wantContains: []string{"Ghost", "не найден"},
		},
		{
			name:   "GetAggregated_ByName_ListProjectsError",
			intent: &domain.Intent{Type: domain.IntentGetAggregated, Params: map[string]string{"project_name": "Alpha"}},
			userID: "user-1",
			projects: &mockProjectManager{
				listFn: func(_ context.Context, _ string, _, _ int) ([]domain.ProjectSummary, int, error) {
					return nil, 0, errors.New("db error")
				},
			},
			wantErr: true,
		},
		{
			name:         "UpdateProject_NoName",
			intent:       &domain.Intent{Type: domain.IntentUpdateProject, Params: map[string]string{}},
			userID:       "user-1",
			wantContains: []string{"Укажите проект"},
		},
		{
			name:         "UpdateProject_NoChanges",
			intent:       &domain.Intent{Type: domain.IntentUpdateProject, Params: map[string]string{"project_name": "Alpha"}},
			userID:       "user-1",
			wantContains: []string{"обновить"},
		},
		{
			name:   "UpdateProject_ByName_Success",
			intent: &domain.Intent{Type: domain.IntentUpdateProject, Params: map[string]string{"project_name": "Alpha", "description": "new desc"}},
			userID: "user-1",
			projects: &mockProjectManager{
				listFn: func(_ context.Context, _ string, _, _ int) ([]domain.ProjectSummary, int, error) {
					return []domain.ProjectSummary{{ID: "p1", Name: "Alpha", Status: "active"}}, 1, nil
				},
			},
			wantContains:      []string{"Alpha", "new desc"},
			wantKeyboard:      true,
			wantKeyboardTexts: []string{"Подтвердить", "Отмена"},
		},
		{
			name:   "UpdateProject_ByName_RenameAndDescription",
			intent: &domain.Intent{Type: domain.IntentUpdateProject, Params: map[string]string{"project_name": "Alpha", "new_name": "Alpha-2", "description": "v2"}},
			userID: "user-1",
			projects: &mockProjectManager{
				listFn: func(_ context.Context, _ string, _, _ int) ([]domain.ProjectSummary, int, error) {
					return []domain.ProjectSummary{{ID: "p1", Name: "Alpha", Status: "active"}}, 1, nil
				},
			},
			wantContains:      []string{"Alpha-2", "v2"},
			wantKeyboard:      true,
			wantKeyboardTexts: []string{"Подтвердить", "Отмена"},
		},
		{
			name:   "UpdateProject_ByName_NotFound",
			intent: &domain.Intent{Type: domain.IntentUpdateProject, Params: map[string]string{"project_name": "Ghost", "description": "new"}},
			userID: "user-1",
			projects: &mockProjectManager{
				listFn: func(_ context.Context, _ string, _, _ int) ([]domain.ProjectSummary, int, error) {
					return []domain.ProjectSummary{}, 0, nil
				},
			},
			wantContains: []string{"Ghost", "не найден"},
		},
		{
			name:   "UpdateProject_ListProjectsError",
			intent: &domain.Intent{Type: domain.IntentUpdateProject, Params: map[string]string{"project_name": "Alpha", "description": "new"}},
			userID: "user-1",
			projects: &mockProjectManager{
				listFn: func(_ context.Context, _ string, _, _ int) ([]domain.ProjectSummary, int, error) {
					return nil, 0, errors.New("db error")
				},
			},
			wantErr: true,
		},
		{
			name: "SubmitEstimation_Success",
			intent: &domain.Intent{Type: domain.IntentSubmitEstimation, Params: map[string]string{
				"project_name": "Alpha", "task_name": "Login",
				"min_hours": "8", "likely_hours": "12", "max_hours": "20",
			}},
			userID: "user-1",
			projects: &mockProjectManager{
				listFn: func(_ context.Context, _ string, _, _ int) ([]domain.ProjectSummary, int, error) {
					return []domain.ProjectSummary{{ID: "p1", Name: "Alpha"}}, 1, nil
				},
			},
			estimations: &mockEstimationManager{
				submitItemFn: func(_ context.Context, projectID, uid, taskName string, minH, likelyH, maxH float64) error {
					if projectID != "p1" || uid != "user-1" || taskName != "Login" {
						return errors.New("wrong args: " + projectID + "," + uid + "," + taskName)
					}
					if minH != 8 || likelyH != 12 || maxH != 20 {
						return errors.New("wrong hours")
					}
					return nil
				},
			},
			wantContains: []string{"Login", "Alpha", "отправлена"},
		},
		{
			name: "SubmitEstimation_NoProjectName",
			intent: &domain.Intent{Type: domain.IntentSubmitEstimation, Params: map[string]string{
				"task_name": "X", "min_hours": "1", "likely_hours": "2", "max_hours": "3",
			}},
			userID:       "user-1",
			wantContains: []string{"Укажите", "проект"},
		},
		{
			name: "SubmitEstimation_NoTaskName",
			intent: &domain.Intent{Type: domain.IntentSubmitEstimation, Params: map[string]string{
				"project_name": "Alpha", "min_hours": "1", "likely_hours": "2", "max_hours": "3",
			}},
			userID:       "user-1",
			wantContains: []string{"задач"},
		},
		{
			name: "SubmitEstimation_MissingHours",
			intent: &domain.Intent{Type: domain.IntentSubmitEstimation, Params: map[string]string{
				"project_name": "Alpha", "task_name": "X", "min_hours": "5",
			}},
			userID:       "user-1",
			wantContains: []string{"час"},
		},
		{
			name: "SubmitEstimation_NonNumericHours",
			intent: &domain.Intent{Type: domain.IntentSubmitEstimation, Params: map[string]string{
				"project_name": "Alpha", "task_name": "X",
				"min_hours": "abc", "likely_hours": "12", "max_hours": "20",
			}},
			userID:       "user-1",
			wantContains: []string{"числ"},
		},
		{
			// Verifies sentinel-error-mapping path specifically: params are
			// syntactically valid (parseHours OK), executor reaches SubmitItem,
			// mock returns ErrInvalidEstimationHours, executor maps to the
			// invariant-message text. The "удовлетворять условию" substring
			// is distinct from parseHours-failure ("числами") and from
			// project-not-found / domain-not-identified texts, ensuring this
			// test pins exactly the sentinel-mapping branch.
			name: "SubmitEstimation_DomainInvalidHours",
			intent: &domain.Intent{Type: domain.IntentSubmitEstimation, Params: map[string]string{
				"project_name": "Alpha", "task_name": "X",
				"min_hours": "1", "likely_hours": "2", "max_hours": "3",
			}},
			userID: "user-1",
			projects: &mockProjectManager{
				listFn: func(_ context.Context, _ string, _, _ int) ([]domain.ProjectSummary, int, error) {
					return []domain.ProjectSummary{{ID: "p1", Name: "Alpha"}}, 1, nil
				},
			},
			estimations: &mockEstimationManager{
				submitItemFn: func(_ context.Context, _, _, _ string, _, _, _ float64) error {
					return domain.ErrInvalidEstimationHours
				},
			},
			wantContains: []string{"удовлетворять"},
		},
		{
			name: "RequestEstimation_FeatureNotImplemented",
			intent: &domain.Intent{Type: domain.IntentRequestEstimation, Params: map[string]string{
				"project_name": "Alpha", "task_name": "Auth",
			}},
			userID: "user-1",
			projects: &mockProjectManager{
				listFn: func(_ context.Context, _ string, _, _ int) ([]domain.ProjectSummary, int, error) {
					return []domain.ProjectSummary{{ID: "p1", Name: "Alpha"}}, 1, nil
				},
			},
			estimations: &mockEstimationManager{
				requestEstimationFn: func(_ context.Context, _, _, _ string) error {
					return domain.ErrFeatureNotImplemented
				},
			},
			wantContains: []string{"разработк"},
		},
		{
			name: "SubmitEstimation_ProjectNotFound",
			intent: &domain.Intent{Type: domain.IntentSubmitEstimation, Params: map[string]string{
				"project_name": "Ghost", "task_name": "X",
				"min_hours": "1", "likely_hours": "2", "max_hours": "3",
			}},
			userID: "user-1",
			projects: &mockProjectManager{
				listFn: func(_ context.Context, _ string, _, _ int) ([]domain.ProjectSummary, int, error) {
					return []domain.ProjectSummary{}, 0, nil
				},
			},
			wantContains: []string{"Ghost", "не найден"},
		},
		{
			name: "SubmitEstimation_SubmitError",
			intent: &domain.Intent{Type: domain.IntentSubmitEstimation, Params: map[string]string{
				"project_name": "Alpha", "task_name": "X",
				"min_hours": "1", "likely_hours": "2", "max_hours": "3",
			}},
			userID: "user-1",
			projects: &mockProjectManager{
				listFn: func(_ context.Context, _ string, _, _ int) ([]domain.ProjectSummary, int, error) {
					return []domain.ProjectSummary{{ID: "p1", Name: "Alpha"}}, 1, nil
				},
			},
			estimations: &mockEstimationManager{
				submitItemFn: func(_ context.Context, _, _, _ string, _, _, _ float64) error {
					return errors.New("submit failed")
				},
			},
			wantErr: true,
		},
		{
			name: "RequestEstimation_Success",
			intent: &domain.Intent{Type: domain.IntentRequestEstimation, Params: map[string]string{
				"project_name": "Alpha", "task_name": "Auth",
			}},
			userID: "user-1",
			projects: &mockProjectManager{
				listFn: func(_ context.Context, _ string, _, _ int) ([]domain.ProjectSummary, int, error) {
					return []domain.ProjectSummary{{ID: "p1", Name: "Alpha"}}, 1, nil
				},
			},
			estimations: &mockEstimationManager{
				requestEstimationFn: func(_ context.Context, projectID, uid, taskName string) error {
					if projectID != "p1" || uid != "user-1" || taskName != "Auth" {
						return errors.New("wrong args: " + projectID + "," + uid + "," + taskName)
					}
					return nil
				},
			},
			wantContains: []string{"Auth", "Alpha", "Запрос"},
		},
		{
			name:         "RequestEstimation_NoProjectName",
			intent:       &domain.Intent{Type: domain.IntentRequestEstimation, Params: map[string]string{"task_name": "Auth"}},
			userID:       "user-1",
			wantContains: []string{"Укажите", "проект"},
		},
		{
			name:         "RequestEstimation_NoTaskName",
			intent:       &domain.Intent{Type: domain.IntentRequestEstimation, Params: map[string]string{"project_name": "Alpha"}},
			userID:       "user-1",
			wantContains: []string{"задач"},
		},
		{
			name: "RequestEstimation_ProjectNotFound",
			intent: &domain.Intent{Type: domain.IntentRequestEstimation, Params: map[string]string{
				"project_name": "Ghost", "task_name": "Auth",
			}},
			userID: "user-1",
			projects: &mockProjectManager{
				listFn: func(_ context.Context, _ string, _, _ int) ([]domain.ProjectSummary, int, error) {
					return []domain.ProjectSummary{}, 0, nil
				},
			},
			wantContains: []string{"Ghost", "не найден"},
		},
		{
			name:         "UploadDocument_NoProjectName",
			intent:       &domain.Intent{Type: domain.IntentUploadDocument, Params: map[string]string{}},
			userID:       "user-1",
			wantContains: []string{"Укажите", "проект"},
		},
		{
			name:   "UploadDocument_ByName_Success",
			intent: &domain.Intent{Type: domain.IntentUploadDocument, Params: map[string]string{"project_name": "Alpha"}},
			userID: "user-1",
			projects: &mockProjectManager{
				listFn: func(_ context.Context, _ string, _, _ int) ([]domain.ProjectSummary, int, error) {
					return []domain.ProjectSummary{{ID: "p1", Name: "Alpha"}}, 1, nil
				},
			},
			wantContains: []string{"Alpha", "файл"},
		},
		{
			name:   "UploadDocument_ByName_NotFound",
			intent: &domain.Intent{Type: domain.IntentUploadDocument, Params: map[string]string{"project_name": "Ghost"}},
			userID: "user-1",
			projects: &mockProjectManager{
				listFn: func(_ context.Context, _ string, _, _ int) ([]domain.ProjectSummary, int, error) {
					return []domain.ProjectSummary{}, 0, nil
				},
			},
			wantContains: []string{"Ghost", "не найден"},
		},
		{
			name:   "UploadDocument_ListProjectsError",
			intent: &domain.Intent{Type: domain.IntentUploadDocument, Params: map[string]string{"project_name": "Alpha"}},
			userID: "user-1",
			projects: &mockProjectManager{
				listFn: func(_ context.Context, _ string, _, _ int) ([]domain.ProjectSummary, int, error) {
					return nil, 0, errors.New("db error")
				},
			},
			wantErr: true,
		},
		{
			name: "RequestEstimation_RequestError",
			intent: &domain.Intent{Type: domain.IntentRequestEstimation, Params: map[string]string{
				"project_name": "Alpha", "task_name": "Auth",
			}},
			userID: "user-1",
			projects: &mockProjectManager{
				listFn: func(_ context.Context, _ string, _, _ int) ([]domain.ProjectSummary, int, error) {
					return []domain.ProjectSummary{{ID: "p1", Name: "Alpha"}}, 1, nil
				},
			},
			estimations: &mockEstimationManager{
				requestEstimationFn: func(_ context.Context, _, _, _ string) error {
					return errors.New("notify dispatch failed")
				},
			},
			wantErr: true,
		},
		{
			name:   "GetAggregated_Error",
			intent: &domain.Intent{Type: domain.IntentGetAggregated, Params: map[string]string{"project_id": "p1"}},
			userID: "user-1",
			estimations: &mockEstimationManager{
				getAggregatedFn: func(_ context.Context, _ string) (string, error) {
					return "", errors.New("estimation error")
				},
			},
			wantErr: true,
		},
		{
			name: "CreateProject_ReturnsConfirmation",
			intent: &domain.Intent{
				Type:   domain.IntentCreateProject,
				Params: map[string]string{"name": "NewProject", "description": "A cool project"},
			},
			userID:            "user-1",
			wantContains:      []string{"NewProject", "A cool project"},
			wantKeyboard:      true,
			wantKeyboardTexts: []string{"Подтвердить", "Отмена"},
		},
		{
			name: "CreateProject_NoDescription",
			intent: &domain.Intent{
				Type:   domain.IntentCreateProject,
				Params: map[string]string{"name": "SimpleProject"},
			},
			userID:            "user-1",
			wantContains:      []string{"SimpleProject"},
			wantNotContains:   []string{"Описание"},
			wantKeyboard:      true,
			wantKeyboardTexts: []string{"Подтвердить", "Отмена"},
		},
		{
			name: "AddMember_ReturnsRoleKeyboard",
			intent: &domain.Intent{
				Type:   domain.IntentAddMember,
				Params: map[string]string{"project_name": "Alpha", "email": "dev@example.com"},
			},
			userID:            "user-1",
			wantContains:      []string{"dev@example.com", "Alpha"},
			wantKeyboard:      true,
			wantKeyboardTexts: []string{"Developer", "Tech Lead", "PM", "Observer", "Admin"},
		},
		{
			name: "AddMember_MissingParams",
			intent: &domain.Intent{
				Type:   domain.IntentAddMember,
				Params: map[string]string{"project_name": "Alpha"},
			},
			userID:       "user-1",
			wantContains: []string{"Укажите"},
		},
		{
			name: "RemoveMember_Success",
			intent: &domain.Intent{
				Type:   domain.IntentRemoveMember,
				Params: map[string]string{"project_name": "Alpha", "user_name": "John"},
			},
			userID:            "user-1",
			wantContains:      []string{"Удалить", "John", "Alpha"},
			wantKeyboard:      true,
			wantKeyboardTexts: []string{"Подтвердить", "Отмена"},
		},
		{
			name: "RemoveMember_MissingParams",
			intent: &domain.Intent{
				Type:   domain.IntentRemoveMember,
				Params: map[string]string{},
			},
			userID:       "user-1",
			wantContains: []string{"Укажите"},
		},
		{
			name: "ListMembers_Success",
			intent: &domain.Intent{
				Type:   domain.IntentListMembers,
				Params: map[string]string{"project_id": "p1"},
			},
			userID: "user-1",
			members: &mockMemberManager{
				listFn: func(_ context.Context, _ string) ([]domain.MemberSummary, error) {
					return []domain.MemberSummary{
						{UserID: "u1", UserName: "Alice", Role: "admin"},
						{UserID: "u2", UserName: "Bob", Role: "developer"},
					}, nil
				},
			},
			wantContains: []string{"Alice", "Bob", "admin", "developer", "👥", "2"},
		},
		{
			name: "ListMembers_Empty",
			intent: &domain.Intent{
				Type:   domain.IntentListMembers,
				Params: map[string]string{"project_id": "p1"},
			},
			userID: "user-1",
			members: &mockMemberManager{
				listFn: func(_ context.Context, _ string) ([]domain.MemberSummary, error) {
					return nil, nil
				},
			},
			wantContains: []string{"нет участников"},
		},
		{
			name: "ListMembers_NoIdentifier",
			intent: &domain.Intent{
				Type:   domain.IntentListMembers,
				Params: map[string]string{},
			},
			userID:       "user-1",
			wantContains: []string{"Укажите проект"},
		},
		{
			name: "ListMembers_ByName_Success",
			intent: &domain.Intent{
				Type:   domain.IntentListMembers,
				Params: map[string]string{"project_name": "Alpha"},
			},
			userID: "user-1",
			projects: &mockProjectManager{
				listFn: func(_ context.Context, _ string, _, _ int) ([]domain.ProjectSummary, int, error) {
					return []domain.ProjectSummary{
						{ID: "p1", Name: "Alpha", Status: "active"},
					}, 1, nil
				},
			},
			members: &mockMemberManager{
				listFn: func(_ context.Context, projectID string) ([]domain.MemberSummary, error) {
					if projectID != "p1" {
						return nil, errors.New("wrong project_id passed: " + projectID)
					}
					return []domain.MemberSummary{
						{UserID: "u1", UserName: "Alice", Role: "admin"},
					}, nil
				},
			},
			wantContains: []string{"Alice", "admin", "👥"},
		},
		{
			name: "ListMembers_ByName_NotFound",
			intent: &domain.Intent{
				Type:   domain.IntentListMembers,
				Params: map[string]string{"project_name": "Ghost"},
			},
			userID: "user-1",
			projects: &mockProjectManager{
				listFn: func(_ context.Context, _ string, _, _ int) ([]domain.ProjectSummary, int, error) {
					return []domain.ProjectSummary{
						{ID: "p1", Name: "Alpha", Status: "active"},
					}, 1, nil
				},
			},
			wantContains: []string{"Ghost", "не найден"},
		},
		{
			name: "ListMembers_ByName_ListProjectsError",
			intent: &domain.Intent{
				Type:   domain.IntentListMembers,
				Params: map[string]string{"project_name": "Alpha"},
			},
			userID: "user-1",
			projects: &mockProjectManager{
				listFn: func(_ context.Context, _ string, _, _ int) ([]domain.ProjectSummary, int, error) {
					return nil, 0, errors.New("db error")
				},
			},
			wantErr: true,
		},
		{
			name: "ListMembers_Error",
			intent: &domain.Intent{
				Type:   domain.IntentListMembers,
				Params: map[string]string{"project_id": "p1"},
			},
			userID: "user-1",
			members: &mockMemberManager{
				listFn: func(_ context.Context, _ string) ([]domain.MemberSummary, error) {
					return nil, errors.New("db error")
				},
			},
			wantErr: true,
		},
		{
			name: "GetProjectStatus_Success",
			intent: &domain.Intent{
				Type:   domain.IntentGetProjectStatus,
				Params: map[string]string{"project_name": "Alpha"},
			},
			userID: "user-1",
			projects: &mockProjectManager{
				listFn: func(_ context.Context, _ string, _, _ int) ([]domain.ProjectSummary, int, error) {
					return []domain.ProjectSummary{
						{ID: "p1", Name: "Alpha", Status: "active", MemberCount: 5},
					}, 1, nil
				},
			},
			wantContains: []string{"Alpha", "active", "5"},
		},
		{
			name: "GetProjectStatus_NotFound",
			intent: &domain.Intent{
				Type:   domain.IntentGetProjectStatus,
				Params: map[string]string{"project_name": "NonExistent"},
			},
			userID: "user-1",
			projects: &mockProjectManager{
				listFn: func(_ context.Context, _ string, _, _ int) ([]domain.ProjectSummary, int, error) {
					return []domain.ProjectSummary{
						{ID: "p1", Name: "Alpha", Status: "active"},
					}, 1, nil
				},
			},
			wantContains: []string{"не найден"},
		},
		{
			name: "GetProjectStatus_NoName",
			intent: &domain.Intent{
				Type:   domain.IntentGetProjectStatus,
				Params: map[string]string{},
			},
			userID:       "user-1",
			wantContains: []string{"Укажите"},
		},
		{
			name: "GetProjectStatus_ListError",
			intent: &domain.Intent{
				Type:   domain.IntentGetProjectStatus,
				Params: map[string]string{"project_name": "Alpha"},
			},
			userID: "user-1",
			projects: &mockProjectManager{
				listFn: func(_ context.Context, _ string, _, _ int) ([]domain.ProjectSummary, int, error) {
					return nil, 0, errors.New("db error")
				},
			},
			wantErr: true,
		},
		{
			name: "ForgotPassword_Success",
			intent: &domain.Intent{
				Type:   domain.IntentForgotPassword,
				Params: map[string]string{},
			},
			userID: "user-1",
			passwords: &mockPasswordResetManager{
				requestResetFn: func(_ context.Context, _ string) (string, error) {
					return "https://example.com/reset/abc123", nil
				},
			},
			wantContains: []string{"https://example.com/reset/abc123", "15 минут"},
		},
		{
			name: "ForgotPassword_OAuthUser",
			intent: &domain.Intent{
				Type:   domain.IntentForgotPassword,
				Params: map[string]string{},
			},
			userID: "user-1",
			passwords: &mockPasswordResetManager{
				requestResetFn: func(_ context.Context, _ string) (string, error) {
					return "", domain.ErrNoPassword
				},
			},
			wantContains: []string{"Google/GitHub"},
		},
		{
			name: "ForgotPassword_Error",
			intent: &domain.Intent{
				Type:   domain.IntentForgotPassword,
				Params: map[string]string{},
			},
			userID: "user-1",
			passwords: &mockPasswordResetManager{
				requestResetFn: func(_ context.Context, _ string) (string, error) {
					return "", errors.New("internal error")
				},
			},
			wantErr: true,
		},
		{
			name: "ForgotPassword_NilManager",
			intent: &domain.Intent{
				Type:   domain.IntentForgotPassword,
				Params: map[string]string{},
			},
			userID:       "user-1",
			passwords:    nil, // nil means not configured
			wantContains: []string{"not configured"},
		},
		{
			name: "ListProjects_WithMemberCounts",
			intent: &domain.Intent{Type: domain.IntentListProjects},
			userID: "user-1",
			projects: &mockProjectManager{
				listFn: func(_ context.Context, _ string, _, _ int) ([]domain.ProjectSummary, int, error) {
					return []domain.ProjectSummary{
						{ID: "p1", Name: "Active", Status: "active", MemberCount: 10},
						{ID: "p2", Name: "Draft", Status: "draft", MemberCount: 0},
					}, 2, nil
				},
			},
			wantContains:    []string{"Active", "10 уч.", "Draft", "📌"},
			wantNotContains: []string{"Draft · 0"}, // zero member count should not show
		},
		{
			name: "GetProjectStatus_CaseInsensitive",
			intent: &domain.Intent{
				Type:   domain.IntentGetProjectStatus,
				Params: map[string]string{"project_name": "alpha"},
			},
			userID: "user-1",
			projects: &mockProjectManager{
				listFn: func(_ context.Context, _ string, _, _ int) ([]domain.ProjectSummary, int, error) {
					return []domain.ProjectSummary{
						{ID: "p1", Name: "Alpha", Status: "active", MemberCount: 3},
					}, 1, nil
				},
			},
			wantContains: []string{"Alpha", "active"},
		},
		{
			name: "ListMembers_ByName_CaseInsensitive",
			intent: &domain.Intent{
				Type:   domain.IntentListMembers,
				Params: map[string]string{"project_name": "alpha"},
			},
			userID: "user-1",
			projects: &mockProjectManager{
				listFn: func(_ context.Context, _ string, _, _ int) ([]domain.ProjectSummary, int, error) {
					return []domain.ProjectSummary{
						{ID: "p1", Name: "Alpha", Status: "active"},
					}, 1, nil
				},
			},
			members: &mockMemberManager{
				listFn: func(_ context.Context, projectID string) ([]domain.MemberSummary, error) {
					if projectID != "p1" {
						return nil, errors.New("wrong project_id passed: " + projectID)
					}
					return []domain.MemberSummary{
						{UserID: "u1", UserName: "Alice", Role: "admin"},
					}, nil
				},
			},
			wantContains: []string{"Alice", "admin"},
		},
		{
			name: "GetAggregated_ByName_CaseInsensitive",
			intent: &domain.Intent{
				Type:   domain.IntentGetAggregated,
				Params: map[string]string{"project_name": "ALPHA"},
			},
			userID: "user-1",
			projects: &mockProjectManager{
				listFn: func(_ context.Context, _ string, _, _ int) ([]domain.ProjectSummary, int, error) {
					return []domain.ProjectSummary{
						{ID: "p1", Name: "Alpha", Status: "active"},
					}, 1, nil
				},
			},
			estimations: &mockEstimationManager{
				getAggregatedFn: func(_ context.Context, projectID string) (string, error) {
					if projectID != "p1" {
						return "", errors.New("wrong project_id passed: " + projectID)
					}
					return "Общая оценка: 80ч", nil
				},
			},
			wantContains: []string{"80ч"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			projects := tc.projects
			if projects == nil {
				projects = &mockProjectManager{}
			}
			members := tc.members
			if members == nil {
				members = &mockMemberManager{}
			}
			estimations := tc.estimations
			if estimations == nil {
				estimations = &mockEstimationManager{}
			}
			docs := &mockDocumentManager{}

			var passwords domain.PasswordResetManager
			if tc.passwords != nil {
				passwords = tc.passwords
			}

			executor := usecase.NewIntentExecutor(projects, members, estimations, docs, passwords)

			msg, keyboard, err := executor.Execute(t.Context(), tc.intent, tc.userID)
			if tc.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			for _, want := range tc.wantContains {
				if !strings.Contains(msg, want) {
					t.Errorf("message should contain %q, got:\n%s", want, msg)
				}
			}

			for _, notWant := range tc.wantNotContains {
				if strings.Contains(msg, notWant) {
					t.Errorf("message should NOT contain %q, got:\n%s", notWant, msg)
				}
			}

			if tc.wantKeyboard && keyboard == nil {
				t.Error("expected keyboard, got nil")
			}
			if !tc.wantKeyboard && keyboard != nil {
				t.Errorf("expected no keyboard, got %v", keyboard)
			}

			if len(tc.wantKeyboardTexts) > 0 && keyboard != nil {
				allTexts := make(map[string]bool)
				for _, row := range keyboard {
					for _, btn := range row {
						allTexts[btn.Text] = true
					}
				}
				for _, want := range tc.wantKeyboardTexts {
					if !allTexts[want] {
						t.Errorf("keyboard should contain button %q, got buttons: %v", want, allTexts)
					}
				}
			}
		})
	}
}

// TestExecute_CreateProject_CancelButtonUsesColonFormat verifies the cancel
// button in createProject's keyboard uses the canonical "cancel:" format
// (action:payload convention) rather than legacy "cancel" without colon.
// See issue #20.
func TestExecute_CreateProject_CancelButtonUsesColonFormat(t *testing.T) {
	executor := usecase.NewIntentExecutor(&mockProjectManager{}, &mockMemberManager{}, &mockEstimationManager{}, &mockDocumentManager{}, nil)
	intent := &domain.Intent{
		Type:   domain.IntentCreateProject,
		Params: map[string]string{"name": "X"},
	}
	_, keyboard, err := executor.Execute(t.Context(), intent, "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cancelData := findButtonCallbackData(keyboard, "Отмена")
	if cancelData != "cancel:" {
		t.Errorf("CreateProject cancel button: CallbackData = %q, want %q", cancelData, "cancel:")
	}
}

// TestExecute_RemoveMember_CancelButtonUsesColonFormat verifies the cancel
// button in removeMember's keyboard uses the canonical "cancel:" format.
// See issue #20.
func TestExecute_RemoveMember_CancelButtonUsesColonFormat(t *testing.T) {
	executor := usecase.NewIntentExecutor(&mockProjectManager{}, &mockMemberManager{}, &mockEstimationManager{}, &mockDocumentManager{}, nil)
	intent := &domain.Intent{
		Type:   domain.IntentRemoveMember,
		Params: map[string]string{"project_name": "Alpha", "user_name": "John"},
	}
	_, keyboard, err := executor.Execute(t.Context(), intent, "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cancelData := findButtonCallbackData(keyboard, "Отмена")
	if cancelData != "cancel:" {
		t.Errorf("RemoveMember cancel button: CallbackData = %q, want %q", cancelData, "cancel:")
	}
}

// TestExecute_AddMember_RoleButtonsUseSelPrefix verifies that role-selection
// buttons in addMember's keyboard use canonical "sel_role:*" CallbackData.
// Pre-fix the callback was "role:developer" → ProcessCallback default →
// silent skip → flow hung until session TTL. See issue #26 (found via #21).
func TestExecute_AddMember_RoleButtonsUseSelPrefix(t *testing.T) {
	executor := usecase.NewIntentExecutor(&mockProjectManager{}, &mockMemberManager{}, &mockEstimationManager{}, &mockDocumentManager{}, nil)
	intent := &domain.Intent{
		Type:   domain.IntentAddMember,
		Params: map[string]string{"project_name": "Backend", "email": "dev@example.com"},
	}
	_, keyboard, err := executor.Execute(t.Context(), intent, "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, role := range []string{"Developer", "Tech Lead", "PM", "Observer", "Admin"} {
		got := findButtonCallbackData(keyboard, role)
		if got == "" {
			t.Errorf("button %q not found in keyboard", role)
			continue
		}
		if !strings.HasPrefix(got, "sel_role:") {
			t.Errorf("button %q: CallbackData = %q, want prefix \"sel_role:\"", role, got)
		}
	}
}

// TestExecute_AllValidIntentsHaveCase is a defensive gate that protects
// against the regression that caused issue #19: an intent declared in
// IntentType.IsValid() but missing a case in Execute switch falls through
// to default → unknown() → bot says «не понял команду» despite confidently
// classifying the message.
//
// If you add a new IntentType, you MUST add a corresponding case in Execute.
// This test will fail otherwise.
func TestExecute_AllValidIntentsHaveCase(t *testing.T) {
	// All currently-defined intents (mirrors IntentType.IsValid in
	// bot/domain/entities.go). Excludes IntentUnknown which by design
	// goes to default → unknown().
	allValidIntents := []domain.IntentType{
		domain.IntentCreateProject,
		domain.IntentUpdateProject,
		domain.IntentListProjects,
		domain.IntentGetProjectStatus,
		domain.IntentAddMember,
		domain.IntentRemoveMember,
		domain.IntentListMembers,
		domain.IntentRequestEstimation,
		domain.IntentSubmitEstimation,
		domain.IntentGetAggregated,
		domain.IntentUploadDocument,
		domain.IntentForgotPassword,
		domain.IntentHelp,
	}

	executor := usecase.NewIntentExecutor(
		&mockProjectManager{},
		&mockMemberManager{},
		&mockEstimationManager{},
		&mockDocumentManager{},
		&mockPasswordResetManager{},
	)
	const unknownMarker = "Не удалось распознать команду"

	for _, it := range allValidIntents {
		t.Run(string(it), func(t *testing.T) {
			if !it.IsValid() {
				t.Fatalf("intent %q is in test list but IsValid()==false — fix the test list or domain.IsValid", it)
			}
			intent := &domain.Intent{Type: it, Params: map[string]string{}}
			msg, _, _ := executor.Execute(t.Context(), intent, "user-1")
			if strings.Contains(msg, unknownMarker) {
				t.Errorf("intent %q falls through to default — add a case to IntentExecutor.Execute switch", it)
			}
		})
	}
}

// TestExecute_UploadDocument_EnrichesParamsWithProjectID verifies that the
// upload_document intent executor resolves project_name → project_id and
// stores the resolved ID back into intent.Params, so that BotUsecase
// (which serialises intent.Params into the new session state) makes
// project_id available to handleFileUpload when the file actually arrives.
// See issue #19.
func TestExecute_UploadDocument_EnrichesParamsWithProjectID(t *testing.T) {
	projects := &mockProjectManager{
		listFn: func(_ context.Context, _ string, _, _ int) ([]domain.ProjectSummary, int, error) {
			return []domain.ProjectSummary{{ID: "p1", Name: "Alpha"}}, 1, nil
		},
	}
	executor := usecase.NewIntentExecutor(projects, &mockMemberManager{}, &mockEstimationManager{}, &mockDocumentManager{}, nil)
	intent := &domain.Intent{
		Type:   domain.IntentUploadDocument,
		Params: map[string]string{"project_name": "Alpha"},
	}
	_, _, err := executor.Execute(t.Context(), intent, "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := intent.Params["project_id"]; got != "p1" {
		t.Errorf("project_id enrichment: got %q, want p1", got)
	}
}

// findButtonCallbackData returns the CallbackData of the first button matching
// text, or empty string if not found.
func findButtonCallbackData(keyboard [][]domain.InlineKeyboardButton, text string) string {
	for _, row := range keyboard {
		for _, btn := range row {
			if btn.Text == text {
				return btn.CallbackData
			}
		}
	}
	return ""
}

func TestNeedsSession(t *testing.T) {
	tests := []struct {
		intent domain.IntentType
		want   bool
	}{
		{domain.IntentCreateProject, true},
		{domain.IntentUpdateProject, true},
		{domain.IntentAddMember, true},
		{domain.IntentRemoveMember, true},
		{domain.IntentListProjects, false},
		{domain.IntentHelp, false},
		{domain.IntentGetAggregated, false},
		{domain.IntentListMembers, false},
		{domain.IntentGetProjectStatus, false},
		{domain.IntentUploadDocument, true},
		{domain.IntentForgotPassword, false},
		{domain.IntentUnknown, false},
	}

	for _, tc := range tests {
		t.Run(string(tc.intent), func(t *testing.T) {
			got := usecase.NeedsSession(tc.intent)
			if got != tc.want {
				t.Errorf("NeedsSession(%s) = %v, want %v", tc.intent, got, tc.want)
			}
		})
	}
}
