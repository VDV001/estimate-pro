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
	listErr         error
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
	if f.listErr != nil {
		return nil, f.listErr
	}
	out := make([]*domain.Extraction, 0, len(f.extractions))
	for _, e := range f.extractions {
		out = append(out, e)
	}
	return out, nil
}

// Compile-time assertion: fakeRepo satisfies the production port.
var _ usecase.ExtractionRepository = (*fakeRepo)(nil)

// fakeEnqueuer is a JobEnqueuer test double that records which
// extraction IDs were enqueued so tests can assert the call was made.
type fakeEnqueuer struct {
	calls []string
	err   error
}

func (f *fakeEnqueuer) Enqueue(_ context.Context, extractionID string) error {
	if f.err != nil {
		return f.err
	}
	f.calls = append(f.calls, extractionID)
	return nil
}

// Compile-time assertion: fakeEnqueuer satisfies the production port.
var _ usecase.JobEnqueuer = (*fakeEnqueuer)(nil)

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

	uc := usecase.NewExtractor(repo, 0, nil)
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
	uc := usecase.NewExtractor(repo, 0, nil)
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

	uc := usecase.NewExtractor(repo, 0, nil)
	err := uc.CancelExtraction(t.Context(), ext.ID, "user:42")
	if !errors.Is(err, domain.ErrInvalidStatusTransition) {
		t.Fatalf("err=%v, want errors.Is %v", err, domain.ErrInvalidStatusTransition)
	}
	if len(repo.events) != 0 {
		t.Errorf("events len=%d, want 0 (no event on rejected transition)", len(repo.events))
	}
}

// ---------- RetryExtraction ----------

func TestExtractor_RetryExtraction(t *testing.T) {
	repo := newFakeRepo()
	ext := mustNewExtraction(t, "doc-1", "ver-1")
	if err := ext.MarkProcessing(); err != nil {
		t.Fatalf("MarkProcessing: %v", err)
	}
	if err := ext.MarkFailed("LLM timeout"); err != nil {
		t.Fatalf("MarkFailed: %v", err)
	}
	repo.extractions[ext.ID] = ext

	uc := usecase.NewExtractor(repo, 0, nil)
	if err := uc.RetryExtraction(t.Context(), ext.ID, "user:42"); err != nil {
		t.Fatalf("RetryExtraction: %v", err)
	}

	if ext.Status != domain.StatusPending {
		t.Errorf("Status=%q, want %q", ext.Status, domain.StatusPending)
	}
	if ext.FailureReason != "" {
		t.Errorf("FailureReason=%q, want empty after retry", ext.FailureReason)
	}
	if ext.StartedAt != nil {
		t.Errorf("StartedAt should be cleared, got %v", *ext.StartedAt)
	}
	if ext.CompletedAt != nil {
		t.Errorf("CompletedAt should be cleared, got %v", *ext.CompletedAt)
	}
	if len(repo.events) != 1 {
		t.Fatalf("events len=%d, want 1", len(repo.events))
	}
	if repo.events[0].FromStatus != domain.StatusFailed || repo.events[0].ToStatus != domain.StatusPending {
		t.Errorf("event %s→%s, want failed→pending",
			repo.events[0].FromStatus, repo.events[0].ToStatus)
	}
	if repo.events[0].Actor != "user:42" {
		t.Errorf("event.Actor=%q, want %q", repo.events[0].Actor, "user:42")
	}
}

func TestExtractor_RetryExtraction_RejectsNonFailed(t *testing.T) {
	repo := newFakeRepo()
	ext := mustNewExtraction(t, "doc-1", "ver-1")
	repo.extractions[ext.ID] = ext // pending, never failed

	uc := usecase.NewExtractor(repo, 0, nil)
	err := uc.RetryExtraction(t.Context(), ext.ID, "user:42")
	if !errors.Is(err, domain.ErrInvalidStatusTransition) {
		t.Fatalf("err=%v, want errors.Is %v", err, domain.ErrInvalidStatusTransition)
	}
}

func TestExtractor_RetryExtraction_NotFound(t *testing.T) {
	repo := newFakeRepo()
	uc := usecase.NewExtractor(repo, 0, nil)
	err := uc.RetryExtraction(t.Context(), "ghost", "user:42")
	if !errors.Is(err, domain.ErrExtractionNotFound) {
		t.Fatalf("err=%v, want errors.Is %v", err, domain.ErrExtractionNotFound)
	}
}

// ---------- GetExtraction ----------

func TestExtractor_GetExtraction(t *testing.T) {
	repo := newFakeRepo()
	ext := mustNewExtraction(t, "doc-1", "ver-1")
	if err := ext.MarkProcessing(); err != nil {
		t.Fatalf("MarkProcessing: %v", err)
	}
	repo.extractions[ext.ID] = ext

	// Two synthetic events.
	ev1, _ := domain.NewExtractionEvent(ext.ID, domain.StatusPending, domain.StatusProcessing, "", "worker")
	repo.events = append(repo.events, ev1)
	ev2, _ := domain.NewExtractionEvent(ext.ID, domain.StatusProcessing, domain.StatusFailed, "boom", "worker")
	repo.events = append(repo.events, ev2)

	uc := usecase.NewExtractor(repo, 0, nil)
	gotExt, gotEvents, err := uc.GetExtraction(t.Context(), ext.ID)
	if err != nil {
		t.Fatalf("GetExtraction: %v", err)
	}
	if gotExt.ID != ext.ID {
		t.Errorf("ext.ID=%q, want %q", gotExt.ID, ext.ID)
	}
	if len(gotEvents) != 2 {
		t.Fatalf("events len=%d, want 2", len(gotEvents))
	}
}

func TestExtractor_GetExtraction_NotFound(t *testing.T) {
	repo := newFakeRepo()
	uc := usecase.NewExtractor(repo, 0, nil)
	_, _, err := uc.GetExtraction(t.Context(), "ghost")
	if !errors.Is(err, domain.ErrExtractionNotFound) {
		t.Fatalf("err=%v, want errors.Is %v", err, domain.ErrExtractionNotFound)
	}
}

// ---------- RequestExtraction ----------

func TestExtractor_RequestExtraction_NoActive_CreatesPending(t *testing.T) {
	repo := newFakeRepo()
	uc := usecase.NewExtractor(repo, 0, nil)

	ext, err := uc.RequestExtraction(t.Context(), "doc-1", "ver-1", 1024, "user:42")
	if err != nil {
		t.Fatalf("RequestExtraction: %v", err)
	}
	if ext == nil {
		t.Fatal("expected non-nil extraction")
	}
	if ext.Status != domain.StatusPending {
		t.Errorf("Status=%q, want %q", ext.Status, domain.StatusPending)
	}
	if ext.DocumentID != "doc-1" || ext.DocumentVersionID != "ver-1" {
		t.Errorf("ids=(%q,%q), want (doc-1,ver-1)", ext.DocumentID, ext.DocumentVersionID)
	}
	if _, ok := repo.extractions[ext.ID]; !ok {
		t.Error("expected extraction to be persisted")
	}
}

func TestExtractor_RequestExtraction_ActiveExists_ReturnsExisting(t *testing.T) {
	repo := newFakeRepo()
	existing := mustNewExtraction(t, "doc-1", "ver-1")
	if err := existing.MarkProcessing(); err != nil {
		t.Fatalf("MarkProcessing: %v", err)
	}
	repo.extractions[existing.ID] = existing

	uc := usecase.NewExtractor(repo, 0, nil)
	got, err := uc.RequestExtraction(t.Context(), "doc-1", "ver-1", 1024, "user:42")
	if err != nil {
		t.Fatalf("RequestExtraction: %v", err)
	}
	if got.ID != existing.ID {
		t.Errorf("ID=%q, want existing %q", got.ID, existing.ID)
	}
	if len(repo.extractions) != 1 {
		t.Errorf("repo extractions=%d, want 1 (no new row)", len(repo.extractions))
	}
}

func TestExtractor_RequestExtraction_CompletedExists_ReturnsExisting(t *testing.T) {
	repo := newFakeRepo()
	existing := mustNewExtraction(t, "doc-1", "ver-1")
	_ = existing.MarkProcessing()
	_ = existing.MarkCompleted(nil)
	repo.extractions[existing.ID] = existing

	uc := usecase.NewExtractor(repo, 0, nil)
	got, err := uc.RequestExtraction(t.Context(), "doc-1", "ver-1", 1024, "user:42")
	if err != nil {
		t.Fatalf("RequestExtraction: %v", err)
	}
	if got.ID != existing.ID {
		t.Errorf("ID=%q, want existing completed %q", got.ID, existing.ID)
	}
}

func TestExtractor_RequestExtraction_FailedExists_CreatesNew(t *testing.T) {
	repo := newFakeRepo()
	prev := mustNewExtraction(t, "doc-1", "ver-1")
	_ = prev.MarkProcessing()
	_ = prev.MarkFailed("transient")
	repo.extractions[prev.ID] = prev

	uc := usecase.NewExtractor(repo, 0, nil)
	got, err := uc.RequestExtraction(t.Context(), "doc-1", "ver-1", 1024, "user:42")
	if err != nil {
		t.Fatalf("RequestExtraction: %v", err)
	}
	if got.ID == prev.ID {
		t.Errorf("ID=%q, expected a fresh extraction (prev was failed)", got.ID)
	}
	if got.Status != domain.StatusPending {
		t.Errorf("Status=%q, want pending", got.Status)
	}
	if len(repo.extractions) != 2 {
		t.Errorf("repo extractions=%d, want 2", len(repo.extractions))
	}
}

func TestExtractor_RequestExtraction_SizeGuard(t *testing.T) {
	repo := newFakeRepo()
	const maxBytes int64 = 1024
	uc := usecase.NewExtractor(repo, maxBytes, nil)

	_, err := uc.RequestExtraction(t.Context(), "doc-1", "ver-1", maxBytes+1, "user:42")
	if !errors.Is(err, domain.ErrDocumentTooLarge) {
		t.Fatalf("err=%v, want errors.Is %v", err, domain.ErrDocumentTooLarge)
	}
	if len(repo.extractions) != 0 {
		t.Errorf("repo extractions=%d, want 0 (oversize must short-circuit)", len(repo.extractions))
	}
}

func TestExtractor_RequestExtraction_SizeAtLimit_OK(t *testing.T) {
	repo := newFakeRepo()
	const maxBytes int64 = 1024
	uc := usecase.NewExtractor(repo, maxBytes, nil)

	if _, err := uc.RequestExtraction(t.Context(), "doc-1", "ver-1", maxBytes, "user:42"); err != nil {
		t.Fatalf("size==maxBytes should pass, got %v", err)
	}
}

func TestExtractor_RequestExtraction_RejectsEmptyIDs(t *testing.T) {
	repo := newFakeRepo()
	uc := usecase.NewExtractor(repo, 0, nil)

	cases := []struct {
		name      string
		docID     string
		versionID string
		wantErrIs error
	}{
		{name: "empty doc id", docID: "", versionID: "ver-1", wantErrIs: domain.ErrMissingDocument},
		{name: "empty version id", docID: "doc-1", versionID: "", wantErrIs: domain.ErrMissingDocumentVersion},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			_, err := uc.RequestExtraction(t.Context(), tt.docID, tt.versionID, 1024, "user:42")
			if !errors.Is(err, tt.wantErrIs) {
				t.Fatalf("err=%v, want errors.Is %v", err, tt.wantErrIs)
			}
		})
	}
}

// ---------- JobEnqueuer ----------

func TestExtractor_RequestExtraction_EnqueuesAfterCreate(t *testing.T) {
	repo := newFakeRepo()
	enqueuer := &fakeEnqueuer{}
	uc := usecase.NewExtractor(repo, 0, enqueuer)

	ext, err := uc.RequestExtraction(t.Context(), "doc-1", "ver-1", 1024, "user:42")
	if err != nil {
		t.Fatalf("RequestExtraction: %v", err)
	}
	if len(enqueuer.calls) != 1 {
		t.Fatalf("Enqueue calls=%d, want 1", len(enqueuer.calls))
	}
	if enqueuer.calls[0] != ext.ID {
		t.Errorf("Enqueue got ID=%q, want %q", enqueuer.calls[0], ext.ID)
	}
}

func TestExtractor_RequestExtraction_ActiveExists_NoEnqueue(t *testing.T) {
	repo := newFakeRepo()
	existing := mustNewExtraction(t, "doc-1", "ver-1")
	_ = existing.MarkProcessing()
	repo.extractions[existing.ID] = existing

	enqueuer := &fakeEnqueuer{}
	uc := usecase.NewExtractor(repo, 0, enqueuer)
	if _, err := uc.RequestExtraction(t.Context(), "doc-1", "ver-1", 1024, "user:42"); err != nil {
		t.Fatalf("RequestExtraction: %v", err)
	}
	if len(enqueuer.calls) != 0 {
		t.Errorf("Enqueue calls=%d, want 0 (idempotent path returns existing)", len(enqueuer.calls))
	}
}

func TestExtractor_RetryExtraction_EnqueuesAfterTransition(t *testing.T) {
	repo := newFakeRepo()
	ext := mustNewExtraction(t, "doc-1", "ver-1")
	_ = ext.MarkProcessing()
	_ = ext.MarkFailed("LLM timeout")
	repo.extractions[ext.ID] = ext

	enqueuer := &fakeEnqueuer{}
	uc := usecase.NewExtractor(repo, 0, enqueuer)
	if err := uc.RetryExtraction(t.Context(), ext.ID, "user:42"); err != nil {
		t.Fatalf("RetryExtraction: %v", err)
	}
	if len(enqueuer.calls) != 1 {
		t.Fatalf("Enqueue calls=%d, want 1", len(enqueuer.calls))
	}
	if enqueuer.calls[0] != ext.ID {
		t.Errorf("Enqueue got ID=%q, want %q", enqueuer.calls[0], ext.ID)
	}
}

// TestExtractor_ListByProject backfills coverage for the
// ListByProject pass-through — the use case is a thin forward to
// the repository, so the assertion is on the contract: empty
// repo returns an empty slice + nil error, populated repo returns
// every extraction, repo errors propagate verbatim. Fix-forward
// item #1 from PR-B2 reviewer (no per-method TDD pair existed).
func TestExtractor_ListByProject(t *testing.T) {
	t.Run("empty_repo_returns_empty_slice", func(t *testing.T) {
		repo := newFakeRepo()
		uc := usecase.NewExtractor(repo, 0, nil)
		got, err := uc.ListByProject(t.Context(), "proj-1")
		if err != nil {
			t.Fatalf("ListByProject: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("len=%d, want 0", len(got))
		}
	})

	t.Run("populated_repo_returns_all_extractions", func(t *testing.T) {
		repo := newFakeRepo()
		ext1 := mustNewExtraction(t, "doc-1", "ver-1")
		ext2 := mustNewExtraction(t, "doc-2", "ver-2")
		repo.extractions[ext1.ID] = ext1
		repo.extractions[ext2.ID] = ext2

		uc := usecase.NewExtractor(repo, 0, nil)
		got, err := uc.ListByProject(t.Context(), "proj-1")
		if err != nil {
			t.Fatalf("ListByProject: %v", err)
		}
		if len(got) != 2 {
			t.Errorf("len=%d, want 2", len(got))
		}
	})

	t.Run("repo_error_propagates", func(t *testing.T) {
		repo := newFakeRepo()
		boom := errors.New("db down")
		repo.listErr = boom

		uc := usecase.NewExtractor(repo, 0, nil)
		_, err := uc.ListByProject(t.Context(), "proj-1")
		if !errors.Is(err, boom) {
			t.Fatalf("err=%v, want errors.Is %v", err, boom)
		}
	})
}
