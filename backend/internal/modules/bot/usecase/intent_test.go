// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package usecase_test

import (
	"context"
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
	getAggregatedFn func(ctx context.Context, projectID string) (string, error)
}

func (m *mockEstimationManager) GetAggregated(ctx context.Context, projectID string) (string, error) {
	if m.getAggregatedFn != nil {
		return m.getAggregatedFn(ctx, projectID)
	}
	return "", nil
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

// --- Tests ---

func TestExecute(t *testing.T) {
	tests := []struct {
		name            string
		intent          *domain.Intent
		userID          string
		projects        *mockProjectManager
		members         *mockMemberManager
		estimations     *mockEstimationManager
		wantContains       []string
		wantNotContains    []string
		wantKeyboard       bool
		wantKeyboardTexts  []string
		wantErr            bool
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
			name:   "Unknown",
			intent: &domain.Intent{Type: domain.IntentUnknown},
			userID: "user-1",
			wantContains: []string{
				"Не удалось распознать",
				"помощь",
			},
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
			name: "CreateProject_ReturnsConfirmation",
			intent: &domain.Intent{
				Type:   domain.IntentCreateProject,
				Params: map[string]string{"name": "NewProject", "description": "A cool project"},
			},
			userID:       "user-1",
			wantContains:      []string{"NewProject", "A cool project"},
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

			executor := usecase.NewIntentExecutor(projects, members, estimations, docs)

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
