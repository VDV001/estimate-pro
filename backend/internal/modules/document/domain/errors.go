package domain

import "errors"

var (
	ErrDocumentNotFound    = errors.New("document not found")
	ErrVersionNotFound     = errors.New("document version not found")
	ErrUnsupportedFileType = errors.New("unsupported file type")
	ErrFileTooLarge        = errors.New("file exceeds maximum allowed size")
)
