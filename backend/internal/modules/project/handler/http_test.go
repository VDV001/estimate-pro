package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/daniilrusanov/estimate-pro/backend/internal/modules/project/domain"
	"github.com/daniilrusanov/estimate-pro/backend/internal/modules/project/handler"
	"github.com/daniilrusanov/estimate-pro/backend/internal/shared/middleware"
)

// --- Mock workspace repository ---

type mockWorkspaceRepo struct {
	workspaces []*domain.Workspace
	createErr  error
}

func (m *mockWorkspaceRepo) Create(_ context.Context, ws *domain.Workspace) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.workspaces = append(m.workspaces, ws)
	return nil
}

func (m *mockWorkspaceRepo) GetByID(_ context.Context, id string) (*domain.Workspace, error) {
	for _, ws := range m.workspaces {
		if ws.ID == id {
			return ws, nil
		}
	}
	return nil, domain.ErrWorkspaceNotFound
}

func (m *mockWorkspaceRepo) ListByUser(_ context.Context, userID string) ([]*domain.Workspace, error) {
	var result []*domain.Workspace
	for _, ws := range m.workspaces {
		if ws.OwnerID == userID {
			result = append(result, ws)
		}
	}
	return result, nil
}

// --- Helper to set user ID in context ---

func requestWithUserID(r *http.Request, userID string) *http.Request {
	ctx := context.WithValue(r.Context(), middleware.UserIDKey, userID)
	return r.WithContext(ctx)
}

// --- Tests ---

func TestCreateWorkspace_Success(t *testing.T) {
	repo := &mockWorkspaceRepo{}
	h := handler.New(nil, nil, repo)

	body, _ := json.Marshal(map[string]string{"name": "My Workspace"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces", bytes.NewReader(body))
	req = requestWithUserID(req, "user-1")
	rec := httptest.NewRecorder()

	h.CreateWorkspace(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", rec.Code)
	}

	var ws domain.Workspace
	if err := json.NewDecoder(rec.Body).Decode(&ws); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if ws.Name != "My Workspace" {
		t.Errorf("expected name %q, got %q", "My Workspace", ws.Name)
	}
	if ws.OwnerID != "user-1" {
		t.Errorf("expected owner_id %q, got %q", "user-1", ws.OwnerID)
	}
	if ws.ID == "" {
		t.Error("expected non-empty ID")
	}
	if ws.CreatedAt.IsZero() {
		t.Error("expected non-zero CreatedAt")
	}

	if len(repo.workspaces) != 1 {
		t.Fatalf("expected 1 workspace in repo, got %d", len(repo.workspaces))
	}
}

func TestCreateWorkspace_EmptyName(t *testing.T) {
	repo := &mockWorkspaceRepo{}
	h := handler.New(nil, nil, repo)

	body, _ := json.Marshal(map[string]string{"name": ""})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces", bytes.NewReader(body))
	req = requestWithUserID(req, "user-1")
	rec := httptest.NewRecorder()

	h.CreateWorkspace(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}

	if len(repo.workspaces) != 0 {
		t.Error("expected no workspace to be created")
	}
}

func TestCreateWorkspace_MissingUserContext(t *testing.T) {
	repo := &mockWorkspaceRepo{}
	h := handler.New(nil, nil, repo)

	body, _ := json.Marshal(map[string]string{"name": "Test"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces", bytes.NewReader(body))
	// No user ID in context
	rec := httptest.NewRecorder()

	h.CreateWorkspace(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rec.Code)
	}
}

func TestCreateWorkspace_InvalidJSON(t *testing.T) {
	repo := &mockWorkspaceRepo{}
	h := handler.New(nil, nil, repo)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces", bytes.NewReader([]byte("not json")))
	req = requestWithUserID(req, "user-1")
	rec := httptest.NewRecorder()

	h.CreateWorkspace(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
}

func TestListWorkspaces_ReturnsUserWorkspaces(t *testing.T) {
	repo := &mockWorkspaceRepo{
		workspaces: []*domain.Workspace{
			{ID: "ws-1", Name: "Workspace 1", OwnerID: "user-1"},
			{ID: "ws-2", Name: "Workspace 2", OwnerID: "user-1"},
			{ID: "ws-3", Name: "Other User WS", OwnerID: "user-2"},
		},
	}
	h := handler.New(nil, nil, repo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/workspaces", nil)
	req = requestWithUserID(req, "user-1")
	rec := httptest.NewRecorder()

	h.ListWorkspaces(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var workspaces []*domain.Workspace
	if err := json.NewDecoder(rec.Body).Decode(&workspaces); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(workspaces) != 2 {
		t.Fatalf("expected 2 workspaces, got %d", len(workspaces))
	}
}

func TestListWorkspaces_MissingUserContext(t *testing.T) {
	repo := &mockWorkspaceRepo{}
	h := handler.New(nil, nil, repo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/workspaces", nil)
	// No user ID in context
	rec := httptest.NewRecorder()

	h.ListWorkspaces(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rec.Code)
	}
}
