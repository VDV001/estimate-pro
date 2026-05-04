package reader

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"strings"
)

// CSVReader extracts text from CSV bytes via stdlib encoding/csv.
// Rows are joined with '\n', cells with '\t'.
type CSVReader struct{}

func NewCSVReader() *CSVReader { return &CSVReader{} }

func (r *CSVReader) Supports(ft FileType) bool { return ft == FileTypeCSV }

func (r *CSVReader) Parse(ctx context.Context, filename string, data []byte) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if len(data) == 0 {
		return "", fmt.Errorf("csv %q: %w", filename, ErrEmptyContent)
	}

	cr := csv.NewReader(bytes.NewReader(data))
	cr.FieldsPerRecord = -1 // tolerate ragged rows
	rows, err := cr.ReadAll()
	if err != nil {
		return "", fmt.Errorf("csv %q: %w (%v)", filename, ErrCorruptedFile, err)
	}

	var sb strings.Builder
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
	return sb.String(), nil
}
