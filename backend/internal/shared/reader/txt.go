package reader

import (
	"context"
	"fmt"
	"unicode/utf8"
)

// TXTReader returns plain-text bytes verbatim, rejecting any payload
// that is not valid UTF-8.
type TXTReader struct{}

func NewTXTReader() *TXTReader { return &TXTReader{} }

func (r *TXTReader) Supports(ft FileType) bool { return ft == FileTypeTXT }

func (r *TXTReader) Parse(ctx context.Context, filename string, data []byte) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if len(data) == 0 {
		return "", fmt.Errorf("txt %q: %w", filename, ErrEmptyContent)
	}
	if !utf8.Valid(data) {
		return "", fmt.Errorf("txt %q: %w (invalid utf-8)", filename, ErrCorruptedFile)
	}
	return string(data), nil
}
