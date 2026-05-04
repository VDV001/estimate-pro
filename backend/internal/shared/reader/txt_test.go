package reader_test

import (
	"context"
	"errors"
	"testing"

	"github.com/VDV001/estimate-pro/backend/internal/shared/reader"
)

func TestTXTReader_Supports(t *testing.T) {
	r := reader.NewTXTReader()
	tests := []struct {
		name string
		ft   reader.FileType
		want bool
	}{
		{name: "TXT", ft: reader.FileTypeTXT, want: true},
		{name: "MD", ft: reader.FileTypeMD, want: false},
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

func TestTXTReader_Parse(t *testing.T) {
	r := reader.NewTXTReader()

	tests := []struct {
		name      string
		data      []byte
		wantErrIs error
		wantText  string
	}{
		{name: "ascii body", data: []byte("hello world"), wantText: "hello world"},
		{name: "utf-8 cyrillic", data: []byte("Привет, мир"), wantText: "Привет, мир"},
		{name: "utf-8 emoji", data: []byte("ok 👍"), wantText: "ok 👍"},
		{name: "empty bytes", data: nil, wantErrIs: reader.ErrEmptyContent},
		{name: "zero-length bytes", data: []byte{}, wantErrIs: reader.ErrEmptyContent},
		{name: "invalid utf-8 lone continuation", data: []byte{0xc3, 0x28}, wantErrIs: reader.ErrCorruptedFile},
		{name: "invalid utf-8 random bytes", data: []byte{0xff, 0xfe, 0x00, 0x80}, wantErrIs: reader.ErrCorruptedFile},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			text, err := r.Parse(t.Context(), "doc.txt", tt.data)
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

func TestTXTReader_Parse_RespectsCancelledContext(t *testing.T) {
	r := reader.NewTXTReader()
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	if _, err := r.Parse(ctx, "doc.txt", []byte("hi")); !errors.Is(err, context.Canceled) {
		t.Fatalf("Parse err=%v, want errors.Is context.Canceled", err)
	}
}

// Compile-time check: *TXTReader satisfies DocumentReader.
var _ reader.DocumentReader = (*reader.TXTReader)(nil)
