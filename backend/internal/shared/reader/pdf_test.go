package reader_test

import (
	"bytes"
	"context"
	"errors"
	"os"
	"testing"

	"github.com/VDV001/estimate-pro/backend/internal/shared/reader"
	pdfcpuapi "github.com/pdfcpu/pdfcpu/pkg/api"
	pdfcpumodel "github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
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

// buildEncryptedPDF wraps testdata/hello.pdf with pdfcpu's AES
// encryption so the test fixture is genuine — ledongthuc/pdf
// will encounter a real /Encrypt dict on read. Mirrors the
// PR-B1 buildXLSX pattern (test-only dep used as fixture
// generator). pdfcpu is Apache-2.0; isolating its import to
// the _test.go file keeps it out of the production binary.
func buildEncryptedPDF(t *testing.T) []byte {
	t.Helper()
	plaintext, err := os.ReadFile("testdata/hello.pdf")
	if err != nil {
		t.Fatalf("read seed fixture hello.pdf: %v", err)
	}

	conf := pdfcpumodel.NewAESConfiguration("user-secret", "owner-secret", 256)
	var buf bytes.Buffer
	if err := pdfcpuapi.Encrypt(bytes.NewReader(plaintext), &buf, conf); err != nil {
		t.Fatalf("pdfcpu encrypt fixture: %v", err)
	}
	return buf.Bytes()
}

// TestPDFReader_EncryptedPDF_ReturnsErrEncryptedFile pins the
// new sentinel: when ledongthuc/pdf encounters an /Encrypt dict
// in the trailer it surfaces an error (typically
// pdf.ErrInvalidPassword for nil-password reads). The reader
// must translate that into reader.ErrEncryptedFile so workers
// (extractor/worker.Process) can match via errors.Is and stamp
// a meaningful failure reason rather than a generic
// ErrCorruptedFile.
//
// Closes #46 (deferred from PR-B1 because qpdf was unavailable
// in dev env to generate the fixture; pdfcpu replaces qpdf as
// a pure-Go test-only dep).
func TestPDFReader_EncryptedPDF_ReturnsErrEncryptedFile(t *testing.T) {
	encrypted := buildEncryptedPDF(t)
	r := reader.NewPDFReader()

	_, err := r.Parse(t.Context(), "secret.pdf", encrypted)
	if !errors.Is(err, reader.ErrEncryptedFile) {
		t.Fatalf("Parse encrypted PDF err=%v, want errors.Is reader.ErrEncryptedFile", err)
	}
	if errors.Is(err, reader.ErrCorruptedFile) {
		t.Fatalf("encrypted PDF should not collapse onto ErrCorruptedFile (operator UX would lose the password-protected signal): %v", err)
	}
}

// Compile-time check: PDFReader satisfies DocumentReader.
var _ reader.DocumentReader = (*reader.PDFReader)(nil)
