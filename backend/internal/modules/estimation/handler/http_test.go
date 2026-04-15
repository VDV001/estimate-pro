package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/VDV001/estimate-pro/backend/internal/modules/estimation/domain"
	"github.com/VDV001/estimate-pro/backend/internal/modules/estimation/handler"
	"github.com/VDV001/estimate-pro/backend/internal/modules/estimation/usecase"
	"github.com/VDV001/estimate-pro/backend/internal/shared/middleware"
)

// --- Mock EstimationRepository ---

type mockEstimationRepo struct {
	estimations map[string]*domain.Estimation
	byProject   map[string][]*domain.Estimation
	createErr   error
	updateErr   error
	deleteErr   error
}

func newMockEstimationRepo() *mockEstimationRepo {
	return &mockEstimationRepo{
		estimations: make(map[string]*domain.Estimation),
		byProject:   make(map[string][]*domain.Estimation),
	}
}

func (m *mockEstimationRepo) Create(_ context.Context, est *domain.Estimation) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.estimations[est.ID] = est
	m.byProject[est.ProjectID] = append(m.byProject[est.ProjectID], est)
	return nil
}

func (m *mockEstimationRepo) GetByID(_ context.Context, id string) (*domain.Estimation, error) {
	est, ok := m.estimations[id]
	if !ok {
		return nil, domain.ErrEstimationNotFound
	}
	return est, nil
}

func (m *mockEstimationRepo) ListByProject(_ context.Context, projectID string) ([]*domain.Estimation, error) {
	return m.byProject[projectID], nil
}

func (m *mockEstimationRepo) UpdateStatus(_ context.Context, id string, status domain.Status) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	est, ok := m.estimations[id]
	if !ok {
		return domain.ErrEstimationNotFound
	}
	est.Status = status
	if status == domain.StatusSubmitted {
		est.SubmittedAt = time.Now()
	}
	return nil
}

func (m *mockEstimationRepo) Delete(_ context.Context, id string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	est, ok := m.estimations[id]
	if !ok {
		return domain.ErrEstimationNotFound
	}
	delete(m.estimations, id)
	// Remove from byProject slice.
	pID := est.ProjectID
	filtered := make([]*domain.Estimation, 0)
	for _, e := range m.byProject[pID] {
		if e.ID != id {
			filtered = append(filtered, e)
		}
	}
	m.byProject[pID] = filtered
	return nil
}

func (m *mockEstimationRepo) seed(est *domain.Estimation) {
	m.estimations[est.ID] = est
	m.byProject[est.ProjectID] = append(m.byProject[est.ProjectID], est)
}

// --- Mock ItemRepository ---

type mockItemRepo struct {
	items     map[string][]*domain.EstimationItem // estimationID -> items
	createErr error
	deleteErr error
}

func newMockItemRepo() *mockItemRepo {
	return &mockItemRepo{items: make(map[string][]*domain.EstimationItem)}
}

func (m *mockItemRepo) CreateBatch(_ context.Context, items []*domain.EstimationItem) error {
	if m.createErr != nil {
		return m.createErr
	}
	if len(items) > 0 {
		m.items[items[0].EstimationID] = append(m.items[items[0].EstimationID], items...)
	}
	return nil
}

func (m *mockItemRepo) ListByEstimation(_ context.Context, estimationID string) ([]*domain.EstimationItem, error) {
	return m.items[estimationID], nil
}

func (m *mockItemRepo) DeleteByEstimation(_ context.Context, estimationID string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	delete(m.items, estimationID)
	return nil
}

func (m *mockItemRepo) seed(estimationID string, items []*domain.EstimationItem) {
	m.items[estimationID] = items
}

// --- Mock RoleChecker ---

type mockRoleChecker struct {
	canEstimate bool
}

func (m *mockRoleChecker) CanEstimate(_ context.Context, _, _ string) bool {
	return m.canEstimate
}

// --- Helpers ---

func newTestUsecase(estRepo domain.EstimationRepository, itemRepo domain.ItemRepository) *usecase.EstimationUsecase {
	return usecase.New(estRepo, itemRepo)
}

func newTestHandler(uc *usecase.EstimationUsecase, roleChecker handler.RoleChecker) *handler.Handler {
	return handler.New(uc, roleChecker)
}

func requestWithUserID(r *http.Request, userID string) *http.Request {
	ctx := context.WithValue(r.Context(), middleware.UserIDKey, userID)
	return r.WithContext(ctx)
}

func requestWithChiParams(r *http.Request, params map[string]string) *http.Request {
	rctx := chi.NewRouteContext()
	for k, v := range params {
		rctx.URLParams.Add(k, v)
	}
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func jsonBody(v any) *bytes.Reader {
	b, _ := json.Marshal(v)
	return bytes.NewReader(b)
}

func decodeJSON(t *testing.T, body io.Reader, v any) {
	t.Helper()
	if err := json.NewDecoder(body).Decode(v); err != nil {
		t.Fatalf("failed to decode JSON response: %v", err)
	}
}

const (
	testProjectID = "proj-001"
	testUserID    = "user-001"
	testEstID     = "est-001"
)

// ==============================
// CreateEstimation tests
// ==============================

func TestCreateEstimation_Success(t *testing.T) {
	estRepo := newMockEstimationRepo()
	itemRepo := newMockItemRepo()
	uc := newTestUsecase(estRepo, itemRepo)
	h := newTestHandler(uc, &mockRoleChecker{canEstimate: true})

	body := jsonBody(map[string]any{
		"items": []map[string]any{
			{"task_name": "Design", "min_hours": 2, "likely_hours": 4, "max_hours": 8, "sort_order": 1},
			{"task_name": "Backend", "min_hours": 4, "likely_hours": 8, "max_hours": 16, "sort_order": 2},
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+testProjectID+"/estimations", body)
	req = requestWithChiParams(req, map[string]string{"projectId": testProjectID})
	req = requestWithUserID(req, testUserID)
	rec := httptest.NewRecorder()

	h.CreateEstimation(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp domain.EstimationWithItems
	decodeJSON(t, rec.Body, &resp)

	if resp.Estimation.ProjectID != testProjectID {
		t.Errorf("expected projectID %s, got %s", testProjectID, resp.Estimation.ProjectID)
	}
	if resp.Estimation.SubmittedBy != testUserID {
		t.Errorf("expected submittedBy %s, got %s", testUserID, resp.Estimation.SubmittedBy)
	}
	if resp.Estimation.Status != domain.StatusDraft {
		t.Errorf("expected status draft, got %s", resp.Estimation.Status)
	}
	if len(resp.Items) != 2 {
		t.Errorf("expected 2 items, got %d", len(resp.Items))
	}
}

func TestCreateEstimation_EmptyItems(t *testing.T) {
	estRepo := newMockEstimationRepo()
	itemRepo := newMockItemRepo()
	uc := newTestUsecase(estRepo, itemRepo)
	h := newTestHandler(uc, &mockRoleChecker{canEstimate: true})

	body := jsonBody(map[string]any{"items": []map[string]any{}})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+testProjectID+"/estimations", body)
	req = requestWithChiParams(req, map[string]string{"projectId": testProjectID})
	req = requestWithUserID(req, testUserID)
	rec := httptest.NewRecorder()

	h.CreateEstimation(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCreateEstimation_InvalidJSON(t *testing.T) {
	estRepo := newMockEstimationRepo()
	itemRepo := newMockItemRepo()
	uc := newTestUsecase(estRepo, itemRepo)
	h := newTestHandler(uc, &mockRoleChecker{canEstimate: true})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+testProjectID+"/estimations", bytes.NewReader([]byte("not json")))
	req = requestWithChiParams(req, map[string]string{"projectId": testProjectID})
	req = requestWithUserID(req, testUserID)
	rec := httptest.NewRecorder()

	h.CreateEstimation(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCreateEstimation_MissingUserContext(t *testing.T) {
	estRepo := newMockEstimationRepo()
	itemRepo := newMockItemRepo()
	uc := newTestUsecase(estRepo, itemRepo)
	h := newTestHandler(uc, &mockRoleChecker{canEstimate: true})

	body := jsonBody(map[string]any{
		"items": []map[string]any{
			{"task_name": "Design", "min_hours": 2, "likely_hours": 4, "max_hours": 8},
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+testProjectID+"/estimations", body)
	req = requestWithChiParams(req, map[string]string{"projectId": testProjectID})
	// No user context set.
	rec := httptest.NewRecorder()

	h.CreateEstimation(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCreateEstimation_ObserverForbidden(t *testing.T) {
	estRepo := newMockEstimationRepo()
	itemRepo := newMockItemRepo()
	uc := newTestUsecase(estRepo, itemRepo)
	h := newTestHandler(uc, &mockRoleChecker{canEstimate: false})

	body := jsonBody(map[string]any{
		"items": []map[string]any{
			{"task_name": "Design", "min_hours": 2, "likely_hours": 4, "max_hours": 8},
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+testProjectID+"/estimations", body)
	req = requestWithChiParams(req, map[string]string{"projectId": testProjectID})
	req = requestWithUserID(req, testUserID)
	rec := httptest.NewRecorder()

	h.CreateEstimation(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCreateEstimation_InvalidHours(t *testing.T) {
	estRepo := newMockEstimationRepo()
	itemRepo := newMockItemRepo()
	uc := newTestUsecase(estRepo, itemRepo)
	h := newTestHandler(uc, &mockRoleChecker{canEstimate: true})

	body := jsonBody(map[string]any{
		"items": []map[string]any{
			{"task_name": "Design", "min_hours": 10, "likely_hours": 4, "max_hours": 8},
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+testProjectID+"/estimations", body)
	req = requestWithChiParams(req, map[string]string{"projectId": testProjectID})
	req = requestWithUserID(req, testUserID)
	rec := httptest.NewRecorder()

	h.CreateEstimation(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ==============================
// ListEstimations tests
// ==============================

func TestListEstimations_ReturnsList(t *testing.T) {
	estRepo := newMockEstimationRepo()
	itemRepo := newMockItemRepo()
	estRepo.seed(&domain.Estimation{
		ID: "est-1", ProjectID: testProjectID, SubmittedBy: testUserID,
		Status: domain.StatusDraft, CreatedAt: time.Now(),
	})
	estRepo.seed(&domain.Estimation{
		ID: "est-2", ProjectID: testProjectID, SubmittedBy: "user-002",
		Status: domain.StatusSubmitted, CreatedAt: time.Now(),
	})

	uc := newTestUsecase(estRepo, itemRepo)
	h := newTestHandler(uc, &mockRoleChecker{canEstimate: true})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+testProjectID+"/estimations", nil)
	req = requestWithChiParams(req, map[string]string{"projectId": testProjectID})
	req = requestWithUserID(req, testUserID)
	rec := httptest.NewRecorder()

	h.ListEstimations(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp []*domain.Estimation
	decodeJSON(t, rec.Body, &resp)
	if len(resp) != 2 {
		t.Errorf("expected 2 estimations, got %d", len(resp))
	}
}

func TestListEstimations_Empty(t *testing.T) {
	estRepo := newMockEstimationRepo()
	itemRepo := newMockItemRepo()
	uc := newTestUsecase(estRepo, itemRepo)
	h := newTestHandler(uc, &mockRoleChecker{canEstimate: true})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+testProjectID+"/estimations", nil)
	req = requestWithChiParams(req, map[string]string{"projectId": testProjectID})
	req = requestWithUserID(req, testUserID)
	rec := httptest.NewRecorder()

	h.ListEstimations(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestListEstimations_MineFilter(t *testing.T) {
	estRepo := newMockEstimationRepo()
	itemRepo := newMockItemRepo()
	estRepo.seed(&domain.Estimation{
		ID: "est-1", ProjectID: testProjectID, SubmittedBy: testUserID,
		Status: domain.StatusDraft, CreatedAt: time.Now(),
	})
	estRepo.seed(&domain.Estimation{
		ID: "est-2", ProjectID: testProjectID, SubmittedBy: "user-002",
		Status: domain.StatusSubmitted, CreatedAt: time.Now(),
	})

	uc := newTestUsecase(estRepo, itemRepo)
	h := newTestHandler(uc, &mockRoleChecker{canEstimate: true})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+testProjectID+"/estimations?mine=true", nil)
	req = requestWithChiParams(req, map[string]string{"projectId": testProjectID})
	req = requestWithUserID(req, testUserID)
	rec := httptest.NewRecorder()

	h.ListEstimations(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp []*domain.Estimation
	decodeJSON(t, rec.Body, &resp)
	if len(resp) != 1 {
		t.Errorf("expected 1 estimation (mine), got %d", len(resp))
	}
	if len(resp) > 0 && resp[0].SubmittedBy != testUserID {
		t.Errorf("expected mine filter to return user %s, got %s", testUserID, resp[0].SubmittedBy)
	}
}

// ==============================
// GetEstimation tests
// ==============================

func TestGetEstimation_Found(t *testing.T) {
	estRepo := newMockEstimationRepo()
	itemRepo := newMockItemRepo()
	estRepo.seed(&domain.Estimation{
		ID: testEstID, ProjectID: testProjectID, SubmittedBy: testUserID,
		Status: domain.StatusDraft, CreatedAt: time.Now(),
	})
	itemRepo.seed(testEstID, []*domain.EstimationItem{
		{ID: "item-1", EstimationID: testEstID, TaskName: "Design", MinHours: 2, LikelyHours: 4, MaxHours: 8},
	})

	uc := newTestUsecase(estRepo, itemRepo)
	h := newTestHandler(uc, &mockRoleChecker{canEstimate: true})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+testProjectID+"/estimations/"+testEstID, nil)
	req = requestWithChiParams(req, map[string]string{"projectId": testProjectID, "estId": testEstID})
	rec := httptest.NewRecorder()

	h.GetEstimation(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp domain.EstimationWithItems
	decodeJSON(t, rec.Body, &resp)
	if resp.Estimation.ID != testEstID {
		t.Errorf("expected estimation ID %s, got %s", testEstID, resp.Estimation.ID)
	}
	if len(resp.Items) != 1 {
		t.Errorf("expected 1 item, got %d", len(resp.Items))
	}
}

func TestGetEstimation_NotFound(t *testing.T) {
	estRepo := newMockEstimationRepo()
	itemRepo := newMockItemRepo()
	uc := newTestUsecase(estRepo, itemRepo)
	h := newTestHandler(uc, &mockRoleChecker{canEstimate: true})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+testProjectID+"/estimations/nonexistent", nil)
	req = requestWithChiParams(req, map[string]string{"projectId": testProjectID, "estId": "nonexistent"})
	rec := httptest.NewRecorder()

	h.GetEstimation(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ==============================
// SubmitEstimation tests
// ==============================

func TestSubmitEstimation_Success(t *testing.T) {
	estRepo := newMockEstimationRepo()
	itemRepo := newMockItemRepo()
	estRepo.seed(&domain.Estimation{
		ID: testEstID, ProjectID: testProjectID, SubmittedBy: testUserID,
		Status: domain.StatusDraft, CreatedAt: time.Now(),
	})

	uc := newTestUsecase(estRepo, itemRepo)
	h := newTestHandler(uc, &mockRoleChecker{canEstimate: true})

	req := httptest.NewRequest(http.MethodPut, "/api/v1/projects/"+testProjectID+"/estimations/"+testEstID+"/submit", nil)
	req = requestWithChiParams(req, map[string]string{"projectId": testProjectID, "estId": testEstID})
	req = requestWithUserID(req, testUserID)
	rec := httptest.NewRecorder()

	h.SubmitEstimation(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]string
	decodeJSON(t, rec.Body, &resp)
	if resp["status"] != "submitted" {
		t.Errorf("expected status submitted, got %s", resp["status"])
	}
}

func TestSubmitEstimation_AlreadySubmitted(t *testing.T) {
	estRepo := newMockEstimationRepo()
	itemRepo := newMockItemRepo()
	estRepo.seed(&domain.Estimation{
		ID: testEstID, ProjectID: testProjectID, SubmittedBy: testUserID,
		Status: domain.StatusSubmitted, CreatedAt: time.Now(),
	})

	uc := newTestUsecase(estRepo, itemRepo)
	h := newTestHandler(uc, &mockRoleChecker{canEstimate: true})

	req := httptest.NewRequest(http.MethodPut, "/api/v1/projects/"+testProjectID+"/estimations/"+testEstID+"/submit", nil)
	req = requestWithChiParams(req, map[string]string{"projectId": testProjectID, "estId": testEstID})
	req = requestWithUserID(req, testUserID)
	rec := httptest.NewRecorder()

	h.SubmitEstimation(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestSubmitEstimation_NotFound(t *testing.T) {
	estRepo := newMockEstimationRepo()
	itemRepo := newMockItemRepo()
	uc := newTestUsecase(estRepo, itemRepo)
	h := newTestHandler(uc, &mockRoleChecker{canEstimate: true})

	req := httptest.NewRequest(http.MethodPut, "/api/v1/projects/"+testProjectID+"/estimations/nonexistent/submit", nil)
	req = requestWithChiParams(req, map[string]string{"projectId": testProjectID, "estId": "nonexistent"})
	req = requestWithUserID(req, testUserID)
	rec := httptest.NewRecorder()

	h.SubmitEstimation(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestSubmitEstimation_MissingUserContext(t *testing.T) {
	estRepo := newMockEstimationRepo()
	itemRepo := newMockItemRepo()
	uc := newTestUsecase(estRepo, itemRepo)
	h := newTestHandler(uc, &mockRoleChecker{canEstimate: true})

	req := httptest.NewRequest(http.MethodPut, "/api/v1/projects/"+testProjectID+"/estimations/"+testEstID+"/submit", nil)
	req = requestWithChiParams(req, map[string]string{"projectId": testProjectID, "estId": testEstID})
	rec := httptest.NewRecorder()

	h.SubmitEstimation(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ==============================
// DeleteEstimation tests
// ==============================

func TestDeleteEstimation_DraftSuccess(t *testing.T) {
	estRepo := newMockEstimationRepo()
	itemRepo := newMockItemRepo()
	estRepo.seed(&domain.Estimation{
		ID: testEstID, ProjectID: testProjectID, SubmittedBy: testUserID,
		Status: domain.StatusDraft, CreatedAt: time.Now(),
	})
	itemRepo.seed(testEstID, []*domain.EstimationItem{
		{ID: "item-1", EstimationID: testEstID, TaskName: "Design"},
	})

	uc := newTestUsecase(estRepo, itemRepo)
	h := newTestHandler(uc, &mockRoleChecker{canEstimate: true})

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/projects/"+testProjectID+"/estimations/"+testEstID, nil)
	req = requestWithChiParams(req, map[string]string{"projectId": testProjectID, "estId": testEstID})
	req = requestWithUserID(req, testUserID)
	rec := httptest.NewRecorder()

	h.DeleteEstimation(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ==============================
// SetOnEvent + emit tests
// ==============================

func TestSubmitEstimation_EmitsEvent(t *testing.T) {
	estRepo := newMockEstimationRepo()
	itemRepo := newMockItemRepo()
	estRepo.seed(&domain.Estimation{
		ID: testEstID, ProjectID: testProjectID, SubmittedBy: testUserID,
		Status: domain.StatusDraft, CreatedAt: time.Now(),
	})

	uc := newTestUsecase(estRepo, itemRepo)
	h := newTestHandler(uc, &mockRoleChecker{canEstimate: true})

	var emittedEvent, emittedProject, emittedUser string
	h.SetOnEvent(func(eventType, projectID, userID string) {
		emittedEvent = eventType
		emittedProject = projectID
		emittedUser = userID
	})

	req := httptest.NewRequest(http.MethodPut, "/api/v1/projects/"+testProjectID+"/estimations/"+testEstID+"/submit", nil)
	req = requestWithChiParams(req, map[string]string{"projectId": testProjectID, "estId": testEstID})
	req = requestWithUserID(req, testUserID)
	rec := httptest.NewRecorder()

	h.SubmitEstimation(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	if emittedEvent != "estimation.submitted" {
		t.Errorf("event: got %q, want estimation.submitted", emittedEvent)
	}
	if emittedProject != testProjectID {
		t.Errorf("project: got %q, want %q", emittedProject, testProjectID)
	}
	if emittedUser != testUserID {
		t.Errorf("user: got %q, want %q", emittedUser, testUserID)
	}
}

func TestCreateEstimation_EmitsEvent(t *testing.T) {
	estRepo := newMockEstimationRepo()
	itemRepo := newMockItemRepo()
	uc := newTestUsecase(estRepo, itemRepo)

	var emittedEvent string
	h := handler.New(uc, &mockRoleChecker{canEstimate: true}, func(eventType, projectID, userID string) {
		emittedEvent = eventType
	})

	body := jsonBody(map[string]any{
		"items": []map[string]any{
			{"task_name": "Design", "min_hours": 2, "likely_hours": 4, "max_hours": 8, "sort_order": 1},
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+testProjectID+"/estimations", body)
	req = requestWithChiParams(req, map[string]string{"projectId": testProjectID})
	req = requestWithUserID(req, testUserID)
	rec := httptest.NewRecorder()

	h.CreateEstimation(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	if emittedEvent != "estimation.created" {
		t.Errorf("event: got %q, want estimation.created", emittedEvent)
	}
}

func TestDeleteEstimation_SubmittedFails(t *testing.T) {
	estRepo := newMockEstimationRepo()
	itemRepo := newMockItemRepo()
	estRepo.seed(&domain.Estimation{
		ID: testEstID, ProjectID: testProjectID, SubmittedBy: testUserID,
		Status: domain.StatusSubmitted, CreatedAt: time.Now(),
	})

	uc := newTestUsecase(estRepo, itemRepo)
	h := newTestHandler(uc, &mockRoleChecker{canEstimate: true})

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/projects/"+testProjectID+"/estimations/"+testEstID, nil)
	req = requestWithChiParams(req, map[string]string{"projectId": testProjectID, "estId": testEstID})
	req = requestWithUserID(req, testUserID)
	rec := httptest.NewRecorder()

	h.DeleteEstimation(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestDeleteEstimation_NotFound(t *testing.T) {
	estRepo := newMockEstimationRepo()
	itemRepo := newMockItemRepo()
	uc := newTestUsecase(estRepo, itemRepo)
	h := newTestHandler(uc, &mockRoleChecker{canEstimate: true})

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/projects/"+testProjectID+"/estimations/nonexistent", nil)
	req = requestWithChiParams(req, map[string]string{"projectId": testProjectID, "estId": "nonexistent"})
	req = requestWithUserID(req, testUserID)
	rec := httptest.NewRecorder()

	h.DeleteEstimation(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestDeleteEstimation_MissingUserContext(t *testing.T) {
	estRepo := newMockEstimationRepo()
	itemRepo := newMockItemRepo()
	uc := newTestUsecase(estRepo, itemRepo)
	h := newTestHandler(uc, &mockRoleChecker{canEstimate: true})

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/projects/"+testProjectID+"/estimations/"+testEstID, nil)
	req = requestWithChiParams(req, map[string]string{"projectId": testProjectID, "estId": testEstID})
	rec := httptest.NewRecorder()

	h.DeleteEstimation(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ==============================
// GetAggregated tests
// ==============================

func TestGetAggregated_Success(t *testing.T) {
	estRepo := newMockEstimationRepo()
	itemRepo := newMockItemRepo()

	est1 := &domain.Estimation{
		ID: "est-1", ProjectID: testProjectID, SubmittedBy: testUserID,
		Status: domain.StatusSubmitted, CreatedAt: time.Now(),
	}
	estRepo.seed(est1)
	itemRepo.seed("est-1", []*domain.EstimationItem{
		{ID: "item-1", EstimationID: "est-1", TaskName: "Design", MinHours: 2, LikelyHours: 4, MaxHours: 8, SortOrder: 1},
		{ID: "item-2", EstimationID: "est-1", TaskName: "Backend", MinHours: 4, LikelyHours: 8, MaxHours: 16, SortOrder: 2},
	})

	uc := newTestUsecase(estRepo, itemRepo)
	h := newTestHandler(uc, &mockRoleChecker{canEstimate: true})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+testProjectID+"/estimations/aggregated", nil)
	req = requestWithChiParams(req, map[string]string{"projectId": testProjectID})
	rec := httptest.NewRecorder()

	h.GetAggregated(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp domain.AggregatedResult
	decodeJSON(t, rec.Body, &resp)
	if len(resp.Items) != 2 {
		t.Errorf("expected 2 aggregated items, got %d", len(resp.Items))
	}
	if resp.TotalHours <= 0 {
		t.Errorf("expected positive total hours, got %f", resp.TotalHours)
	}
}

func TestGetAggregated_EmptyProject(t *testing.T) {
	estRepo := newMockEstimationRepo()
	itemRepo := newMockItemRepo()
	uc := newTestUsecase(estRepo, itemRepo)
	h := newTestHandler(uc, &mockRoleChecker{canEstimate: true})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+testProjectID+"/estimations/aggregated", nil)
	req = requestWithChiParams(req, map[string]string{"projectId": testProjectID})
	rec := httptest.NewRecorder()

	h.GetAggregated(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp domain.AggregatedResult
	decodeJSON(t, rec.Body, &resp)
	if resp.TotalHours != 0 {
		t.Errorf("expected 0 total hours for empty project, got %f", resp.TotalHours)
	}
}
