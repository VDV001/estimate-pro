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
	FileTypeMD   FileType = "md"
	FileTypeTXT  FileType = "txt"
	FileTypeCSV  FileType = "csv"
	FileTypeXLSX FileType = "xlsx"
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
	case "md":
		return FileTypeMD, nil
	case "txt":
		return FileTypeTXT, nil
	case "csv":
		return FileTypeCSV, nil
	case "xlsx":
		return FileTypeXLSX, nil
	default:
		return "", ErrUnsupportedFormat
	}
}

// Sentinel errors. All five are returned by readers and the composite
// dispatcher; consumers match via errors.Is. ADR-014: every sentinel
// has a real consumer-branch in the codebase and a test fixture that
// drives it.
var (
	ErrEmptyContent      = errors.New("reader: empty content")
	ErrUnsupportedFormat = errors.New("reader: unsupported format")
	ErrCorruptedFile     = errors.New("reader: corrupted file")
	ErrFileTooLarge      = errors.New("reader: file too large")
	ErrEncryptedFile     = errors.New("reader: encrypted file (password protected)")
)
