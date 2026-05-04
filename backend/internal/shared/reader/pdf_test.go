package reader_test

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/VDV001/estimate-pro/backend/internal/shared/reader"
)

func TestPDFReader_Supports(t *testing.T) {
	r := reader.NewPDFReader()
	tests := []struct {
		name string
		ft   reader.FileType
		want bool
	}{
		{name: "PDF", ft: reader.FileTypePDF, want: true},
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

func TestPDFReader_Parse(t *testing.T) {
	r := reader.NewPDFReader()

	hello, err := os.ReadFile("testdata/hello.pdf")
	if err != nil {
		t.Fatalf("read fixture hello.pdf: %v", err)
	}
	multipage, err := os.ReadFile("testdata/multipage.pdf")
	if err != nil {
		t.Fatalf("read fixture multipage.pdf: %v", err)
	}

	tests := []struct {
		name        string
		filename    string
		data        []byte
		wantErrIs   error
		wantNonZero bool
	}{
		{name: "valid PDF", filename: "hello.pdf", data: hello, wantNonZero: true},
		{name: "multipage PDF", filename: "multipage.pdf", data: multipage, wantNonZero: true},
		{name: "empty bytes", filename: "empty.pdf", data: nil, wantErrIs: reader.ErrEmptyContent},
		{name: "zero-length bytes", filename: "empty.pdf", data: []byte{}, wantErrIs: reader.ErrEmptyContent},
		{name: "corrupt bytes", filename: "bad.pdf", data: []byte("not a pdf at all"), wantErrIs: reader.ErrCorruptedFile},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			text, err := r.Parse(t.Context(), tt.filename, tt.data)
			if tt.wantErrIs != nil {
				if !errors.Is(err, tt.wantErrIs) {
					t.Fatalf("Parse err=%v, want errors.Is %v", err, tt.wantErrIs)
				}
				return
			}
			if err != nil {
				t.Fatalf("Parse unexpected err: %v", err)
			}
			if tt.wantNonZero && text == "" {
				t.Error("Parse returned empty text, want non-empty")
			}
		})
	}
}

func TestPDFReader_Parse_RespectsCancelledContext(t *testing.T) {
	r := reader.NewPDFReader()
	hello, err := os.ReadFile("testdata/hello.pdf")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	if _, err := r.Parse(ctx, "hello.pdf", hello); !errors.Is(err, context.Canceled) {
		t.Fatalf("Parse err=%v, want errors.Is context.Canceled", err)
	}
}

// Compile-time check: PDFReader satisfies DocumentReader.
var _ reader.DocumentReader = (*reader.PDFReader)(nil)
