package reader_test

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/VDV001/estimate-pro/backend/internal/shared/reader"
)

func TestComposite_Supports(t *testing.T) {
	full := reader.NewComposite(0, reader.NewPDFReader(), reader.NewDOCXReader())
	empty := reader.NewComposite(0)

	tests := []struct {
		name string
		c    *reader.Composite
		ft   reader.FileType
		want bool
	}{
		{name: "PDF in full composite", c: full, ft: reader.FileTypePDF, want: true},
		{name: "DOCX in full composite", c: full, ft: reader.FileTypeDOCX, want: true},
		{name: "PDF in empty composite", c: empty, ft: reader.FileTypePDF, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.c.Supports(tt.ft); got != tt.want {
				t.Errorf("Supports(%v)=%v, want %v", tt.ft, got, tt.want)
			}
		})
	}
}

func TestComposite_Parse(t *testing.T) {
	hello, err := os.ReadFile("testdata/hello.pdf")
	if err != nil {
		t.Fatalf("read hello.pdf: %v", err)
	}
	helloDocx, err := os.ReadFile("testdata/hello.docx")
	if err != nil {
		t.Fatalf("read hello.docx: %v", err)
	}
	helloXLSX := buildXLSX(t, map[string][][]string{
		"Sheet1": {{"a", "b"}, {"1", "2"}},
	}, []string{"Sheet1"})

	const tinyMax = int64(100) // smaller than hello.pdf (587 bytes)
	full := reader.NewComposite(0,
		reader.NewPDFReader(),
		reader.NewDOCXReader(),
		reader.NewMDReader(),
		reader.NewTXTReader(),
		reader.NewCSVReader(),
		reader.NewXLSXReader(),
	)
	pdfDocxOnly := reader.NewComposite(0, reader.NewPDFReader(), reader.NewDOCXReader())
	tight := reader.NewComposite(tinyMax, reader.NewPDFReader(), reader.NewDOCXReader())
	emptyComposite := reader.NewComposite(0)

	tests := []struct {
		name        string
		c           *reader.Composite
		filename    string
		data        []byte
		wantErrIs   error
		wantNonZero bool
	}{
		{name: "PDF route", c: full, filename: "doc.pdf", data: hello, wantNonZero: true},
		{name: "PDF route with .PDF uppercase", c: full, filename: "doc.PDF", data: hello, wantNonZero: true},
		{name: "DOCX route", c: full, filename: "doc.docx", data: helloDocx, wantNonZero: true},
		{name: "MD route", c: full, filename: "notes.md", data: []byte("# heading"), wantNonZero: true},
		{name: "TXT route", c: full, filename: "log.txt", data: []byte("hello"), wantNonZero: true},
		{name: "CSV route", c: full, filename: "rows.csv", data: []byte("a,b\n1,2"), wantNonZero: true},
		{name: "XLSX route", c: full, filename: "book.xlsx", data: helloXLSX, wantNonZero: true},
		{name: "oversize fails before dispatch", c: tight, filename: "doc.pdf", data: hello, wantErrIs: reader.ErrFileTooLarge},
		{name: "unsupported extension", c: full, filename: "doc.html", data: []byte("hi"), wantErrIs: reader.ErrUnsupportedFormat},
		{name: "filename without extension", c: full, filename: "noext", data: []byte("hi"), wantErrIs: reader.ErrUnsupportedFormat},
		{name: "empty composite rejects PDF", c: emptyComposite, filename: "doc.pdf", data: hello, wantErrIs: reader.ErrUnsupportedFormat},
		{name: "partial composite rejects MD", c: pdfDocxOnly, filename: "notes.md", data: []byte("# heading"), wantErrIs: reader.ErrUnsupportedFormat},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			text, err := tt.c.Parse(t.Context(), tt.filename, tt.data)
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

func TestComposite_Parse_RespectsCancelledContext(t *testing.T) {
	hello, err := os.ReadFile("testdata/hello.pdf")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	c := reader.NewComposite(0, reader.NewPDFReader(), reader.NewDOCXReader())

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	if _, err := c.Parse(ctx, "hello.pdf", hello); !errors.Is(err, context.Canceled) {
		t.Fatalf("Parse err=%v, want errors.Is context.Canceled", err)
	}
}

// Compile-time check: *Composite satisfies DocumentReader.
var _ reader.DocumentReader = (*reader.Composite)(nil)
