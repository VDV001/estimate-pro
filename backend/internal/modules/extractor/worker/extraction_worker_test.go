// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package worker

import (
	"context"
	"errors"
	"testing"
)

// TestExtractionWorker_ProcessReturnsNotImplementedSentinel pins the
// PR-B2 contract: the worker package exists as a placeholder
// scaffold whose Process method must surface ErrWorkerNotImplemented
// so callers (river runner, manual triggers in tests) can detect
// the deferral via errors.Is. The real body — MinIO download, reader
// dispatch, prompt-injection check, LLM call, schema validation,
// SaveTasks + UpdateStatus — ships in PR-B3 (issue #7), at which
// point this test is replaced with success / failure scenarios
// driven through testcontainers and the sentinel is removed.
func TestExtractionWorker_ProcessReturnsNotImplementedSentinel(t *testing.T) {
	w := NewExtractionWorker()
	err := w.Process(context.Background(), ExtractionArgs{ExtractionID: "any-id"})
	if !errors.Is(err, ErrWorkerNotImplemented) {
		t.Fatalf("expected ErrWorkerNotImplemented, got %v", err)
	}
}
