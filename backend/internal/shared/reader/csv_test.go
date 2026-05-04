package reader_test

import (
	"context"
	"errors"
	"testing"

	"github.com/VDV001/estimate-pro/backend/internal/shared/reader"
)

func TestCSVReader_Supports(t *testing.T) {
	r := reader.NewCSVReader()
	tests := []struct {
		name string
		ft   reader.FileType
		want bool
	}{
		{name: "CSV", ft: reader.FileTypeCSV, want: true},
		{name: "TXT", ft: reader.FileTypeTXT, want: false},
		{name: "XLSX", ft: reader.FileTypeXLSX, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := r.Supports(tt.ft); got != tt.want {
				t.Errorf("Supports(%v)=%v, want %v", tt.ft, got, tt.want)
			}
		})
	}
}

func TestCSVReader_Parse(t *testing.T) {
	r := reader.NewCSVReader()

	tests := []struct {
		name      string
		data      []byte
		wantErrIs error
		wantText  string
	}{
		{
			name:     "simple csv",
			data:     []byte("name,age\nAlice,30\nBob,25"),
			wantText: "name\tage\nAlice\t30\nBob\t25",
		},
		{
			name:     "quoted fields with comma",
			data:     []byte(`task,owner` + "\n" + `"Buy, sell, hold",Alice`),
			wantText: "task\towner\nBuy, sell, hold\tAlice",
		},
		{
			name:     "variable column count",
			data:     []byte("a,b,c\nx,y\nz"),
			wantText: "a\tb\tc\nx\ty\nz",
		},
		{name: "empty bytes", data: nil, wantErrIs: reader.ErrEmptyContent},
		{name: "zero-length bytes", data: []byte{}, wantErrIs: reader.ErrEmptyContent},
		{
			name:      "bare quote inside unquoted field",
			data:      []byte(`name,note` + "\n" + `Alice,she said "hi`),
			wantErrIs: reader.ErrCorruptedFile,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			text, err := r.Parse(t.Context(), "doc.csv", tt.data)
			if tt.wantErrIs != nil {
				if !errors.Is(err, tt.wantErrIs) {
					t.Fatalf("Parse err=%v, want errors.Is %v", err, tt.wantErrIs)
				}
				return
			}
			if err != nil {
				t.Fatalf("Parse unexpected err: %v", err)
			}
			if text != tt.wantText {
				t.Errorf("Parse text =\n%q\nwant\n%q", text, tt.wantText)
			}
		})
	}
}

func TestCSVReader_Parse_RespectsCancelledContext(t *testing.T) {
	r := reader.NewCSVReader()
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	if _, err := r.Parse(ctx, "doc.csv", []byte("a,b\n1,2")); !errors.Is(err, context.Canceled) {
		t.Fatalf("Parse err=%v, want errors.Is context.Canceled", err)
	}
}

// Compile-time check: *CSVReader satisfies DocumentReader.
var _ reader.DocumentReader = (*reader.CSVReader)(nil)
