// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package domain_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/VDV001/estimate-pro/backend/internal/modules/document/domain"
)

func TestNewDocument_Valid(t *testing.T) {
	d, err := domain.NewDocument("proj-1", "Specs", "user-1")
	if err != nil {
		t.Fatalf("NewDocument: %v", err)
	}
	if d.ID == "" {
		t.Error("ID must be auto-generated")
	}
	if d.ProjectID != "proj-1" || d.Title != "Specs" || d.UploadedBy != "user-1" {
		t.Errorf("fields wrong: %+v", d)
	}
	if d.CreatedAt.IsZero() {
		t.Error("CreatedAt must be set")
	}
}

func TestNewDocument_Validation(t *testing.T) {
	tests := []struct {
		name       string
		projectID  string
		title      string
		uploadedBy string
		want       error
	}{
		{"empty project", "", "t", "u1", domain.ErrMissingProject},
		{"empty title", "p1", "", "u1", domain.ErrInvalidTitle},
		{"whitespace title", "p1", "   ", "u1", domain.ErrInvalidTitle},
		{"title too long", "p1", strings.Repeat("x", 256), "u1", domain.ErrInvalidTitle},
		{"empty uploader", "p1", "t", "", domain.ErrMissingUploader},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := domain.NewDocument(tc.projectID, tc.title, tc.uploadedBy)
			if !errors.Is(err, tc.want) {
				t.Errorf("err = %v, want %v", err, tc.want)
			}
		})
	}
}

func TestNewDocumentVersion_Valid(t *testing.T) {
	v, err := domain.NewDocumentVersion("doc-1", 1, "key", domain.FileTypePDF, 1024, "user-1")
	if err != nil {
		t.Fatalf("NewDocumentVersion: %v", err)
	}
	if v.ID == "" {
		t.Error("ID must be auto-generated")
	}
	if v.DocumentID != "doc-1" || v.VersionNumber != 1 || v.FileKey != "key" {
		t.Errorf("fields wrong: %+v", v)
	}
	if v.FileType != domain.FileTypePDF || v.FileSize != 1024 {
		t.Errorf("file fields wrong: %+v", v)
	}
	if v.ParsedStatus != domain.ParsedStatusPending {
		t.Errorf("ParsedStatus = %q, want pending", v.ParsedStatus)
	}
	if v.UploadedAt.IsZero() {
		t.Error("UploadedAt must be set")
	}
}

func TestNewDocumentVersion_Validation(t *testing.T) {
	tests := []struct {
		name       string
		docID      string
		versionNo  int
		fileKey    string
		fileType   domain.FileType
		fileSize   int64
		uploadedBy string
		want       error
	}{
		{"empty doc", "", 1, "k", domain.FileTypePDF, 100, "u1", domain.ErrMissingDocument},
		{"zero version", "d1", 0, "k", domain.FileTypePDF, 100, "u1", domain.ErrInvalidVersion},
		{"negative version", "d1", -1, "k", domain.FileTypePDF, 100, "u1", domain.ErrInvalidVersion},
		{"empty file key", "d1", 1, "", domain.FileTypePDF, 100, "u1", domain.ErrMissingFileKey},
		{"invalid file type", "d1", 1, "k", domain.FileType("exe"), 100, "u1", domain.ErrUnsupportedFileType},
		{"negative size", "d1", 1, "k", domain.FileTypePDF, -1, "u1", domain.ErrInvalidFileSize},
		{"too large", "d1", 1, "k", domain.FileTypePDF, domain.MaxFileSize + 1, "u1", domain.ErrFileTooLarge},
		{"empty uploader", "d1", 1, "k", domain.FileTypePDF, 100, "", domain.ErrMissingUploader},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := domain.NewDocumentVersion(tc.docID, tc.versionNo, tc.fileKey, tc.fileType, tc.fileSize, tc.uploadedBy)
			if !errors.Is(err, tc.want) {
				t.Errorf("err = %v, want %v", err, tc.want)
			}
		})
	}
}

func TestDocumentVersion_SetFlags(t *testing.T) {
	v, _ := domain.NewDocumentVersion("d1", 1, "k", domain.FileTypePDF, 100, "u1")

	v.SetFlags(true, false)
	if !v.IsSigned || v.IsFinal {
		t.Errorf("flags wrong: signed=%v final=%v", v.IsSigned, v.IsFinal)
	}

	v.SetFlags(false, true)
	if v.IsSigned || !v.IsFinal {
		t.Errorf("flags wrong: signed=%v final=%v", v.IsSigned, v.IsFinal)
	}
}
