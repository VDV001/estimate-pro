package reader_test

import (
	"context"
	"errors"
	"testing"

	"github.com/VDV001/estimate-pro/backend/internal/shared/reader"
)

func TestMDReader_Supports(t *testing.T) {
	r := reader.NewMDReader()
	tests := []struct {
		name string
		ft   reader.FileType
		want bool
	}{
		{name: "MD", ft: reader.FileTypeMD, want: true},
		{name: "TXT", ft: reader.FileTypeTXT, want: false},
		{name: "PDF", ft: reader.FileTypePDF, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := r.Supports(tt.ft); got != tt.want {
				t.Errorf("Supports(%v)=%v, want %v", tt.ft, got, tt.want)
			}
		})
	}
}

func TestMDReader_Parse(t *testing.T) {
	r := reader.NewMDReader()
	const sample = "# Title\n\nParagraph with **bold** and `code`.\n"

	tests := []struct {
		name      string
		data      []byte
		wantErrIs error
		wantText  string
	}{
		{name: "valid markdown", data: []byte(sample), wantText: sample},
		{name: "ascii body", data: []byte("hello world"), wantText: "hello world"},
		{name: "utf-8 cyrillic", data: []byte("Привет, мир"), wantText: "Привет, мир"},
		{name: "empty bytes", data: nil, wantErrIs: reader.ErrEmptyContent},
		{name: "zero-length bytes", data: []byte{}, wantErrIs: reader.ErrEmptyContent},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			text, err := r.Parse(t.Context(), "doc.md", tt.data)
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
				t.Errorf("Parse text = %q, want %q", text, tt.wantText)
			}
		})
	}
}

func TestMDReader_Parse_RespectsCancelledContext(t *testing.T) {
	r := reader.NewMDReader()
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	if _, err := r.Parse(ctx, "doc.md", []byte("# heading")); !errors.Is(err, context.Canceled) {
		t.Fatalf("Parse err=%v, want errors.Is context.Canceled", err)
	}
}

// Compile-time check: *MDReader satisfies DocumentReader.
var _ reader.DocumentReader = (*reader.MDReader)(nil)
