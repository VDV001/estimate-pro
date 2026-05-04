// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/VDV001/estimate-pro/backend/internal/modules/extractor/domain"
)

func TestNewExtraction(t *testing.T) {
	tests := []struct {
		name              string
		docID             string
		versionID         string
		wantErrIs         error
		wantDocID         string
		wantVersionID     string
	}{
		{name: "valid", docID: "doc-1", versionID: "ver-1", wantDocID: "doc-1", wantVersionID: "ver-1"},
		{name: "trims fields", docID: "  doc-2  ", versionID: "\tver-2\n", wantDocID: "doc-2", wantVersionID: "ver-2"},
		{name: "empty document id", docID: "", versionID: "ver-1", wantErrIs: domain.ErrMissingDocument},
		{name: "whitespace document id", docID: "   ", versionID: "ver-1", wantErrIs: domain.ErrMissingDocument},
		{name: "empty version id", docID: "doc-1", versionID: "", wantErrIs: domain.ErrMissingDocumentVersion},
		{name: "whitespace version id", docID: "doc-1", versionID: " \t ", wantErrIs: domain.ErrMissingDocumentVersion},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := time.Now()
			ext, err := domain.NewExtraction(tt.docID, tt.versionID)
			after := time.Now()

			if tt.wantErrIs != nil {
				if !errors.Is(err, tt.wantErrIs) {
					t.Fatalf("err=%v, want errors.Is %v", err, tt.wantErrIs)
				}
				if ext != nil {
					t.Errorf("expected nil Extraction on error, got %#v", ext)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}

			if ext.ID == "" {
				t.Error("ID should be auto-assigned")
			}
			if ext.DocumentID != tt.wantDocID {
				t.Errorf("DocumentID=%q, want %q", ext.DocumentID, tt.wantDocID)
			}
			if ext.DocumentVersionID != tt.wantVersionID {
				t.Errorf("DocumentVersionID=%q, want %q", ext.DocumentVersionID, tt.wantVersionID)
			}
			if ext.Status != domain.StatusPending {
				t.Errorf("Status=%q, want %q", ext.Status, domain.StatusPending)
			}
			if len(ext.Tasks) != 0 {
				t.Errorf("Tasks len=%d, want 0", len(ext.Tasks))
			}
			if ext.FailureReason != "" {
				t.Errorf("FailureReason=%q, want empty", ext.FailureReason)
			}
			if ext.StartedAt != nil {
				t.Errorf("StartedAt=%v, want nil", ext.StartedAt)
			}
			if ext.CompletedAt != nil {
				t.Errorf("CompletedAt=%v, want nil", ext.CompletedAt)
			}
			if ext.CreatedAt.Before(before) || ext.CreatedAt.After(after) {
				t.Errorf("CreatedAt=%v, want within [%v, %v]", ext.CreatedAt, before, after)
			}
			if !ext.UpdatedAt.Equal(ext.CreatedAt) {
				t.Errorf("UpdatedAt=%v, want equal to CreatedAt=%v", ext.UpdatedAt, ext.CreatedAt)
			}
		})
	}
}

func TestNewExtraction_GeneratesUniqueIDs(t *testing.T) {
	a, err := domain.NewExtraction("doc", "ver")
	if err != nil {
		t.Fatalf("first: %v", err)
	}
	b, err := domain.NewExtraction("doc", "ver")
	if err != nil {
		t.Fatalf("second: %v", err)
	}
	if a.ID == b.ID {
		t.Errorf("expected unique IDs, got %q twice", a.ID)
	}
}
