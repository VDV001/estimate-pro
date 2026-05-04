// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/VDV001/estimate-pro/backend/internal/modules/extractor/domain"
	"github.com/VDV001/estimate-pro/backend/internal/modules/extractor/usecase"
)

// fakeRepo is an in-memory ExtractionRepository for use-case unit
// tests. The postgres-backed implementation is exercised separately
// by repository_integration_test.go; here we only verify the
// orchestration logic.
type fakeRepo struct {
	extractions map[string]*domain.Extraction
	events      []*domain.ExtractionEvent

	createErr       error
	getByIDErr      error
	updateStatusErr error
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{extractions: make(map[string]*domain.Extraction)}
}

func (f *fakeRepo) Create(_ context.Context, ext *domain.Extraction) error {
	if f.createErr != nil {
		return f.createErr
	}
	f.extractions[ext.ID] = ext
	return nil
}

func (f *fakeRepo) GetByID(_ context.Context, id string) (*domain.Extraction, error) {
	if f.getByIDErr != nil {
		return nil, f.getByIDErr
	}
	e, ok := f.extractions[id]
	if !ok {
		return nil, domain.ErrExtractionNotFound
	}
	return e, nil
}

func (f *fakeRepo) GetActiveByDocumentVersion(_ context.Context, docID, versionID string) (*domain.Extraction, error) {
	for _, e := range f.extractions {
		if e.DocumentID != docID || e.DocumentVersionID != versionID {
			continue
		}
		switch e.Status {
		case domain.StatusPending, domain.StatusProcessing, domain.StatusCompleted:
			return e, nil
		}
	}
	return nil, domain.ErrExtractionNotFound
}

func (f *fakeRepo) UpdateStatus(_ context.Context, ext *domain.Extraction, ev *domain.ExtractionEvent) error {
	if f.updateStatusErr != nil {
		return f.updateStatusErr
	}
	if _, ok := f.extractions[ext.ID]; !ok {
		return domain.ErrExtractionNotFound
	}
	f.extractions[ext.ID] = ext
	f.events = append(f.events, ev)
	return nil
}

func (f *fakeRepo) SaveTasks(_ context.Context, id string, tasks []domain.ExtractedTask) error {
	e, ok := f.extractions[id]
	if !ok {
		return domain.ErrExtractionNotFound
	}
	e.Tasks = tasks
	return nil
}

func (f *fakeRepo) GetEvents(_ context.Context, id string) ([]*domain.ExtractionEvent, error) {
	var out []*domain.ExtractionEvent
	for _, ev := range f.events {
		if ev.ExtractionID == id {
			out = append(out, ev)
		}
	}
	return out, nil
}

func (f *fakeRepo) ListByProject(_ context.Context, _ string) ([]*domain.Extraction, error) {
	return nil, nil
}

// Compile-time assertion: fakeRepo satisfies the production port.
var _ usecase.ExtractionRepository = (*fakeRepo)(nil)

// helper

func mustNewExtraction(t *testing.T, docID, versionID string) *domain.Extraction {
	t.Helper()
	ext, err := domain.NewExtraction(docID, versionID)
	if err != nil {
		t.Fatalf("NewExtraction: %v", err)
	}
	return ext
}

// ---------- CancelExtraction ----------

func TestExtractor_CancelExtraction(t *testing.T) {
	repo := newFakeRepo()
	ext := mustNewExtraction(t, "doc-1", "ver-1")
	repo.extractions[ext.ID] = ext

	uc := usecase.NewExtractor(repo)
	err := uc.CancelExtraction(t.Context(), ext.ID, "user:42")
	if err != nil {
		t.Fatalf("CancelExtraction: %v", err)
	}

	if ext.Status != domain.StatusCancelled {
		t.Errorf("Status=%q, want %q", ext.Status, domain.StatusCancelled)
	}
	if len(repo.events) != 1 {
		t.Fatalf("events len=%d, want 1", len(repo.events))
	}
	ev := repo.events[0]
	if ev.FromStatus != domain.StatusPending {
		t.Errorf("event.From=%q, want %q", ev.FromStatus, domain.StatusPending)
	}
	if ev.ToStatus != domain.StatusCancelled {
		t.Errorf("event.To=%q, want %q", ev.ToStatus, domain.StatusCancelled)
	}
	if ev.Actor != "user:42" {
		t.Errorf("event.Actor=%q, want %q", ev.Actor, "user:42")
	}
}

func TestExtractor_CancelExtraction_NotFound(t *testing.T) {
	repo := newFakeRepo()
	uc := usecase.NewExtractor(repo)
	err := uc.CancelExtraction(t.Context(), "ghost-id", "user:42")
	if !errors.Is(err, domain.ErrExtractionNotFound) {
		t.Fatalf("err=%v, want errors.Is %v", err, domain.ErrExtractionNotFound)
	}
}

func TestExtractor_CancelExtraction_AlreadyTerminal(t *testing.T) {
	repo := newFakeRepo()
	ext := mustNewExtraction(t, "doc-1", "ver-1")
	if err := ext.MarkProcessing(); err != nil {
		t.Fatalf("MarkProcessing: %v", err)
	}
	if err := ext.MarkCompleted(nil); err != nil {
		t.Fatalf("MarkCompleted: %v", err)
	}
	repo.extractions[ext.ID] = ext

	uc := usecase.NewExtractor(repo)
	err := uc.CancelExtraction(t.Context(), ext.ID, "user:42")
	if !errors.Is(err, domain.ErrInvalidStatusTransition) {
		t.Fatalf("err=%v, want errors.Is %v", err, domain.ErrInvalidStatusTransition)
	}
	if len(repo.events) != 0 {
		t.Errorf("events len=%d, want 0 (no event on rejected transition)", len(repo.events))
	}
}
