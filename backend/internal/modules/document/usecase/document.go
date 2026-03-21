// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package usecase

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"

	"github.com/VDV001/estimate-pro/backend/internal/modules/document/domain"
)

type DocumentUsecase struct {
	docRepo     domain.DocumentRepository
	versionRepo domain.VersionRepository
	storage     domain.FileStorage
}

func New(docRepo domain.DocumentRepository, versionRepo domain.VersionRepository, storage domain.FileStorage) *DocumentUsecase {
	return &DocumentUsecase{docRepo: docRepo, versionRepo: versionRepo, storage: storage}
}

type UploadInput struct {
	ProjectID string
	Title     string
	FileName  string
	FileSize  int64
	FileType  domain.FileType
	Content   io.Reader
	UserID    string
}

type DocumentWithLatestVersion struct {
	Document      *domain.Document        `json:"document"`
	LatestVersion *domain.DocumentVersion  `json:"latest_version,omitempty"`
}

func (uc *DocumentUsecase) Upload(ctx context.Context, input UploadInput) (*domain.Document, *domain.DocumentVersion, error) {
	if !input.FileType.IsValid() {
		return nil, nil, fmt.Errorf("document.Upload: %w", domain.ErrUnsupportedFileType)
	}
	if input.FileSize > domain.MaxFileSize {
		return nil, nil, fmt.Errorf("document.Upload: %w", domain.ErrFileTooLarge)
	}

	now := time.Now()

	doc := &domain.Document{
		ID:         uuid.New().String(),
		ProjectID:  input.ProjectID,
		Title:      input.Title,
		UploadedBy: input.UserID,
		CreatedAt:  now,
	}
	if err := uc.docRepo.Create(ctx, doc); err != nil {
		return nil, nil, fmt.Errorf("document.Upload create doc: %w", err)
	}

	versionID := uuid.New().String()
	fileKey := fmt.Sprintf("documents/%s/%s/%s/%s", input.ProjectID, doc.ID, versionID, input.FileName)

	if err := uc.storage.Upload(ctx, fileKey, input.Content, input.FileSize, contentTypeByFileType(input.FileType)); err != nil {
		return nil, nil, fmt.Errorf("document.Upload s3: %w", err)
	}

	version := &domain.DocumentVersion{
		ID:            versionID,
		DocumentID:    doc.ID,
		VersionNumber: 1,
		FileKey:       fileKey,
		FileType:      input.FileType,
		FileSize:      input.FileSize,
		ParsedStatus:  domain.ParsedStatusPending,
		UploadedBy:    input.UserID,
		UploadedAt:    now,
	}
	if err := uc.versionRepo.Create(ctx, version); err != nil {
		return nil, nil, fmt.Errorf("document.Upload create version: %w", err)
	}

	return doc, version, nil
}

func (uc *DocumentUsecase) List(ctx context.Context, projectID string) ([]*domain.Document, error) {
	docs, err := uc.docRepo.ListByProject(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("document.List: %w", err)
	}
	return docs, nil
}

func (uc *DocumentUsecase) Get(ctx context.Context, id string) (*DocumentWithLatestVersion, error) {
	doc, err := uc.docRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("document.Get: %w", err)
	}

	latest, err := uc.versionRepo.GetLatestByDocument(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("document.Get latest version: %w", err)
	}

	// Load tags
	tags, _ := uc.versionRepo.GetTags(ctx, latest.ID)
	latest.Tags = tags

	return &DocumentWithLatestVersion{Document: doc, LatestVersion: latest}, nil
}

func (uc *DocumentUsecase) GetVersions(ctx context.Context, documentID string) ([]*domain.DocumentVersion, error) {
	versions, err := uc.versionRepo.ListByDocument(ctx, documentID)
	if err != nil {
		return nil, fmt.Errorf("document.GetVersions: %w", err)
	}
	return versions, nil
}

func (uc *DocumentUsecase) Download(ctx context.Context, documentID string) (io.ReadCloser, string, error) {
	latest, err := uc.versionRepo.GetLatestByDocument(ctx, documentID)
	if err != nil {
		return nil, "", fmt.Errorf("document.Download: %w", err)
	}

	reader, err := uc.storage.Download(ctx, latest.FileKey)
	if err != nil {
		return nil, "", fmt.Errorf("document.Download s3: %w", err)
	}

	return reader, latest.FileKey, nil
}

func (uc *DocumentUsecase) Delete(ctx context.Context, id, userID string) error {
	versions, err := uc.versionRepo.ListByDocument(ctx, id)
	if err != nil {
		return fmt.Errorf("document.Delete list versions: %w", err)
	}

	for _, v := range versions {
		if err := uc.storage.Delete(ctx, v.FileKey); err != nil {
			return fmt.Errorf("document.Delete s3 key=%s: %w", v.FileKey, err)
		}
	}

	if err := uc.docRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("document.Delete: %w", err)
	}

	return nil
}

func (uc *DocumentUsecase) UpdateVersionFlags(ctx context.Context, projectID, versionID string, isSigned, isFinal bool) error {
	// If setting as final, clear final flag from ALL versions in ALL documents of this project
	if isFinal {
		if err := uc.versionRepo.ClearFinalByProject(ctx, projectID); err != nil {
			return fmt.Errorf("document.UpdateVersionFlags clear: %w", err)
		}
	}

	return uc.versionRepo.UpdateFlags(ctx, versionID, isSigned, isFinal)
}

func (uc *DocumentUsecase) SetVersionTags(ctx context.Context, versionID string, tags []string) error {
	if len(tags) > domain.MaxTagsPerVersion {
		return fmt.Errorf("document.SetVersionTags: max %d tags allowed", domain.MaxTagsPerVersion)
	}
	return uc.versionRepo.SetTags(ctx, versionID, tags)
}

func contentTypeByFileType(ft domain.FileType) string {
	switch ft {
	case domain.FileTypePDF:
		return "application/pdf"
	case domain.FileTypeDOCX:
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case domain.FileTypeXLSX:
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	case domain.FileTypeMD:
		return "text/markdown"
	case domain.FileTypeTXT:
		return "text/plain"
	case domain.FileTypeCSV:
		return "text/csv"
	default:
		return "application/octet-stream"
	}
}
