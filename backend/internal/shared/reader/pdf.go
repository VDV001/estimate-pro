package reader

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/ledongthuc/pdf"
)

// PDFReader extracts plain text from PDF bytes via ledongthuc/pdf.
type PDFReader struct{}

func NewPDFReader() *PDFReader { return &PDFReader{} }

func (r *PDFReader) Supports(ft FileType) bool { return ft == FileTypePDF }

func (r *PDFReader) Parse(_ context.Context, filename string, data []byte) (string, error) {
	if len(data) == 0 {
		return "", fmt.Errorf("pdf %q: %w", filename, ErrEmptyContent)
	}

	pr, err := pdf.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", fmt.Errorf("pdf %q: %w (%v)", filename, ErrCorruptedFile, err)
	}

	var sb strings.Builder
	for i := 1; i <= pr.NumPage(); i++ {
		text, _ := pr.Page(i).GetPlainText(nil)
		sb.WriteString(text)
	}
	return strings.TrimSpace(sb.String()), nil
}
