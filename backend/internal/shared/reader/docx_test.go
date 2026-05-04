package reader_test

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/VDV001/estimate-pro/backend/internal/shared/reader"
)

func TestDOCXReader_Supports(t *testing.T) {
	r := reader.NewDOCXReader()
	tests := []struct {
		name string
		ft   reader.FileType
		want bool
	}{
		{name: "DOCX", ft: reader.FileTypeDOCX, want: true},
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

func TestDOCXReader_Parse(t *testing.T) {
	r := reader.NewDOCXReader()

	hello, err := os.ReadFile("testdata/hello.docx")
	if err != nil {
		t.Fatalf("read fixture hello.docx: %v", err)
	}

	tests := []struct {
		name      string
		filename  string
		data      []byte
		wantErrIs error
		wantText  string // substring required when no error
	}{
		{name: "valid DOCX", filename: "hello.docx", data: hello, wantText: "Test document content"},
		{name: "empty bytes", filename: "empty.docx", data: nil, wantErrIs: reader.ErrEmptyContent},
		{name: "zero-length", filename: "empty.docx", data: []byte{}, wantErrIs: reader.ErrEmptyContent},
		{name: "not a zip", filename: "bad.docx", data: []byte("not a docx"), wantErrIs: reader.ErrCorruptedFile},
		{name: "zip without document.xml", filename: "nodoc.docx", data: zipWithoutDocumentXML(t), wantErrIs: reader.ErrCorruptedFile},
		{name: "zip with broken document.xml", filename: "broken.docx", data: zipWithBrokenXML(t), wantErrIs: reader.ErrCorruptedFile},
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
			if !strings.Contains(text, tt.wantText) {
				t.Errorf("Parse text = %q, want substring %q", text, tt.wantText)
			}
		})
	}
}

func zipWithoutDocumentXML(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	fw, err := w.Create("other.txt")
	if err != nil {
		t.Fatalf("zip create: %v", err)
	}
	if _, err := fw.Write([]byte("hello")); err != nil {
		t.Fatalf("zip write: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("zip close: %v", err)
	}
	return buf.Bytes()
}

func zipWithBrokenXML(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	fw, err := w.Create("word/document.xml")
	if err != nil {
		t.Fatalf("zip create: %v", err)
	}
	if _, err := fw.Write([]byte("<broken xml")); err != nil {
		t.Fatalf("zip write: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("zip close: %v", err)
	}
	return buf.Bytes()
}

func TestDOCXReader_Parse_RespectsCancelledContext(t *testing.T) {
	r := reader.NewDOCXReader()
	hello, err := os.ReadFile("testdata/hello.docx")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	if _, err := r.Parse(ctx, "hello.docx", hello); !errors.Is(err, context.Canceled) {
		t.Fatalf("Parse err=%v, want errors.Is context.Canceled", err)
	}
}

// Compile-time check: DOCXReader satisfies DocumentReader.
var _ reader.DocumentReader = (*reader.DOCXReader)(nil)
