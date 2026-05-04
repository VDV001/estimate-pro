// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package worker_test

import (
	"context"
	"errors"
	"testing"

	"github.com/VDV001/estimate-pro/backend/internal/modules/extractor/domain"
	"github.com/VDV001/estimate-pro/backend/internal/modules/extractor/worker"
	sharedllm "github.com/VDV001/estimate-pro/backend/internal/shared/llm"
)

// fakeStore implements worker.ExtractionStore for unit tests; only
// the methods exercised by the test under question return values,
// the rest panic to surface unexpected interactions loudly.
type fakeStore struct {
	getErr        error
	updateErr     error
	saveErr       error
	got           *domain.Extraction
	updateCalls   []*domain.Extraction
	updateEvents  []*domain.ExtractionEvent
	savedID       string
	savedTasks    []domain.ExtractedTask
	saveCallCount int
}

func (f *fakeStore) GetByID(_ context.Context, _ string) (*domain.Extraction, error) {
	return f.got, f.getErr
}

func (f *fakeStore) UpdateStatus(_ context.Context, ext *domain.Extraction, ev *domain.ExtractionEvent) error {
	f.updateCalls = append(f.updateCalls, ext)
	f.updateEvents = append(f.updateEvents, ev)
	return f.updateErr
}

func (f *fakeStore) SaveTasks(_ context.Context, id string, tasks []domain.ExtractedTask) error {
	f.savedID = id
	f.savedTasks = tasks
	f.saveCallCount++
	return f.saveErr
}

type panickingSource struct{}

func (panickingSource) Fetch(_ context.Context, _ string) ([]byte, string, error) {
	panic("worker: DocumentSource.Fetch not expected to be called in this test")
}

type panickingReader struct{}

func (panickingReader) Parse(_ context.Context, _ string, _ []byte) (string, error) {
	panic("worker: TextExtractor.Parse not expected to be called in this test")
}

type panickingCompleter struct{}

func (panickingCompleter) Complete(_ context.Context, _, _ string, _ sharedllm.CompletionOptions) (string, sharedllm.TokenUsage, error) {
	panic("worker: Completer.Complete not expected to be called in this test")
}

type panickingSecurity struct{}

func (panickingSecurity) IsPromptInjection(_ string) bool {
	panic("worker: SecurityChecker.IsPromptInjection not expected to be called in this test")
}

// TestProcess_ExtractionNotFound_ReturnsWrappedError pins the
// shortest worker error path: when the store cannot find the
// extraction, Process surfaces domain.ErrExtractionNotFound via
// errors.Is so the river job runner can decide retry vs fail-fast.
// No subsequent calls (download / parse / LLM / save) happen — the
// panicking fakes assert that.
func TestProcess_ExtractionNotFound_ReturnsWrappedError(t *testing.T) {
	store := &fakeStore{getErr: domain.ErrExtractionNotFound}
	w := worker.NewExtractionWorker(store, panickingSource{}, panickingReader{}, panickingCompleter{}, panickingSecurity{})

	err := w.Process(context.Background(), worker.ExtractionArgs{ExtractionID: "missing"})
	if !errors.Is(err, domain.ErrExtractionNotFound) {
		t.Fatalf("expected ErrExtractionNotFound, got %v", err)
	}
	if len(store.updateCalls) != 0 {
		t.Fatalf("expected no UpdateStatus calls when extraction missing, got %d", len(store.updateCalls))
	}
	if store.saveCallCount != 0 {
		t.Fatalf("expected no SaveTasks calls when extraction missing, got %d", store.saveCallCount)
	}
}

// pendingExtraction returns a freshly-constructed pending Extraction
// pointing at a stubbed document version. Tests that need to drive
// the worker through real domain transitions use this helper rather
// than poking struct fields directly — keeps the constructor's
// invariants (non-empty doc/version IDs) honored.
func pendingExtraction(t *testing.T) *domain.Extraction {
	t.Helper()
	ext, err := domain.NewExtraction("doc-1", "ver-1")
	if err != nil {
		t.Fatalf("pendingExtraction: %v", err)
	}
	return ext
}

// TestProcess_StatusNotPending_SkipsIdempotently pins the
// re-enqueue safety property: river may dispatch the same
// ExtractionArgs more than once if a worker crashes mid-run; the
// idempotency guarantee is that a second job whose extraction is
// already past pending observes the current state and returns nil
// without re-running any side effect. The panicking fakes prove
// no UpdateStatus / Fetch / Parse / LLM call happens.
func TestProcess_StatusNotPending_SkipsIdempotently(t *testing.T) {
	cases := []struct {
		name        string
		mutate      func(*domain.Extraction)
		wantSkipped bool
	}{
		{
			name:        "processing",
			mutate:      func(e *domain.Extraction) { _ = e.MarkProcessing() },
			wantSkipped: true,
		},
		{
			name: "completed",
			mutate: func(e *domain.Extraction) {
				if err := e.MarkProcessing(); err != nil {
					t.Fatalf("driveTo processing: %v", err)
				}
				if err := e.MarkCompleted(nil); err != nil {
					t.Fatalf("driveTo completed: %v", err)
				}
			},
			wantSkipped: true,
		},
		{
			name: "failed",
			mutate: func(e *domain.Extraction) {
				if err := e.MarkProcessing(); err != nil {
					t.Fatalf("driveTo processing: %v", err)
				}
				if err := e.MarkFailed("upstream LLM down"); err != nil {
					t.Fatalf("driveTo failed: %v", err)
				}
			},
			wantSkipped: true,
		},
		{
			name: "cancelled",
			mutate: func(e *domain.Extraction) {
				if err := e.MarkCancelled(); err != nil {
					t.Fatalf("driveTo cancelled: %v", err)
				}
			},
			wantSkipped: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ext := pendingExtraction(t)
			tc.mutate(ext)

			store := &fakeStore{got: ext}
			w := worker.NewExtractionWorker(store, panickingSource{}, panickingReader{}, panickingCompleter{}, panickingSecurity{})

			if err := w.Process(context.Background(), worker.ExtractionArgs{ExtractionID: ext.ID}); err != nil {
				t.Fatalf("expected nil error for non-pending skip, got %v", err)
			}
			if len(store.updateCalls) != 0 {
				t.Fatalf("expected no UpdateStatus calls when skipping non-pending status, got %d", len(store.updateCalls))
			}
		})
	}
}

// TestProcess_PendingTransitionsToProcessing locks in the first
// state-machine step on the happy path: a pending extraction
// transitions to processing via Extraction.MarkProcessing and the
// transition is recorded by exactly one UpdateStatus call carrying
// an audit ExtractionEvent (actor "worker", from pending, to
// processing). Subsequent pipeline stages (download / parse / LLM
// / save) are not yet implemented — the panicking fakes prove the
// transition lands without leaking into stages that are still
// deferred. Subsequent RED+GREEN pairs replace those panickers
// with real fakes as each stage ships.
func TestProcess_PendingTransitionsToProcessing(t *testing.T) {
	ext := pendingExtraction(t)
	store := &fakeStore{got: ext}

	w := worker.NewExtractionWorker(store, panickingSource{}, panickingReader{}, panickingCompleter{}, panickingSecurity{})

	// Process must not return an error after the transition lands;
	// later RED pairs will tighten this to "and Fetch was called".
	if err := w.Process(context.Background(), worker.ExtractionArgs{ExtractionID: ext.ID}); err != nil {
		t.Fatalf("expected nil error after pending->processing transition, got %v", err)
	}

	if len(store.updateCalls) != 1 {
		t.Fatalf("expected exactly one UpdateStatus call (the pending->processing transition), got %d", len(store.updateCalls))
	}
	gotExt := store.updateCalls[0]
	if gotExt.Status != domain.StatusProcessing {
		t.Fatalf("expected ext.Status=processing after transition, got %s", gotExt.Status)
	}
	gotEvent := store.updateEvents[0]
	if gotEvent.FromStatus != domain.StatusPending {
		t.Fatalf("expected event.FromStatus=pending, got %s", gotEvent.FromStatus)
	}
	if gotEvent.ToStatus != domain.StatusProcessing {
		t.Fatalf("expected event.ToStatus=processing, got %s", gotEvent.ToStatus)
	}
	if gotEvent.Actor != "worker" {
		t.Fatalf("expected event.Actor=worker, got %q", gotEvent.Actor)
	}
	if gotEvent.ExtractionID != ext.ID {
		t.Fatalf("expected event.ExtractionID=%q, got %q", ext.ID, gotEvent.ExtractionID)
	}
}
