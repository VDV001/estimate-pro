// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package domain

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

const maxDocumentTitleLen = 255

// NewDocument constructs a Document enforcing invariants: non-empty project,
// non-empty trimmed title (1..255), non-empty uploader. Auto ID + CreatedAt.
func NewDocument(projectID, title, uploadedBy string) (*Document, error) {
	if projectID == "" {
		return nil, ErrMissingProject
	}
	trimmed := strings.TrimSpace(title)
	if trimmed == "" || len(trimmed) > maxDocumentTitleLen {
		return nil, ErrInvalidTitle
	}
	if uploadedBy == "" {
		return nil, ErrMissingUploader
	}
	return &Document{
		ID:         uuid.New().String(),
		ProjectID:  projectID,
		Title:      trimmed,
		UploadedBy: uploadedBy,
		CreatedAt:  time.Now(),
	}, nil
}

// NewDocumentVersion constructs a DocumentVersion enforcing invariants:
// non-empty document, positive version number, non-empty file key, valid
// file type, file size within MaxFileSize, non-empty uploader.
// ParsedStatus defaults to pending. Auto ID + UploadedAt.
func NewDocumentVersion(documentID string, versionNumber int, fileKey string, fileType FileType, fileSize int64, uploadedBy string) (*DocumentVersion, error) {
	if documentID == "" {
		return nil, ErrMissingDocument
	}
	if versionNumber <= 0 {
		return nil, ErrInvalidVersion
	}
	if fileKey == "" {
		return nil, ErrMissingFileKey
	}
	if !fileType.IsValid() {
		return nil, ErrUnsupportedFileType
	}
	if fileSize < 0 {
		return nil, ErrInvalidFileSize
	}
	if fileSize > MaxFileSize {
		return nil, ErrFileTooLarge
	}
	if uploadedBy == "" {
		return nil, ErrMissingUploader
	}
	return &DocumentVersion{
		ID:            uuid.New().String(),
		DocumentID:    documentID,
		VersionNumber: versionNumber,
		FileKey:       fileKey,
		FileType:      fileType,
		FileSize:      fileSize,
		ParsedStatus:  ParsedStatusPending,
		UploadedBy:    uploadedBy,
		UploadedAt:    time.Now(),
	}, nil
}

// SetFlags updates IsSigned and IsFinal on a version.
func (v *DocumentVersion) SetFlags(isSigned, isFinal bool) {
	v.IsSigned = isSigned
	v.IsFinal = isFinal
}
