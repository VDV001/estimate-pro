package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/VDV001/estimate-pro/backend/internal/modules/estimation/domain"
)

// --- Mock repositories ---

type mockEstimationRepo struct {
	estimations map[string]*domain.Estimation
	byProject   map[string][]*domain.Estimation
	createErr   error
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
	est, ok := m.estimations[id]
	if !ok {
		return domain.ErrEstimationNotFound
	}
	est.Status = status
	return nil
}

func (m *mockEstimationRepo) Delete(_ context.Context, id string) error {
	est, ok := m.estimations[id]
	if !ok {
		return domain.ErrEstimationNotFound
	}
	// Remove from byProject.
	projectID := est.ProjectID
	filtered := make([]*domain.Estimation, 0)
	for _, e := range m.byProject[projectID] {
		if e.ID != id {
			filtered = append(filtered, e)
		}
	}
	m.byProject[projectID] = filtered
	delete(m.estimations, id)
	return nil
}

type mockItemRepo struct {
	items     map[string][]*domain.EstimationItem
	createErr error
}

func newMockItemRepo() *mockItemRepo {
	return &mockItemRepo{items: make(map[string][]*domain.EstimationItem)}
}

func (m *mockItemRepo) CreateBatch(_ context.Context, items []*domain.EstimationItem) error {
	if m.createErr != nil {
		return m.createErr
	}
	if len(items) > 0 {
		estID := items[0].EstimationID
		m.items[estID] = append(m.items[estID], items...)
	}
	return nil
}

func (m *mockItemRepo) ListByEstimation(_ context.Context, estimationID string) ([]*domain.EstimationItem, error) {
	return m.items[estimationID], nil
}

func (m *mockItemRepo) DeleteByEstimation(_ context.Context, estimationID string) error {
	delete(m.items, estimationID)
	return nil
}

// --- Tests ---

func TestCreate_Success(t *testing.T) {
	estRepo := newMockEstimationRepo()
	itemRepo := newMockItemRepo()
	uc := New(estRepo, itemRepo)

	result, err := uc.Create(t.Context(), CreateInput{
		ProjectID: "proj-1",
		UserID:    "user-1",
		Items: []CreateItemInput{
			{TaskName: "Backend API", MinHours: 4, LikelyHours: 8, MaxHours: 16},
			{TaskName: "Frontend UI", MinHours: 2, LikelyHours: 4, MaxHours: 8},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Estimation.Status != domain.StatusDraft {
		t.Errorf("status = %v, want draft", result.Estimation.Status)
	}
	if result.Estimation.ProjectID != "proj-1" {
		t.Errorf("project_id = %v, want proj-1", result.Estimation.ProjectID)
	}
	if len(result.Items) != 2 {
		t.Errorf("items count = %d, want 2", len(result.Items))
	}
	// Verify IDs were assigned.
	for i, item := range result.Items {
		if item.ID == "" {
			t.Errorf("item[%d].ID is empty", i)
		}
		if item.EstimationID != result.Estimation.ID {
			t.Errorf("item[%d].EstimationID = %v, want %v", i, item.EstimationID, result.Estimation.ID)
		}
	}
}

func TestCreate_EmptyItems(t *testing.T) {
	uc := New(newMockEstimationRepo(), newMockItemRepo())

	_, err := uc.Create(t.Context(), CreateInput{
		ProjectID: "proj-1",
		UserID:    "user-1",
		Items:     nil,
	})
	if !errors.Is(err, domain.ErrEmptyItems) {
		t.Errorf("expected ErrEmptyItems, got %v", err)
	}
}

func TestCreate_InvalidHours(t *testing.T) {
	uc := New(newMockEstimationRepo(), newMockItemRepo())

	tests := []struct {
		name  string
		items []CreateItemInput
	}{
		{
			name:  "negative min",
			items: []CreateItemInput{{TaskName: "T", MinHours: -1, LikelyHours: 4, MaxHours: 8}},
		},
		{
			name:  "min > likely",
			items: []CreateItemInput{{TaskName: "T", MinHours: 10, LikelyHours: 4, MaxHours: 8}},
		},
		{
			name:  "likely > max",
			items: []CreateItemInput{{TaskName: "T", MinHours: 2, LikelyHours: 10, MaxHours: 8}},
		},
		{
			name:  "empty task name",
			items: []CreateItemInput{{TaskName: "", MinHours: 2, LikelyHours: 4, MaxHours: 8}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := uc.Create(t.Context(), CreateInput{
				ProjectID: "proj-1",
				UserID:    "user-1",
				Items:     tt.items,
			})
			if err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

func TestSubmit_Success(t *testing.T) {
	estRepo := newMockEstimationRepo()
	itemRepo := newMockItemRepo()
	uc := New(estRepo, itemRepo)

	result, _ := uc.Create(t.Context(), CreateInput{
		ProjectID: "proj-1",
		UserID:    "user-1",
		Items:     []CreateItemInput{{TaskName: "T", MinHours: 2, LikelyHours: 4, MaxHours: 8}},
	})

	err := uc.Submit(t.Context(), result.Estimation.ID, "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	est, _ := estRepo.GetByID(t.Context(), result.Estimation.ID)
	if est.Status != domain.StatusSubmitted {
		t.Errorf("status = %v, want submitted", est.Status)
	}
}

func TestSubmit_AlreadySubmitted(t *testing.T) {
	estRepo := newMockEstimationRepo()
	itemRepo := newMockItemRepo()
	uc := New(estRepo, itemRepo)

	result, _ := uc.Create(t.Context(), CreateInput{
		ProjectID: "proj-1",
		UserID:    "user-1",
		Items:     []CreateItemInput{{TaskName: "T", MinHours: 2, LikelyHours: 4, MaxHours: 8}},
	})

	_ = uc.Submit(t.Context(), result.Estimation.ID, "user-1")
	err := uc.Submit(t.Context(), result.Estimation.ID, "user-1")

	if !errors.Is(err, domain.ErrAlreadySubmitted) {
		t.Errorf("expected ErrAlreadySubmitted, got %v", err)
	}
}

func TestSubmit_WrongUser(t *testing.T) {
	estRepo := newMockEstimationRepo()
	itemRepo := newMockItemRepo()
	uc := New(estRepo, itemRepo)

	result, _ := uc.Create(t.Context(), CreateInput{
		ProjectID: "proj-1",
		UserID:    "user-1",
		Items:     []CreateItemInput{{TaskName: "T", MinHours: 2, LikelyHours: 4, MaxHours: 8}},
	})

	err := uc.Submit(t.Context(), result.Estimation.ID, "user-2")
	if err == nil {
		t.Error("expected error for wrong user, got nil")
	}
}

func TestDelete_Success(t *testing.T) {
	estRepo := newMockEstimationRepo()
	itemRepo := newMockItemRepo()
	uc := New(estRepo, itemRepo)

	result, _ := uc.Create(t.Context(), CreateInput{
		ProjectID: "proj-1",
		UserID:    "user-1",
		Items:     []CreateItemInput{{TaskName: "T", MinHours: 2, LikelyHours: 4, MaxHours: 8}},
	})

	err := uc.Delete(t.Context(), result.Estimation.ID, "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = estRepo.GetByID(t.Context(), result.Estimation.ID)
	if !errors.Is(err, domain.ErrEstimationNotFound) {
		t.Errorf("expected ErrEstimationNotFound after delete, got %v", err)
	}
}

func TestDelete_CannotDeleteSubmitted(t *testing.T) {
	estRepo := newMockEstimationRepo()
	itemRepo := newMockItemRepo()
	uc := New(estRepo, itemRepo)

	result, _ := uc.Create(t.Context(), CreateInput{
		ProjectID: "proj-1",
		UserID:    "user-1",
		Items:     []CreateItemInput{{TaskName: "T", MinHours: 2, LikelyHours: 4, MaxHours: 8}},
	})

	_ = uc.Submit(t.Context(), result.Estimation.ID, "user-1")
	err := uc.Delete(t.Context(), result.Estimation.ID, "user-1")

	if !errors.Is(err, domain.ErrNotDraft) {
		t.Errorf("expected ErrNotDraft, got %v", err)
	}
}

func TestGetAggregated(t *testing.T) {
	estRepo := newMockEstimationRepo()
	itemRepo := newMockItemRepo()
	uc := New(estRepo, itemRepo)

	// Create and submit two estimations.
	r1, _ := uc.Create(t.Context(), CreateInput{
		ProjectID: "proj-1",
		UserID:    "user-1",
		Items: []CreateItemInput{
			{TaskName: "Task A", MinHours: 2, LikelyHours: 4, MaxHours: 8},
		},
	})
	_ = uc.Submit(t.Context(), r1.Estimation.ID, "user-1")

	r2, _ := uc.Create(t.Context(), CreateInput{
		ProjectID: "proj-1",
		UserID:    "user-2",
		Items: []CreateItemInput{
			{TaskName: "Task A", MinHours: 4, LikelyHours: 6, MaxHours: 12},
		},
	})
	_ = uc.Submit(t.Context(), r2.Estimation.ID, "user-2")

	// Create a draft — should be excluded.
	_, _ = uc.Create(t.Context(), CreateInput{
		ProjectID: "proj-1",
		UserID:    "user-3",
		Items: []CreateItemInput{
			{TaskName: "Task A", MinHours: 100, LikelyHours: 200, MaxHours: 300},
		},
	})

	result, err := uc.GetAggregated(t.Context(), "proj-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Items) != 1 {
		t.Fatalf("expected 1 aggregated item, got %d", len(result.Items))
	}

	item := result.Items[0]
	if item.TaskName != "Task A" {
		t.Errorf("task name = %q, want Task A", item.TaskName)
	}
	if item.EstimatorCount != 2 {
		t.Errorf("estimator count = %d, want 2", item.EstimatorCount)
	}
}

func TestListByProject(t *testing.T) {
	estRepo := newMockEstimationRepo()
	itemRepo := newMockItemRepo()
	uc := New(estRepo, itemRepo)

	// Create two estimations for proj-1 and one for proj-2.
	_, _ = uc.Create(t.Context(), CreateInput{
		ProjectID: "proj-1", UserID: "user-1",
		Items: []CreateItemInput{{TaskName: "T1", MinHours: 1, LikelyHours: 2, MaxHours: 4}},
	})
	_, _ = uc.Create(t.Context(), CreateInput{
		ProjectID: "proj-1", UserID: "user-2",
		Items: []CreateItemInput{{TaskName: "T2", MinHours: 1, LikelyHours: 2, MaxHours: 4}},
	})
	_, _ = uc.Create(t.Context(), CreateInput{
		ProjectID: "proj-2", UserID: "user-1",
		Items: []CreateItemInput{{TaskName: "T3", MinHours: 1, LikelyHours: 2, MaxHours: 4}},
	})

	t.Run("returns estimations for project", func(t *testing.T) {
		list, err := uc.ListByProject(t.Context(), "proj-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(list) != 2 {
			t.Errorf("expected 2 estimations, got %d", len(list))
		}
	})

	t.Run("empty project returns empty list", func(t *testing.T) {
		list, err := uc.ListByProject(t.Context(), "proj-999")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(list) != 0 {
			t.Errorf("expected 0 estimations, got %d", len(list))
		}
	})
}

func TestSubmit_NotFound(t *testing.T) {
	uc := New(newMockEstimationRepo(), newMockItemRepo())
	err := uc.Submit(t.Context(), "nonexistent", "user-1")
	if !errors.Is(err, domain.ErrEstimationNotFound) {
		t.Errorf("expected ErrEstimationNotFound, got %v", err)
	}
}

func TestGetByID_NotFound(t *testing.T) {
	uc := New(newMockEstimationRepo(), newMockItemRepo())

	_, err := uc.GetByID(t.Context(), "nonexistent")
	if !errors.Is(err, domain.ErrEstimationNotFound) {
		t.Errorf("expected ErrEstimationNotFound, got %v", err)
	}
}
