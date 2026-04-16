package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/VDV001/estimate-pro/backend/internal/modules/project/domain"
	"github.com/VDV001/estimate-pro/backend/internal/modules/project/handler"
	"github.com/VDV001/estimate-pro/backend/internal/modules/project/usecase"
	"github.com/VDV001/estimate-pro/backend/internal/shared/middleware"
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

func (m *mockWorkspaceRepo) Update(_ context.Context, ws *domain.Workspace) error {
	for i, w := range m.workspaces {
		if w.ID == ws.ID {
			m.workspaces[i] = ws
			return nil
		}
	}
	return domain.ErrWorkspaceNotFound
}

// --- Mock project repository ---

type mockProjectRepo struct {
	projects  []*domain.Project
	createErr error
}

func (m *mockProjectRepo) Create(_ context.Context, p *domain.Project) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.projects = append(m.projects, p)
	return nil
}

func (m *mockProjectRepo) GetByID(_ context.Context, id string) (*domain.Project, error) {
	for _, p := range m.projects {
		if p.ID == id {
			return p, nil
		}
	}
	return nil, domain.ErrProjectNotFound
}

func (m *mockProjectRepo) ListByWorkspace(_ context.Context, workspaceID string, limit, offset int) ([]*domain.Project, int, error) {
	var result []*domain.Project
	for _, p := range m.projects {
		if p.WorkspaceID == workspaceID {
			result = append(result, p)
		}
	}
	total := len(result)
	if offset > len(result) {
		result = nil
	} else {
		result = result[offset:]
	}
	if limit < len(result) {
		result = result[:limit]
	}
	return result, total, nil
}

func (m *mockProjectRepo) ListByUser(_ context.Context, _ string, limit, offset int) ([]*domain.Project, int, error) {
	total := len(m.projects)
	result := m.projects
	if offset > len(result) {
		result = nil
	} else {
		result = result[offset:]
	}
	if limit < len(result) {
		result = result[:limit]
	}
	return result, total, nil
}

func (m *mockProjectRepo) Update(_ context.Context, p *domain.Project) error {
	for i, existing := range m.projects {
		if existing.ID == p.ID {
			m.projects[i] = p
			return nil
		}
	}
	return domain.ErrProjectNotFound
}

func (m *mockProjectRepo) Delete(_ context.Context, id string) error {
	for i, p := range m.projects {
		if p.ID == id {
			m.projects = append(m.projects[:i], m.projects[i+1:]...)
			return nil
		}
	}
	return domain.ErrProjectNotFound
}

// --- Mock member repository ---

type mockMemberRepo struct {
	members   []*domain.Member
	addErr    error
	removeErr error
}

func (m *mockMemberRepo) Add(_ context.Context, member *domain.Member) error {
	if m.addErr != nil {
		return m.addErr
	}
	for _, existing := range m.members {
		if existing.ProjectID == member.ProjectID && existing.UserID == member.UserID {
			return domain.ErrMemberAlreadyAdded
		}
	}
	m.members = append(m.members, member)
	return nil
}

func (m *mockMemberRepo) Remove(_ context.Context, projectID, userID string) error {
	if m.removeErr != nil {
		return m.removeErr
	}
	for i, member := range m.members {
		if member.ProjectID == projectID && member.UserID == userID {
			m.members = append(m.members[:i], m.members[i+1:]...)
			return nil
		}
	}
	return domain.ErrMemberNotFound
}

func (m *mockMemberRepo) ListByProject(_ context.Context, projectID string) ([]*domain.Member, error) {
	var result []*domain.Member
	for _, member := range m.members {
		if member.ProjectID == projectID {
			result = append(result, member)
		}
	}
	return result, nil
}

func (m *mockMemberRepo) GetRole(_ context.Context, projectID, userID string) (domain.Role, error) {
	for _, member := range m.members {
		if member.ProjectID == projectID && member.UserID == userID {
			return member.Role, nil
		}
	}
	return "", domain.ErrMemberNotFound
}

func (m *mockMemberRepo) ListByProjectWithUsers(_ context.Context, projectID string) ([]*domain.MemberWithUser, error) {
	var result []*domain.MemberWithUser
	for _, member := range m.members {
		if member.ProjectID == projectID {
			result = append(result, &domain.MemberWithUser{
				ProjectID: member.ProjectID,
				UserID:    member.UserID,
				Role:      member.Role,
				UserName:  "User " + member.UserID,
				UserEmail: member.UserID + "@test.com",
			})
		}
	}
	return result, nil
}

// --- Mock user finder ---

type mockUserFinder struct {
	users map[string]string // email -> userID
}

func (m *mockUserFinder) FindByEmail(_ context.Context, email string) (string, error) {
	if id, ok := m.users[email]; ok {
		return id, nil
	}
	return "", fmt.Errorf("user not found for email %q", email)
}

// --- Helpers ---

func requestWithUserID(r *http.Request, userID string) *http.Request {
	ctx := context.WithValue(r.Context(), middleware.UserIDKey, userID)
	return r.WithContext(ctx)
}

func requestWithChiParam(r *http.Request, key, value string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, value)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func requestWithChiParams(r *http.Request, params map[string]string) *http.Request {
	rctx := chi.NewRouteContext()
	for k, v := range params {
		rctx.URLParams.Add(k, v)
	}
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func newTestHandler(projectRepo domain.ProjectRepository, memberRepo domain.MemberRepository, workspaceRepo domain.WorkspaceRepository, userFinder domain.UserFinder) *handler.Handler {
	uc := usecase.New(projectRepo, workspaceRepo, memberRepo)
	memberUC := usecase.NewMemberUsecase(memberRepo, projectRepo, userFinder)
	workspaceUC := usecase.NewWorkspaceUsecase(workspaceRepo)
	return handler.New(uc, memberUC, workspaceUC)
}

func seedProject(id, wsID, name, createdBy string) *domain.Project {
	return &domain.Project{
		ID:          id,
		WorkspaceID: wsID,
		Name:        name,
		Description: "desc",
		Status:      domain.ProjectStatusActive,
		CreatedBy:   createdBy,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

// ==============================
// Workspace tests
// ==============================

func TestCreateWorkspace_Success(t *testing.T) {
	repo := &mockWorkspaceRepo{}
	h := handler.New(nil, nil, usecase.NewWorkspaceUsecase(repo))

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
	h := handler.New(nil, nil, usecase.NewWorkspaceUsecase(repo))

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
	h := handler.New(nil, nil, usecase.NewWorkspaceUsecase(repo))

	body, _ := json.Marshal(map[string]string{"name": "Test"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	h.CreateWorkspace(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rec.Code)
	}
}

func TestCreateWorkspace_InvalidJSON(t *testing.T) {
	repo := &mockWorkspaceRepo{}
	h := handler.New(nil, nil, usecase.NewWorkspaceUsecase(repo))

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
	h := handler.New(nil, nil, usecase.NewWorkspaceUsecase(repo))

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
	h := handler.New(nil, nil, usecase.NewWorkspaceUsecase(repo))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/workspaces", nil)
	rec := httptest.NewRecorder()

	h.ListWorkspaces(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rec.Code)
	}
}

func TestUpdateWorkspace_Success(t *testing.T) {
	repo := &mockWorkspaceRepo{
		workspaces: []*domain.Workspace{
			{ID: "ws-1", Name: "Old Name", OwnerID: "user-1"},
		},
	}
	h := handler.New(nil, nil, usecase.NewWorkspaceUsecase(repo))

	body, _ := json.Marshal(map[string]string{"name": "New Name"})
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/workspaces/ws-1", bytes.NewReader(body))
	req = requestWithUserID(req, "user-1")
	req = requestWithChiParam(req, "id", "ws-1")
	rec := httptest.NewRecorder()

	h.UpdateWorkspace(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d; body: %s", rec.Code, rec.Body.String())
	}

	var ws domain.Workspace
	if err := json.NewDecoder(rec.Body).Decode(&ws); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if ws.Name != "New Name" {
		t.Errorf("expected name %q, got %q", "New Name", ws.Name)
	}
}

func TestUpdateWorkspace_EmptyName(t *testing.T) {
	repo := &mockWorkspaceRepo{
		workspaces: []*domain.Workspace{
			{ID: "ws-1", Name: "Name", OwnerID: "user-1"},
		},
	}
	h := handler.New(nil, nil, usecase.NewWorkspaceUsecase(repo))

	body, _ := json.Marshal(map[string]string{"name": ""})
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/workspaces/ws-1", bytes.NewReader(body))
	req = requestWithUserID(req, "user-1")
	req = requestWithChiParam(req, "id", "ws-1")
	rec := httptest.NewRecorder()

	h.UpdateWorkspace(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
}

func TestUpdateWorkspace_NotFound(t *testing.T) {
	repo := &mockWorkspaceRepo{}
	h := handler.New(nil, nil, usecase.NewWorkspaceUsecase(repo))

	body, _ := json.Marshal(map[string]string{"name": "New"})
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/workspaces/ws-999", bytes.NewReader(body))
	req = requestWithUserID(req, "user-1")
	req = requestWithChiParam(req, "id", "ws-999")
	rec := httptest.NewRecorder()

	h.UpdateWorkspace(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rec.Code)
	}
}

func TestUpdateWorkspace_NotOwner(t *testing.T) {
	repo := &mockWorkspaceRepo{
		workspaces: []*domain.Workspace{
			{ID: "ws-1", Name: "Name", OwnerID: "user-1"},
		},
	}
	h := handler.New(nil, nil, usecase.NewWorkspaceUsecase(repo))

	body, _ := json.Marshal(map[string]string{"name": "Hacked"})
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/workspaces/ws-1", bytes.NewReader(body))
	req = requestWithUserID(req, "user-2") // not the owner
	req = requestWithChiParam(req, "id", "ws-1")
	rec := httptest.NewRecorder()

	h.UpdateWorkspace(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d", rec.Code)
	}
}

func TestUpdateWorkspace_InvalidJSON(t *testing.T) {
	repo := &mockWorkspaceRepo{}
	h := handler.New(nil, nil, usecase.NewWorkspaceUsecase(repo))

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/workspaces/ws-1", bytes.NewReader([]byte("bad")))
	req = requestWithUserID(req, "user-1")
	req = requestWithChiParam(req, "id", "ws-1")
	rec := httptest.NewRecorder()

	h.UpdateWorkspace(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
}

func TestUpdateWorkspace_MissingUserContext(t *testing.T) {
	repo := &mockWorkspaceRepo{}
	h := handler.New(nil, nil, usecase.NewWorkspaceUsecase(repo))

	body, _ := json.Marshal(map[string]string{"name": "New"})
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/workspaces/ws-1", bytes.NewReader(body))
	req = requestWithChiParam(req, "id", "ws-1")
	rec := httptest.NewRecorder()

	h.UpdateWorkspace(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rec.Code)
	}
}

// ==============================
// Project CRUD tests
// ==============================

func TestCreateProject_Success(t *testing.T) {
	wsRepo := &mockWorkspaceRepo{
		workspaces: []*domain.Workspace{{ID: "ws-1", Name: "WS", OwnerID: "user-1"}},
	}
	projRepo := &mockProjectRepo{}
	memberRepo := &mockMemberRepo{}
	h := newTestHandler(projRepo, memberRepo, wsRepo, nil)

	body, _ := json.Marshal(map[string]string{
		"workspace_id": "ws-1",
		"name":         "My Project",
		"description":  "A cool project",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects", bytes.NewReader(body))
	req = requestWithUserID(req, "user-1")
	rec := httptest.NewRecorder()

	h.CreateProject(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d; body: %s", rec.Code, rec.Body.String())
	}

	var p domain.Project
	if err := json.NewDecoder(rec.Body).Decode(&p); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if p.Name != "My Project" {
		t.Errorf("expected name %q, got %q", "My Project", p.Name)
	}
	if p.WorkspaceID != "ws-1" {
		t.Errorf("expected workspace_id %q, got %q", "ws-1", p.WorkspaceID)
	}
	if p.ID == "" {
		t.Error("expected non-empty ID")
	}
}

func TestCreateProject_MissingName(t *testing.T) {
	wsRepo := &mockWorkspaceRepo{}
	h := newTestHandler(&mockProjectRepo{}, &mockMemberRepo{}, wsRepo, nil)

	body, _ := json.Marshal(map[string]string{"workspace_id": "ws-1"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects", bytes.NewReader(body))
	req = requestWithUserID(req, "user-1")
	rec := httptest.NewRecorder()

	h.CreateProject(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
}

func TestCreateProject_MissingWorkspaceID(t *testing.T) {
	h := newTestHandler(&mockProjectRepo{}, &mockMemberRepo{}, &mockWorkspaceRepo{}, nil)

	body, _ := json.Marshal(map[string]string{"name": "Proj"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects", bytes.NewReader(body))
	req = requestWithUserID(req, "user-1")
	rec := httptest.NewRecorder()

	h.CreateProject(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
}

func TestCreateProject_InvalidJSON(t *testing.T) {
	h := newTestHandler(&mockProjectRepo{}, &mockMemberRepo{}, &mockWorkspaceRepo{}, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects", bytes.NewReader([]byte("{")))
	req = requestWithUserID(req, "user-1")
	rec := httptest.NewRecorder()

	h.CreateProject(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
}

func TestCreateProject_MissingUserContext(t *testing.T) {
	h := newTestHandler(&mockProjectRepo{}, &mockMemberRepo{}, &mockWorkspaceRepo{}, nil)

	body, _ := json.Marshal(map[string]string{"workspace_id": "ws-1", "name": "Proj"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	h.CreateProject(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rec.Code)
	}
}

func TestListProjects_ByWorkspace(t *testing.T) {
	projRepo := &mockProjectRepo{
		projects: []*domain.Project{
			seedProject("p-1", "ws-1", "Proj 1", "user-1"),
			seedProject("p-2", "ws-1", "Proj 2", "user-1"),
			seedProject("p-3", "ws-2", "Proj 3", "user-2"),
		},
	}
	h := newTestHandler(projRepo, &mockMemberRepo{}, &mockWorkspaceRepo{}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects?workspace_id=ws-1", nil)
	req = requestWithUserID(req, "user-1")
	rec := httptest.NewRecorder()

	h.ListProjects(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d; body: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Projects []*domain.Project `json:"projects"`
		Meta     struct {
			Total int `json:"total"`
		} `json:"meta"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if len(resp.Projects) != 2 {
		t.Fatalf("expected 2 projects, got %d", len(resp.Projects))
	}
	if resp.Meta.Total != 2 {
		t.Errorf("expected total 2, got %d", resp.Meta.Total)
	}
}

func TestListProjects_ByUser(t *testing.T) {
	projRepo := &mockProjectRepo{
		projects: []*domain.Project{
			seedProject("p-1", "ws-1", "Proj 1", "user-1"),
		},
	}
	h := newTestHandler(projRepo, &mockMemberRepo{}, &mockWorkspaceRepo{}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects", nil)
	req = requestWithUserID(req, "user-1")
	rec := httptest.NewRecorder()

	h.ListProjects(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var resp struct {
		Projects []*domain.Project `json:"projects"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if len(resp.Projects) != 1 {
		t.Fatalf("expected 1 project, got %d", len(resp.Projects))
	}
}

func TestListProjects_Empty(t *testing.T) {
	projRepo := &mockProjectRepo{}
	h := newTestHandler(projRepo, &mockMemberRepo{}, &mockWorkspaceRepo{}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects?workspace_id=ws-1", nil)
	req = requestWithUserID(req, "user-1")
	rec := httptest.NewRecorder()

	h.ListProjects(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var resp struct {
		Projects []*domain.Project `json:"projects"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	// Null or empty slice is acceptable — just check no error
}

func TestListProjects_MissingUserContext(t *testing.T) {
	h := newTestHandler(&mockProjectRepo{}, &mockMemberRepo{}, &mockWorkspaceRepo{}, nil)

	// No workspace_id → handler will try to get userID from context
	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects", nil)
	rec := httptest.NewRecorder()

	h.ListProjects(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rec.Code)
	}
}

func TestGetProject_Success(t *testing.T) {
	projRepo := &mockProjectRepo{
		projects: []*domain.Project{seedProject("p-1", "ws-1", "Proj", "user-1")},
	}
	h := newTestHandler(projRepo, &mockMemberRepo{}, &mockWorkspaceRepo{}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/p-1", nil)
	req = requestWithChiParam(req, "id", "p-1")
	rec := httptest.NewRecorder()

	h.GetProject(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d; body: %s", rec.Code, rec.Body.String())
	}

	var p domain.Project
	if err := json.NewDecoder(rec.Body).Decode(&p); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if p.ID != "p-1" {
		t.Errorf("expected id %q, got %q", "p-1", p.ID)
	}
	if p.Name != "Proj" {
		t.Errorf("expected name %q, got %q", "Proj", p.Name)
	}
}

func TestGetProject_NotFound(t *testing.T) {
	projRepo := &mockProjectRepo{}
	h := newTestHandler(projRepo, &mockMemberRepo{}, &mockWorkspaceRepo{}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/p-999", nil)
	req = requestWithChiParam(req, "id", "p-999")
	rec := httptest.NewRecorder()

	h.GetProject(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rec.Code)
	}
}

func TestUpdateProject_Success(t *testing.T) {
	projRepo := &mockProjectRepo{
		projects: []*domain.Project{seedProject("p-1", "ws-1", "Old", "user-1")},
	}
	memberRepo := &mockMemberRepo{
		members: []*domain.Member{
			{ProjectID: "p-1", UserID: "user-1", Role: domain.RoleAdmin},
		},
	}
	h := newTestHandler(projRepo, memberRepo, &mockWorkspaceRepo{}, nil)

	body, _ := json.Marshal(map[string]string{"name": "New Name", "description": "Updated"})
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/projects/p-1", bytes.NewReader(body))
	req = requestWithUserID(req, "user-1")
	req = requestWithChiParam(req, "id", "p-1")
	rec := httptest.NewRecorder()

	h.UpdateProject(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d; body: %s", rec.Code, rec.Body.String())
	}

	var p domain.Project
	if err := json.NewDecoder(rec.Body).Decode(&p); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if p.Name != "New Name" {
		t.Errorf("expected name %q, got %q", "New Name", p.Name)
	}
}

func TestUpdateProject_NotFound(t *testing.T) {
	// User is not a member of non-existent project → GetRole fails → usecase returns error
	projRepo := &mockProjectRepo{}
	memberRepo := &mockMemberRepo{}
	h := newTestHandler(projRepo, memberRepo, &mockWorkspaceRepo{}, nil)

	body, _ := json.Marshal(map[string]string{"name": "X"})
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/projects/p-999", bytes.NewReader(body))
	req = requestWithUserID(req, "user-1")
	req = requestWithChiParam(req, "id", "p-999")
	rec := httptest.NewRecorder()

	h.UpdateProject(rec, req)

	// Handler maps any usecase error to 500
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rec.Code)
	}
}

func TestUpdateProject_InvalidJSON(t *testing.T) {
	h := newTestHandler(&mockProjectRepo{}, &mockMemberRepo{}, &mockWorkspaceRepo{}, nil)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/projects/p-1", bytes.NewReader([]byte("bad")))
	req = requestWithUserID(req, "user-1")
	req = requestWithChiParam(req, "id", "p-1")
	rec := httptest.NewRecorder()

	h.UpdateProject(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
}

func TestUpdateProject_MissingUserContext(t *testing.T) {
	h := newTestHandler(&mockProjectRepo{}, &mockMemberRepo{}, &mockWorkspaceRepo{}, nil)

	body, _ := json.Marshal(map[string]string{"name": "X"})
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/projects/p-1", bytes.NewReader(body))
	req = requestWithChiParam(req, "id", "p-1")
	rec := httptest.NewRecorder()

	h.UpdateProject(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rec.Code)
	}
}

func TestDeleteProject_Success(t *testing.T) {
	projRepo := &mockProjectRepo{
		projects: []*domain.Project{seedProject("p-1", "ws-1", "Proj", "user-1")},
	}
	memberRepo := &mockMemberRepo{
		members: []*domain.Member{
			{ProjectID: "p-1", UserID: "user-1", Role: domain.RoleAdmin},
		},
	}
	h := newTestHandler(projRepo, memberRepo, &mockWorkspaceRepo{}, nil)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/projects/p-1", nil)
	req = requestWithUserID(req, "user-1")
	req = requestWithChiParam(req, "id", "p-1")
	rec := httptest.NewRecorder()

	h.DeleteProject(rec, req)

	// DeleteProject (Archive) returns 200 with project body
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d; body: %s", rec.Code, rec.Body.String())
	}

	var p domain.Project
	if err := json.NewDecoder(rec.Body).Decode(&p); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if p.Status != domain.ProjectStatusArchived {
		t.Errorf("expected status %q, got %q", domain.ProjectStatusArchived, p.Status)
	}
}

func TestDeleteProject_NotFound(t *testing.T) {
	projRepo := &mockProjectRepo{}
	memberRepo := &mockMemberRepo{}
	h := newTestHandler(projRepo, memberRepo, &mockWorkspaceRepo{}, nil)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/projects/p-999", nil)
	req = requestWithUserID(req, "user-1")
	req = requestWithChiParam(req, "id", "p-999")
	rec := httptest.NewRecorder()

	h.DeleteProject(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rec.Code)
	}
}

func TestDeleteProject_MissingUserContext(t *testing.T) {
	h := newTestHandler(&mockProjectRepo{}, &mockMemberRepo{}, &mockWorkspaceRepo{}, nil)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/projects/p-1", nil)
	req = requestWithChiParam(req, "id", "p-1")
	rec := httptest.NewRecorder()

	h.DeleteProject(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rec.Code)
	}
}

func TestRestoreProject_Success(t *testing.T) {
	proj := seedProject("p-1", "ws-1", "Proj", "user-1")
	proj.Status = domain.ProjectStatusArchived
	projRepo := &mockProjectRepo{projects: []*domain.Project{proj}}
	memberRepo := &mockMemberRepo{
		members: []*domain.Member{
			{ProjectID: "p-1", UserID: "user-1", Role: domain.RoleAdmin},
		},
	}
	h := newTestHandler(projRepo, memberRepo, &mockWorkspaceRepo{}, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/p-1/restore", nil)
	req = requestWithUserID(req, "user-1")
	req = requestWithChiParam(req, "id", "p-1")
	rec := httptest.NewRecorder()

	h.RestoreProject(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d; body: %s", rec.Code, rec.Body.String())
	}

	var p domain.Project
	if err := json.NewDecoder(rec.Body).Decode(&p); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if p.Status != domain.ProjectStatusActive {
		t.Errorf("expected status %q, got %q", domain.ProjectStatusActive, p.Status)
	}
}

func TestRestoreProject_NotFound(t *testing.T) {
	h := newTestHandler(&mockProjectRepo{}, &mockMemberRepo{}, &mockWorkspaceRepo{}, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/p-999/restore", nil)
	req = requestWithUserID(req, "user-1")
	req = requestWithChiParam(req, "id", "p-999")
	rec := httptest.NewRecorder()

	h.RestoreProject(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rec.Code)
	}
}

func TestRestoreProject_MissingUserContext(t *testing.T) {
	h := newTestHandler(&mockProjectRepo{}, &mockMemberRepo{}, &mockWorkspaceRepo{}, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/p-1/restore", nil)
	req = requestWithChiParam(req, "id", "p-1")
	rec := httptest.NewRecorder()

	h.RestoreProject(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rec.Code)
	}
}

// ==============================
// Member management tests
// ==============================

func TestAddMember_Success(t *testing.T) {
	projRepo := &mockProjectRepo{
		projects: []*domain.Project{seedProject("p-1", "ws-1", "Proj", "user-1")},
	}
	memberRepo := &mockMemberRepo{
		members: []*domain.Member{
			{ProjectID: "p-1", UserID: "user-1", Role: domain.RoleAdmin},
		},
	}
	userFinder := &mockUserFinder{users: map[string]string{"bob@test.com": "user-2"}}
	h := newTestHandler(projRepo, memberRepo, &mockWorkspaceRepo{}, userFinder)

	body, _ := json.Marshal(map[string]string{"email": "bob@test.com", "role": "developer"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/p-1/members", bytes.NewReader(body))
	req = requestWithUserID(req, "user-1")
	req = requestWithChiParam(req, "id", "p-1")
	rec := httptest.NewRecorder()

	h.AddMember(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d; body: %s", rec.Code, rec.Body.String())
	}
}

func TestAddMember_UserNotFound(t *testing.T) {
	projRepo := &mockProjectRepo{
		projects: []*domain.Project{seedProject("p-1", "ws-1", "Proj", "user-1")},
	}
	memberRepo := &mockMemberRepo{
		members: []*domain.Member{
			{ProjectID: "p-1", UserID: "user-1", Role: domain.RoleAdmin},
		},
	}
	userFinder := &mockUserFinder{users: map[string]string{}} // no users
	h := newTestHandler(projRepo, memberRepo, &mockWorkspaceRepo{}, userFinder)

	body, _ := json.Marshal(map[string]string{"email": "nobody@test.com", "role": "developer"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/p-1/members", bytes.NewReader(body))
	req = requestWithUserID(req, "user-1")
	req = requestWithChiParam(req, "id", "p-1")
	rec := httptest.NewRecorder()

	h.AddMember(rec, req)

	// userFinder returns generic error → handler maps to 500
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d; body: %s", rec.Code, rec.Body.String())
	}
}

func TestAddMember_AlreadyMember(t *testing.T) {
	projRepo := &mockProjectRepo{
		projects: []*domain.Project{seedProject("p-1", "ws-1", "Proj", "user-1")},
	}
	memberRepo := &mockMemberRepo{
		members: []*domain.Member{
			{ProjectID: "p-1", UserID: "user-1", Role: domain.RoleAdmin},
			{ProjectID: "p-1", UserID: "user-2", Role: domain.RoleDeveloper},
		},
	}
	userFinder := &mockUserFinder{users: map[string]string{"bob@test.com": "user-2"}}
	h := newTestHandler(projRepo, memberRepo, &mockWorkspaceRepo{}, userFinder)

	body, _ := json.Marshal(map[string]string{"email": "bob@test.com", "role": "developer"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/p-1/members", bytes.NewReader(body))
	req = requestWithUserID(req, "user-1")
	req = requestWithChiParam(req, "id", "p-1")
	rec := httptest.NewRecorder()

	h.AddMember(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected status 409, got %d; body: %s", rec.Code, rec.Body.String())
	}
}

func TestAddMember_MissingEmail(t *testing.T) {
	h := newTestHandler(&mockProjectRepo{}, &mockMemberRepo{}, &mockWorkspaceRepo{}, nil)

	body, _ := json.Marshal(map[string]string{"role": "developer"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/p-1/members", bytes.NewReader(body))
	req = requestWithUserID(req, "user-1")
	req = requestWithChiParam(req, "id", "p-1")
	rec := httptest.NewRecorder()

	h.AddMember(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
}

func TestAddMember_MissingRole(t *testing.T) {
	h := newTestHandler(&mockProjectRepo{}, &mockMemberRepo{}, &mockWorkspaceRepo{}, nil)

	body, _ := json.Marshal(map[string]string{"email": "bob@test.com"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/p-1/members", bytes.NewReader(body))
	req = requestWithUserID(req, "user-1")
	req = requestWithChiParam(req, "id", "p-1")
	rec := httptest.NewRecorder()

	h.AddMember(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
}

func TestAddMember_InvalidJSON(t *testing.T) {
	h := newTestHandler(&mockProjectRepo{}, &mockMemberRepo{}, &mockWorkspaceRepo{}, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/p-1/members", bytes.NewReader([]byte("bad")))
	req = requestWithUserID(req, "user-1")
	req = requestWithChiParam(req, "id", "p-1")
	rec := httptest.NewRecorder()

	h.AddMember(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
}

func TestAddMember_MissingUserContext(t *testing.T) {
	h := newTestHandler(&mockProjectRepo{}, &mockMemberRepo{}, &mockWorkspaceRepo{}, nil)

	body, _ := json.Marshal(map[string]string{"email": "bob@test.com", "role": "developer"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/p-1/members", bytes.NewReader(body))
	req = requestWithChiParam(req, "id", "p-1")
	rec := httptest.NewRecorder()

	h.AddMember(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rec.Code)
	}
}

func TestAddMember_InsufficientRole(t *testing.T) {
	projRepo := &mockProjectRepo{
		projects: []*domain.Project{seedProject("p-1", "ws-1", "Proj", "user-1")},
	}
	memberRepo := &mockMemberRepo{
		members: []*domain.Member{
			{ProjectID: "p-1", UserID: "user-1", Role: domain.RoleDeveloper}, // not admin/pm
		},
	}
	userFinder := &mockUserFinder{users: map[string]string{"bob@test.com": "user-2"}}
	h := newTestHandler(projRepo, memberRepo, &mockWorkspaceRepo{}, userFinder)

	body, _ := json.Marshal(map[string]string{"email": "bob@test.com", "role": "developer"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/p-1/members", bytes.NewReader(body))
	req = requestWithUserID(req, "user-1")
	req = requestWithChiParam(req, "id", "p-1")
	rec := httptest.NewRecorder()

	h.AddMember(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d; body: %s", rec.Code, rec.Body.String())
	}
}

func TestRemoveMember_Success(t *testing.T) {
	projRepo := &mockProjectRepo{
		projects: []*domain.Project{seedProject("p-1", "ws-1", "Proj", "user-1")},
	}
	memberRepo := &mockMemberRepo{
		members: []*domain.Member{
			{ProjectID: "p-1", UserID: "user-1", Role: domain.RoleAdmin},
			{ProjectID: "p-1", UserID: "user-2", Role: domain.RoleDeveloper},
		},
	}
	h := newTestHandler(projRepo, memberRepo, &mockWorkspaceRepo{}, nil)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/projects/p-1/members/user-2", nil)
	req = requestWithUserID(req, "user-1")
	req = requestWithChiParams(req, map[string]string{"id": "p-1", "userId": "user-2"})
	rec := httptest.NewRecorder()

	h.RemoveMember(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d; body: %s", rec.Code, rec.Body.String())
	}
}

func TestRemoveMember_NotFound(t *testing.T) {
	projRepo := &mockProjectRepo{
		projects: []*domain.Project{seedProject("p-1", "ws-1", "Proj", "user-1")},
	}
	memberRepo := &mockMemberRepo{
		members: []*domain.Member{
			{ProjectID: "p-1", UserID: "user-1", Role: domain.RoleAdmin},
		},
	}
	h := newTestHandler(projRepo, memberRepo, &mockWorkspaceRepo{}, nil)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/projects/p-1/members/user-999", nil)
	req = requestWithUserID(req, "user-1")
	req = requestWithChiParams(req, map[string]string{"id": "p-1", "userId": "user-999"})
	rec := httptest.NewRecorder()

	h.RemoveMember(rec, req)

	// GetRole for target fails → wraps ErrMemberNotFound → handler maps to 404
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d; body: %s", rec.Code, rec.Body.String())
	}
}

func TestRemoveMember_MissingUserContext(t *testing.T) {
	h := newTestHandler(&mockProjectRepo{}, &mockMemberRepo{}, &mockWorkspaceRepo{}, nil)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/projects/p-1/members/user-2", nil)
	req = requestWithChiParams(req, map[string]string{"id": "p-1", "userId": "user-2"})
	rec := httptest.NewRecorder()

	h.RemoveMember(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rec.Code)
	}
}

func TestRemoveMember_InsufficientRole(t *testing.T) {
	projRepo := &mockProjectRepo{
		projects: []*domain.Project{seedProject("p-1", "ws-1", "Proj", "user-1")},
	}
	memberRepo := &mockMemberRepo{
		members: []*domain.Member{
			{ProjectID: "p-1", UserID: "user-1", Role: domain.RoleObserver}, // can't manage
			{ProjectID: "p-1", UserID: "user-2", Role: domain.RoleDeveloper},
		},
	}
	h := newTestHandler(projRepo, memberRepo, &mockWorkspaceRepo{}, nil)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/projects/p-1/members/user-2", nil)
	req = requestWithUserID(req, "user-1")
	req = requestWithChiParams(req, map[string]string{"id": "p-1", "userId": "user-2"})
	rec := httptest.NewRecorder()

	h.RemoveMember(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d; body: %s", rec.Code, rec.Body.String())
	}
}

func TestRemoveMember_LastAdmin(t *testing.T) {
	projRepo := &mockProjectRepo{
		projects: []*domain.Project{seedProject("p-1", "ws-1", "Proj", "user-1")},
	}
	memberRepo := &mockMemberRepo{
		members: []*domain.Member{
			{ProjectID: "p-1", UserID: "user-1", Role: domain.RoleAdmin}, // only admin
		},
	}
	h := newTestHandler(projRepo, memberRepo, &mockWorkspaceRepo{}, nil)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/projects/p-1/members/user-1", nil)
	req = requestWithUserID(req, "user-1")
	req = requestWithChiParams(req, map[string]string{"id": "p-1", "userId": "user-1"})
	rec := httptest.NewRecorder()

	h.RemoveMember(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d; body: %s", rec.Code, rec.Body.String())
	}
}

func TestListMembers_Success(t *testing.T) {
	memberRepo := &mockMemberRepo{
		members: []*domain.Member{
			{ProjectID: "p-1", UserID: "user-1", Role: domain.RoleAdmin},
			{ProjectID: "p-1", UserID: "user-2", Role: domain.RoleDeveloper},
			{ProjectID: "p-2", UserID: "user-3", Role: domain.RoleObserver},
		},
	}
	h := newTestHandler(&mockProjectRepo{}, memberRepo, &mockWorkspaceRepo{}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/p-1/members", nil)
	req = requestWithChiParam(req, "id", "p-1")
	rec := httptest.NewRecorder()

	h.ListMembers(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d; body: %s", rec.Code, rec.Body.String())
	}

	var members []*domain.MemberWithUser
	if err := json.NewDecoder(rec.Body).Decode(&members); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if len(members) != 2 {
		t.Fatalf("expected 2 members, got %d", len(members))
	}
}

func TestListMembers_Empty(t *testing.T) {
	memberRepo := &mockMemberRepo{}
	h := newTestHandler(&mockProjectRepo{}, memberRepo, &mockWorkspaceRepo{}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/p-1/members", nil)
	req = requestWithChiParam(req, "id", "p-1")
	rec := httptest.NewRecorder()

	h.ListMembers(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
}
