// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/VDV001/estimate-pro/backend/internal/modules/extractor/domain"
)

func TestNewExtractionEvent(t *testing.T) {
	tests := []struct {
		name         string
		extractionID string
		from         domain.ExtractionStatus
		to           domain.ExtractionStatus
		errorMsg     string
		actor        string
		wantErrIs    error
	}{
		{
			name:         "valid pending → processing",
			extractionID: "ext-1",
			from:         domain.StatusPending,
			to:           domain.StatusProcessing,
			actor:        "worker",
		},
		{
			name:         "valid processing → failed with error",
			extractionID: "ext-2",
			from:         domain.StatusProcessing,
			to:           domain.StatusFailed,
			errorMsg:     "LLM timed out",
			actor:        "worker",
		},
		{
			name:         "trims fields",
			extractionID: "  ext-3  ",
			from:         domain.StatusPending,
			to:           domain.StatusCancelled,
			errorMsg:     "  user request  ",
			actor:        "  user:42  ",
		},
		{
			name:         "empty extraction id",
			extractionID: "",
			from:         domain.StatusPending,
			to:           domain.StatusProcessing,
			actor:        "worker",
			wantErrIs:    domain.ErrMissingExtraction,
		},
		{
			name:         "whitespace extraction id",
			extractionID: "  ",
			from:         domain.StatusPending,
			to:           domain.StatusProcessing,
			actor:        "worker",
			wantErrIs:    domain.ErrMissingExtraction,
		},
		{
			name:         "invalid from status",
			extractionID: "ext-1",
			from:         "running",
			to:           domain.StatusProcessing,
			actor:        "worker",
			wantErrIs:    domain.ErrInvalidStatusTransition,
		},
		{
			name:         "invalid to status",
			extractionID: "ext-1",
			from:         domain.StatusPending,
			to:           "running",
			actor:        "worker",
			wantErrIs:    domain.ErrInvalidStatusTransition,
		},
		{
			name:         "empty actor",
			extractionID: "ext-1",
			from:         domain.StatusPending,
			to:           domain.StatusProcessing,
			actor:        "",
			wantErrIs:    domain.ErrInvalidActor,
		},
		{
			name:         "whitespace actor",
			extractionID: "ext-1",
			from:         domain.StatusPending,
			to:           domain.StatusProcessing,
			actor:        " \t ",
			wantErrIs:    domain.ErrInvalidActor,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := time.Now()
			ev, err := domain.NewExtractionEvent(tt.extractionID, tt.from, tt.to, tt.errorMsg, tt.actor)
			after := time.Now()

			if tt.wantErrIs != nil {
				if !errors.Is(err, tt.wantErrIs) {
					t.Fatalf("err=%v, want errors.Is %v", err, tt.wantErrIs)
				}
				if ev != nil {
					t.Errorf("expected nil event on error, got %#v", ev)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if ev.ID == "" {
				t.Error("ID should be auto-assigned")
			}
			if ev.ExtractionID == "" {
				t.Error("ExtractionID should be set")
			}
			if ev.FromStatus != tt.from {
				t.Errorf("FromStatus=%q, want %q", ev.FromStatus, tt.from)
			}
			if ev.ToStatus != tt.to {
				t.Errorf("ToStatus=%q, want %q", ev.ToStatus, tt.to)
			}
			if ev.CreatedAt.Before(before) || ev.CreatedAt.After(after) {
				t.Errorf("CreatedAt=%v outside window [%v, %v]", ev.CreatedAt, before, after)
			}
		})
	}
}

func TestNewExtractionEvent_TrimsFields(t *testing.T) {
	ev, err := domain.NewExtractionEvent("  ext-trim  ", domain.StatusPending, domain.StatusCancelled, "  reason  ", "  user:1  ")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if ev.ExtractionID != "ext-trim" {
		t.Errorf("ExtractionID=%q, want %q", ev.ExtractionID, "ext-trim")
	}
	if ev.ErrorMessage != "reason" {
		t.Errorf("ErrorMessage=%q, want %q", ev.ErrorMessage, "reason")
	}
	if ev.Actor != "user:1" {
		t.Errorf("Actor=%q, want %q", ev.Actor, "user:1")
	}
}

func TestNewExtractionEvent_GeneratesUniqueIDs(t *testing.T) {
	a, _ := domain.NewExtractionEvent("ext", domain.StatusPending, domain.StatusProcessing, "", "worker")
	b, _ := domain.NewExtractionEvent("ext", domain.StatusPending, domain.StatusProcessing, "", "worker")
	if a.ID == b.ID {
		t.Errorf("expected unique IDs, got %q twice", a.ID)
	}
}
