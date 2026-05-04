// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/VDV001/estimate-pro/backend/internal/modules/extractor/domain"
)

// stateMachineCase exercises one transition method against every
// possible starting status. wantErrIs == nil means the transition
// must succeed and arrive at wantStatus.
type stateMachineCase struct {
	name        string
	from        domain.ExtractionStatus
	wantErrIs   error
	wantStatus  domain.ExtractionStatus
}

// driveTo advances a freshly-built Extraction through valid
// transitions until its status matches target. Tests rely on the
// fact that every status is reachable from pending via at most three
// hops.
func driveTo(t *testing.T, target domain.ExtractionStatus) *domain.Extraction {
	t.Helper()
	ext, err := domain.NewExtraction("doc", "ver")
	if err != nil {
		t.Fatalf("NewExtraction: %v", err)
	}
	switch target {
	case domain.StatusPending:
		return ext
	case domain.StatusProcessing:
		mustOK(t, ext.MarkProcessing())
		return ext
	case domain.StatusCompleted:
		mustOK(t, ext.MarkProcessing())
		mustOK(t, ext.MarkCompleted([]domain.ExtractedTask{}))
		return ext
	case domain.StatusFailed:
		mustOK(t, ext.MarkProcessing())
		mustOK(t, ext.MarkFailed("synthetic"))
		return ext
	case domain.StatusCancelled:
		mustOK(t, ext.MarkCancelled())
		return ext
	}
	t.Fatalf("driveTo: unsupported target %q", target)
	return nil
}

func mustOK(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("transition: %v", err)
	}
}

func TestExtraction_MarkProcessing(t *testing.T) {
	cases := []stateMachineCase{
		{name: "pending → processing", from: domain.StatusPending, wantStatus: domain.StatusProcessing},
		{name: "processing rejected", from: domain.StatusProcessing, wantErrIs: domain.ErrInvalidStatusTransition},
		{name: "completed rejected", from: domain.StatusCompleted, wantErrIs: domain.ErrInvalidStatusTransition},
		{name: "failed rejected", from: domain.StatusFailed, wantErrIs: domain.ErrInvalidStatusTransition},
		{name: "cancelled rejected", from: domain.StatusCancelled, wantErrIs: domain.ErrInvalidStatusTransition},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ext := driveTo(t, c.from)
			before := time.Now()
			err := ext.MarkProcessing()
			after := time.Now()

			if c.wantErrIs != nil {
				if !errors.Is(err, c.wantErrIs) {
					t.Fatalf("err=%v, want errors.Is %v", err, c.wantErrIs)
				}
				if ext.Status != c.from {
					t.Errorf("status mutated on rejected transition: %q", ext.Status)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if ext.Status != c.wantStatus {
				t.Errorf("Status=%q, want %q", ext.Status, c.wantStatus)
			}
			if ext.StartedAt == nil {
				t.Fatal("StartedAt should be set")
			}
			if ext.StartedAt.Before(before) || ext.StartedAt.After(after) {
				t.Errorf("StartedAt=%v outside window [%v, %v]", *ext.StartedAt, before, after)
			}
			if !ext.UpdatedAt.Equal(*ext.StartedAt) {
				t.Errorf("UpdatedAt=%v, want equal StartedAt=%v", ext.UpdatedAt, *ext.StartedAt)
			}
		})
	}
}

func TestExtraction_MarkCompleted(t *testing.T) {
	cases := []stateMachineCase{
		{name: "processing → completed", from: domain.StatusProcessing, wantStatus: domain.StatusCompleted},
		{name: "pending rejected", from: domain.StatusPending, wantErrIs: domain.ErrInvalidStatusTransition},
		{name: "completed → ErrAlreadyCompleted", from: domain.StatusCompleted, wantErrIs: domain.ErrAlreadyCompleted},
		{name: "failed rejected", from: domain.StatusFailed, wantErrIs: domain.ErrInvalidStatusTransition},
		{name: "cancelled rejected", from: domain.StatusCancelled, wantErrIs: domain.ErrInvalidStatusTransition},
	}
	tasks := []domain.ExtractedTask{
		mustNewTask(t, "T1", "1h"),
		mustNewTask(t, "T2", "2h"),
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ext := driveTo(t, c.from)
			err := ext.MarkCompleted(tasks)

			if c.wantErrIs != nil {
				if !errors.Is(err, c.wantErrIs) {
					t.Fatalf("err=%v, want errors.Is %v", err, c.wantErrIs)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if ext.Status != c.wantStatus {
				t.Errorf("Status=%q, want %q", ext.Status, c.wantStatus)
			}
			if len(ext.Tasks) != len(tasks) {
				t.Errorf("Tasks len=%d, want %d", len(ext.Tasks), len(tasks))
			}
			if ext.CompletedAt == nil {
				t.Error("CompletedAt should be set")
			}
		})
	}
}

func TestExtraction_MarkFailed(t *testing.T) {
	cases := []stateMachineCase{
		{name: "processing → failed", from: domain.StatusProcessing, wantStatus: domain.StatusFailed},
		{name: "pending rejected", from: domain.StatusPending, wantErrIs: domain.ErrInvalidStatusTransition},
		{name: "completed rejected", from: domain.StatusCompleted, wantErrIs: domain.ErrInvalidStatusTransition},
		{name: "failed rejected (no double-fail)", from: domain.StatusFailed, wantErrIs: domain.ErrInvalidStatusTransition},
		{name: "cancelled rejected", from: domain.StatusCancelled, wantErrIs: domain.ErrInvalidStatusTransition},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ext := driveTo(t, c.from)
			err := ext.MarkFailed("LLM timed out")

			if c.wantErrIs != nil {
				if !errors.Is(err, c.wantErrIs) {
					t.Fatalf("err=%v, want errors.Is %v", err, c.wantErrIs)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if ext.Status != c.wantStatus {
				t.Errorf("Status=%q, want %q", ext.Status, c.wantStatus)
			}
			if ext.FailureReason != "LLM timed out" {
				t.Errorf("FailureReason=%q, want %q", ext.FailureReason, "LLM timed out")
			}
			if ext.CompletedAt == nil {
				t.Error("CompletedAt should be set")
			}
		})
	}
}

func TestExtraction_MarkCancelled(t *testing.T) {
	cases := []stateMachineCase{
		{name: "pending → cancelled", from: domain.StatusPending, wantStatus: domain.StatusCancelled},
		{name: "processing → cancelled", from: domain.StatusProcessing, wantStatus: domain.StatusCancelled},
		{name: "completed rejected", from: domain.StatusCompleted, wantErrIs: domain.ErrInvalidStatusTransition},
		{name: "failed rejected", from: domain.StatusFailed, wantErrIs: domain.ErrInvalidStatusTransition},
		{name: "cancelled rejected (idempotent denied)", from: domain.StatusCancelled, wantErrIs: domain.ErrInvalidStatusTransition},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ext := driveTo(t, c.from)
			err := ext.MarkCancelled()

			if c.wantErrIs != nil {
				if !errors.Is(err, c.wantErrIs) {
					t.Fatalf("err=%v, want errors.Is %v", err, c.wantErrIs)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if ext.Status != c.wantStatus {
				t.Errorf("Status=%q, want %q", ext.Status, c.wantStatus)
			}
			if ext.CompletedAt == nil {
				t.Error("CompletedAt should be set")
			}
		})
	}
}

func TestExtraction_MarkRetry(t *testing.T) {
	cases := []stateMachineCase{
		{name: "failed → pending", from: domain.StatusFailed, wantStatus: domain.StatusPending},
		{name: "pending rejected", from: domain.StatusPending, wantErrIs: domain.ErrInvalidStatusTransition},
		{name: "processing rejected", from: domain.StatusProcessing, wantErrIs: domain.ErrInvalidStatusTransition},
		{name: "completed rejected", from: domain.StatusCompleted, wantErrIs: domain.ErrInvalidStatusTransition},
		{name: "cancelled rejected", from: domain.StatusCancelled, wantErrIs: domain.ErrInvalidStatusTransition},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ext := driveTo(t, c.from)
			err := ext.MarkRetry()

			if c.wantErrIs != nil {
				if !errors.Is(err, c.wantErrIs) {
					t.Fatalf("err=%v, want errors.Is %v", err, c.wantErrIs)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if ext.Status != c.wantStatus {
				t.Errorf("Status=%q, want %q", ext.Status, c.wantStatus)
			}
			if ext.FailureReason != "" {
				t.Errorf("FailureReason should be cleared, got %q", ext.FailureReason)
			}
			if ext.StartedAt != nil {
				t.Errorf("StartedAt should be reset to nil, got %v", *ext.StartedAt)
			}
			if ext.CompletedAt != nil {
				t.Errorf("CompletedAt should be reset to nil, got %v", *ext.CompletedAt)
			}
		})
	}
}

func mustNewTask(t *testing.T, name, hint string) domain.ExtractedTask {
	t.Helper()
	task, err := domain.NewExtractedTask(name, hint)
	if err != nil {
		t.Fatalf("NewExtractedTask(%q,%q): %v", name, hint, err)
	}
	return task
}
