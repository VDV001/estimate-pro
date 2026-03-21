// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package domain

import "time"

type FileType string

const (
	FileTypePDF  FileType = "pdf"
	FileTypeDOCX FileType = "docx"
	FileTypeXLSX FileType = "xlsx"
	FileTypeMD   FileType = "md"
	FileTypeTXT  FileType = "txt"
	FileTypeCSV  FileType = "csv"
)

func (ft FileType) IsValid() bool {
	switch ft {
	case FileTypePDF, FileTypeDOCX, FileTypeXLSX, FileTypeMD, FileTypeTXT, FileTypeCSV:
		return true
	}
	return false
}

type ParsedStatus string

const (
	ParsedStatusPending     ParsedStatus = "pending"
	ParsedStatusParsed      ParsedStatus = "parsed"
	ParsedStatusFailed      ParsedStatus = "failed"
	ParsedStatusNeedsReview ParsedStatus = "needs_review"
)

const MaxFileSize int64 = 50 << 20 // 50MB

type Document struct {
	ID         string    `json:"id"`
	ProjectID  string    `json:"project_id"`
	Title      string    `json:"title"`
	UploadedBy string    `json:"uploaded_by"`
	CreatedAt  time.Time `json:"created_at,omitzero"`
}

type DocumentVersion struct {
	ID              string       `json:"id"`
	DocumentID      string       `json:"document_id"`
	VersionNumber   int          `json:"version_number"`
	FileKey         string       `json:"file_key"`
	FileType        FileType     `json:"file_type"`
	FileSize        int64        `json:"file_size"`
	ParsedStatus    ParsedStatus `json:"parsed_status"`
	ConfidenceScore float64      `json:"confidence_score"`
	IsSigned        bool         `json:"is_signed"`
	IsFinal         bool         `json:"is_final"`
	Tags            []string     `json:"tags,omitempty"`
	UploadedBy      string       `json:"uploaded_by"`
	UploadedAt      time.Time    `json:"uploaded_at,omitzero"`
}

// Predefined tags
var PredefinedTags = []string{
	"на_подпись", "подписана", "на_правках", "отклонена",
	"черновик", "от_заказчика", "спорная", "архив", "срочно",
}

const MaxTagsPerVersion = 3
