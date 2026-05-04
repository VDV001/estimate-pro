package reader

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/xuri/excelize/v2"
)

// XLSXReader extracts text from .xlsx workbooks via xuri/excelize/v2.
// Sheets are joined with a blank-line separator; within a sheet rows
// join on '\n' and cells on '\t'.
type XLSXReader struct{}

func NewXLSXReader() *XLSXReader { return &XLSXReader{} }

func (r *XLSXReader) Supports(ft FileType) bool { return ft == FileTypeXLSX }

func (r *XLSXReader) Parse(ctx context.Context, filename string, data []byte) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if len(data) == 0 {
		return "", fmt.Errorf("xlsx %q: %w", filename, ErrEmptyContent)
	}

	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("xlsx %q: %w (%v)", filename, ErrCorruptedFile, err)
	}
	defer func() { _ = f.Close() }()

	var sb strings.Builder
	for sheetIdx, sheet := range f.GetSheetList() {
		rows, err := f.GetRows(sheet)
		if err != nil {
			return "", fmt.Errorf("xlsx %q: %w (%v)", filename, ErrCorruptedFile, err)
		}
		if sheetIdx > 0 {
			sb.WriteString("\n\n")
		}
		for i, row := range rows {
			if i > 0 {
				sb.WriteByte('\n')
			}
			for j, cell := range row {
				if j > 0 {
					sb.WriteByte('\t')
				}
				sb.WriteString(cell)
			}
		}
	}
	return strings.TrimSpace(sb.String()), nil
}
