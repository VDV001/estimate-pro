// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package domain_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/VDV001/estimate-pro/backend/internal/modules/estimation/domain"
)

// --- NewEstimationItem ---

func TestNewEstimationItem_Valid(t *testing.T) {
	item, err := domain.NewEstimationItem("Setup CI", 2, 4, 8, "initial")
	if err != nil {
		t.Fatalf("NewEstimationItem: %v", err)
	}
	if item.ID == "" {
		t.Error("ID must be auto-generated")
	}
	if item.TaskName != "Setup CI" || item.MinHours != 2 || item.LikelyHours != 4 || item.MaxHours != 8 {
		t.Errorf("fields wrong: %+v", item)
	}
	if item.Note != "initial" {
		t.Errorf("Note = %q", item.Note)
	}
}

func TestNewEstimationItem_Validation(t *testing.T) {
	tests := []struct {
		name       string
		taskName   string
		min        float64
		likely     float64
		max        float64
		wantErr    error
	}{
		{"empty name", "", 1, 2, 3, domain.ErrTaskNameRequired},
		{"whitespace name", "   ", 1, 2, 3, domain.ErrTaskNameRequired},
		{"name too long", strings.Repeat("x", 256), 1, 2, 3, domain.ErrTaskNameTooLong},
		{"negative min", "ok", -1, 2, 3, domain.ErrInvalidHours},
		{"negative likely", "ok", 1, -1, 3, domain.ErrInvalidHours},
		{"negative max", "ok", 1, 2, -3, domain.ErrInvalidHours},
		{"min > likely", "ok", 5, 3, 10, domain.ErrInvalidHours},
		{"likely > max", "ok", 1, 10, 5, domain.ErrInvalidHours},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := domain.NewEstimationItem(tc.taskName, tc.min, tc.likely, tc.max, "")
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("err = %v, want %v", err, tc.wantErr)
			}
		})
	}
}

func TestNewEstimationItem_EqualBoundaries(t *testing.T) {
	// min == likely == max (e.g. fixed-estimate tasks) must be allowed.
	item, err := domain.NewEstimationItem("fixed", 5, 5, 5, "")
	if err != nil {
		t.Errorf("equal boundaries must be valid: %v", err)
	}
	if item == nil {
		t.Fatal("item must not be nil")
	}
}

func TestEstimationItem_AttachTo(t *testing.T) {
	item, _ := domain.NewEstimationItem("task", 1, 2, 3, "")
	item.AttachTo("est-1", 7)

	if item.EstimationID != "est-1" {
		t.Errorf("EstimationID = %q, want est-1", item.EstimationID)
	}
	if item.SortOrder != 7 {
		t.Errorf("SortOrder = %d, want 7", item.SortOrder)
	}
}

// --- NewEstimation ---

func TestNewEstimation_Valid(t *testing.T) {
	est, err := domain.NewEstimation("proj-1", "user-1", "doc-ver-1")
	if err != nil {
		t.Fatalf("NewEstimation: %v", err)
	}
	if est.ID == "" {
		t.Error("ID must be auto-generated")
	}
	if est.Status != domain.StatusDraft {
		t.Errorf("Status = %q, want draft", est.Status)
	}
	if est.ProjectID != "proj-1" || est.SubmittedBy != "user-1" || est.DocumentVersionID != "doc-ver-1" {
		t.Errorf("fields wrong: %+v", est)
	}
	if est.CreatedAt.IsZero() {
		t.Error("CreatedAt must be set")
	}
	if !est.SubmittedAt.IsZero() {
		t.Error("SubmittedAt must be zero on draft")
	}
}

func TestNewEstimation_Validation(t *testing.T) {
	tests := []struct {
		name        string
		projectID   string
		submittedBy string
		want        error
	}{
		{"empty project", "", "u1", domain.ErrMissingProject},
		{"empty author", "p1", "", domain.ErrMissingAuthor},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := domain.NewEstimation(tc.projectID, tc.submittedBy, "")
			if !errors.Is(err, tc.want) {
				t.Errorf("err = %v, want %v", err, tc.want)
			}
		})
	}
}

func TestEstimation_Submit(t *testing.T) {
	est, _ := domain.NewEstimation("p1", "u1", "")

	if err := est.Submit(); err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if est.Status != domain.StatusSubmitted {
		t.Errorf("Status = %q, want submitted", est.Status)
	}
	if est.SubmittedAt.IsZero() {
		t.Error("SubmittedAt must be set after Submit")
	}
}

func TestEstimation_Submit_AlreadySubmitted(t *testing.T) {
	est, _ := domain.NewEstimation("p1", "u1", "")
	_ = est.Submit()

	err := est.Submit()
	if !errors.Is(err, domain.ErrAlreadySubmitted) {
		t.Errorf("err = %v, want ErrAlreadySubmitted", err)
	}
}

func TestEstimation_AuthorizeAuthor(t *testing.T) {
	est, _ := domain.NewEstimation("p1", "u1", "")

	if err := est.AuthorizeAuthor("u1"); err != nil {
		t.Errorf("own author: %v", err)
	}
	if err := est.AuthorizeAuthor("u2"); !errors.Is(err, domain.ErrNotAuthor) {
		t.Errorf("other: err = %v, want ErrNotAuthor", err)
	}
}
