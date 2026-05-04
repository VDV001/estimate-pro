// Package reader extracts plain text from document bytes for downstream
// processing (LLM extraction, search indexing, etc.). Stateless, no I/O
// other than the in-memory bytes provided by the caller.
package reader

import (
	"errors"
	"strings"
)

// FileType is a value object representing a supported document format.
type FileType string

const (
	FileTypePDF  FileType = "pdf"
	FileTypeDOCX FileType = "docx"
)

func (ft FileType) String() string { return string(ft) }

// ParseFileType normalises a file extension (with or without leading dot,
// any case) to a supported FileType. Returns ErrUnsupportedFormat for
// extensions outside the current reader catalogue.
func ParseFileType(ext string) (FileType, error) {
	switch strings.ToLower(strings.TrimPrefix(ext, ".")) {
	case "pdf":
		return FileTypePDF, nil
	case "docx":
		return FileTypeDOCX, nil
	default:
		return "", ErrUnsupportedFormat
	}
}

// Sentinel errors. All four are returned by readers and the composite
// dispatcher; consumers match via errors.Is. ErrEncryptedFile is
// intentionally deferred to PR-B3 where worker integration provides
// real encrypted-PDF fixtures (ADR-014: no dead sentinels).
var (
	ErrEmptyContent      = errors.New("reader: empty content")
	ErrUnsupportedFormat = errors.New("reader: unsupported format")
	ErrCorruptedFile     = errors.New("reader: corrupted file")
	ErrFileTooLarge      = errors.New("reader: file too large")
)
