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
