// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package domain

import "errors"

var (
	ErrDocumentNotFound    = errors.New("document not found")
	ErrVersionNotFound     = errors.New("document version not found")
	ErrUnsupportedFileType = errors.New("unsupported file type")
	ErrFileTooLarge        = errors.New("file exceeds maximum allowed size")
	ErrMissingProject      = errors.New("document project is required")
	ErrInvalidTitle        = errors.New("document title must be 1..255 characters")
	ErrMissingUploader     = errors.New("document uploader is required")
	ErrMissingDocument     = errors.New("document version parent document is required")
	ErrInvalidVersion      = errors.New("version number must be positive")
	ErrMissingFileKey      = errors.New("document version file key is required")
	ErrInvalidFileSize     = errors.New("document file size must be non-negative")
)
