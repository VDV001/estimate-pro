package reader_test

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"

	"github.com/VDV001/estimate-pro/backend/internal/shared/reader"
)

func TestXLSXReader_Supports(t *testing.T) {
	r := reader.NewXLSXReader()
	tests := []struct {
		name string
		ft   reader.FileType
		want bool
	}{
		{name: "XLSX", ft: reader.FileTypeXLSX, want: true},
		{name: "CSV", ft: reader.FileTypeCSV, want: false},
		{name: "DOCX", ft: reader.FileTypeDOCX, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := r.Supports(tt.ft); got != tt.want {
				t.Errorf("Supports(%v)=%v, want %v", tt.ft, got, tt.want)
			}
		})
	}
}

func TestXLSXReader_Parse(t *testing.T) {
	r := reader.NewXLSXReader()

	singleSheet := buildXLSX(t, map[string][][]string{
		"Sheet1": {{"name", "age"}, {"Alice", "30"}, {"Bob", "25"}},
	}, []string{"Sheet1"})

	multiSheet := buildXLSX(t, map[string][][]string{
		"Tasks":   {{"id", "title"}, {"1", "Design"}, {"2", "Implement"}},
		"Members": {{"role", "name"}, {"PM", "Alice"}},
	}, []string{"Tasks", "Members"})

	tests := []struct {
		name           string
		data           []byte
		wantErrIs      error
		wantSubstrings []string
	}{
		{
			name:           "single sheet",
			data:           singleSheet,
			wantSubstrings: []string{"name\tage", "Alice\t30", "Bob\t25"},
		},
		{
			name:           "multiple sheets",
			data:           multiSheet,
			wantSubstrings: []string{"id\ttitle", "Design", "role\tname", "PM\tAlice"},
		},
		{name: "empty bytes", data: nil, wantErrIs: reader.ErrEmptyContent},
		{name: "zero-length bytes", data: []byte{}, wantErrIs: reader.ErrEmptyContent},
		{name: "not a zip", data: []byte("not a xlsx at all"), wantErrIs: reader.ErrCorruptedFile},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			text, err := r.Parse(t.Context(), "doc.xlsx", tt.data)
			if tt.wantErrIs != nil {
				if !errors.Is(err, tt.wantErrIs) {
					t.Fatalf("Parse err=%v, want errors.Is %v", err, tt.wantErrIs)
				}
				return
			}
			if err != nil {
				t.Fatalf("Parse unexpected err: %v", err)
			}
			for _, sub := range tt.wantSubstrings {
				if !strings.Contains(text, sub) {
					t.Errorf("Parse text =\n%q\nwant substring %q", text, sub)
				}
			}
		})
	}
}

func TestXLSXReader_Parse_RespectsCancelledContext(t *testing.T) {
	r := reader.NewXLSXReader()
	data := buildXLSX(t, map[string][][]string{
		"Sheet1": {{"a", "b"}},
	}, []string{"Sheet1"})

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	if _, err := r.Parse(ctx, "doc.xlsx", data); !errors.Is(err, context.Canceled) {
		t.Fatalf("Parse err=%v, want errors.Is context.Canceled", err)
	}
}

// buildXLSX assembles an XLSX file in memory from the provided sheet → rows map.
// sheetOrder pins the sheet enumeration sequence so multi-sheet tests can rely
// on stable ordering.
func buildXLSX(t *testing.T, sheets map[string][][]string, sheetOrder []string) []byte {
	t.Helper()

	f := excelize.NewFile()
	t.Cleanup(func() { _ = f.Close() })

	defaultSheet := f.GetSheetName(0)
	for i, name := range sheetOrder {
		if i == 0 {
			if name != defaultSheet {
				if err := f.SetSheetName(defaultSheet, name); err != nil {
					t.Fatalf("rename default sheet: %v", err)
				}
			}
			continue
		}
		if _, err := f.NewSheet(name); err != nil {
			t.Fatalf("new sheet %q: %v", name, err)
		}
	}

	for _, name := range sheetOrder {
		rows := sheets[name]
		for i, row := range rows {
			cell, err := excelize.CoordinatesToCellName(1, i+1)
			if err != nil {
				t.Fatalf("coord: %v", err)
			}
			cells := make([]any, len(row))
			for j, v := range row {
				cells[j] = v
			}
			if err := f.SetSheetRow(name, cell, &cells); err != nil {
				t.Fatalf("set row: %v", err)
			}
		}
	}

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		t.Fatalf("write xlsx: %v", err)
	}
	return buf.Bytes()
}

// Compile-time check: *XLSXReader satisfies DocumentReader.
var _ reader.DocumentReader = (*reader.XLSXReader)(nil)
