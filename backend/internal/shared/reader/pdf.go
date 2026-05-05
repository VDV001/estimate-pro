package reader

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/ledongthuc/pdf"
)

// PDFReader extracts plain text from PDF bytes via ledongthuc/pdf.
type PDFReader struct{}

func NewPDFReader() *PDFReader { return &PDFReader{} }

func (r *PDFReader) Supports(ft FileType) bool { return ft == FileTypePDF }

func (r *PDFReader) Parse(ctx context.Context, filename string, data []byte) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if len(data) == 0 {
		return "", fmt.Errorf("pdf %q: %w", filename, ErrEmptyContent)
	}

	pr, err := pdf.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		// ledongthuc surfaces several distinct errors when the
		// trailer carries an /Encrypt dict — the explicit sentinel
		// pdf.ErrInvalidPassword for the standard nil-password
		// case, and ad-hoc fmt.Errorf messages for unsupported
		// filters / versions / key lengths (e.g. AES-256 trips
		// "malformed PDF: 256-bit encryption key" because the
		// library bails before attempting decryption). All of
		// these are "encrypted file we cannot process" from the
		// caller's perspective; collapse them onto ErrEncryptedFile
		// so workers stamp a meaningful failure reason rather than
		// the generic ErrCorruptedFile.
		if isEncryptedPDFError(err) {
			return "", fmt.Errorf("pdf %q: %w (%v)", filename, ErrEncryptedFile, err)
		}
		return "", fmt.Errorf("pdf %q: %w (%v)", filename, ErrCorruptedFile, err)
	}

	var sb strings.Builder
	for i := 1; i <= pr.NumPage(); i++ {
		text, _ := pr.Page(i).GetPlainText(nil)
		sb.WriteString(text)
	}
	return strings.TrimSpace(sb.String()), nil
}

// isEncryptedPDFError reports whether the error returned by
// pdf.NewReader signals an encrypted document. Detection is
// hybrid because ledongthuc/pdf does not expose a single sentinel
// for all encryption-related failure modes:
//
//   - errors.Is(err, pdf.ErrInvalidPassword) covers the standard
//     nil-password case (V=1/2/4 with default user password).
//   - The fallback substring check captures the ad-hoc
//     fmt.Errorf paths for unsupported filters / versions /
//     key lengths ("encryption filter", "encryption version",
//     "<N>-bit encryption key"). All of these errors mention
//     "encrypt" in some form, so a case-insensitive substring
//     check is robust without requiring upstream changes.
//
// Worst case: if a future ledongthuc version drops "encrypt"
// from a relevant message, the error falls through to
// ErrCorruptedFile — graceful degradation rather than panic.
func isEncryptedPDFError(err error) bool {
	if errors.Is(err, pdf.ErrInvalidPassword) {
		return true
	}
	return strings.Contains(strings.ToLower(err.Error()), "encrypt")
}
