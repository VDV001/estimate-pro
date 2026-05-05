// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package generator_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/VDV001/estimate-pro/backend/internal/shared/generator"
)

// gotenbergSuccessHandler returns a fake PDF body. Real Gotenberg
// sets Content-Type: application/pdf and a non-trivial body — we
// mimic the same shape so the converter has nothing special to
// detect.
func gotenbergSuccessHandler(pdfBody []byte) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/forms/libreoffice/convert") {
			http.Error(w, "wrong endpoint", http.StatusNotFound)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		ct := r.Header.Get("Content-Type")
		if !strings.HasPrefix(ct, "multipart/form-data") {
			http.Error(w, "expected multipart", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/pdf")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(pdfBody)
	}
}

// TestGotenbergConverter_HappyPath pins the success contract:
// a multipart POST to /forms/libreoffice/convert returns the
// upstream PDF body verbatim.
func TestGotenbergConverter_HappyPath(t *testing.T) {
	want := []byte("%PDF-1.4\n%fake-body\n%%EOF")
	srv := httptest.NewServer(gotenbergSuccessHandler(want))
	defer srv.Close()

	c := generator.NewGotenbergConverter(srv.URL, 5*time.Second)
	got, err := c.Convert(context.Background(), []byte("FAKE-DOCX-BYTES"), "spec.docx")
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("Convert returned %q, want %q", got, want)
	}
}

// TestGotenbergConverter_ServerError_5xx pins the resilience
// contract: when Gotenberg returns 5xx, the converter surfaces
// ErrGotenbergUnavailable so callers can decide retry vs hard-fail.
func TestGotenbergConverter_ServerError_5xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "internal", http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := generator.NewGotenbergConverter(srv.URL, 5*time.Second)
	_, err := c.Convert(context.Background(), []byte("DOCX"), "x.docx")
	if !errors.Is(err, generator.ErrGotenbergUnavailable) {
		t.Fatalf("err=%v, want errors.Is generator.ErrGotenbergUnavailable", err)
	}
}

// TestGotenbergConverter_ClientError_4xx pins the input-error
// contract: 4xx means our DOCX (or filename, or args) is invalid;
// callers can return 400 to the original requester rather than
// retry. Specific sentinel ErrInvalidConversionInput.
func TestGotenbergConverter_ClientError_4xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "bad doc", http.StatusBadRequest)
	}))
	defer srv.Close()

	c := generator.NewGotenbergConverter(srv.URL, 5*time.Second)
	_, err := c.Convert(context.Background(), []byte("DOCX"), "x.docx")
	if !errors.Is(err, generator.ErrInvalidConversionInput) {
		t.Fatalf("err=%v, want errors.Is generator.ErrInvalidConversionInput", err)
	}
}

// TestGotenbergConverter_NetworkUnreachable_ReturnsUnavailable
// covers the dial-failure path: when the URL points at a closed
// port, the converter surfaces ErrGotenbergUnavailable rather
// than leaking *url.Error / *net.OpError.
func TestGotenbergConverter_NetworkUnreachable_ReturnsUnavailable(t *testing.T) {
	// Bind a temp listener and immediately close it — guaranteed
	// "connection refused" on the next dial.
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	addr := srv.URL
	srv.Close()

	c := generator.NewGotenbergConverter(addr, 1*time.Second)
	_, err := c.Convert(context.Background(), []byte("DOCX"), "x.docx")
	if !errors.Is(err, generator.ErrGotenbergUnavailable) {
		t.Fatalf("err=%v, want errors.Is generator.ErrGotenbergUnavailable", err)
	}
}

// TestGotenbergConverter_EmptyDOCX_ReturnsSentinel keeps the API
// safe under degenerate input — empty bytes never reach the wire.
func TestGotenbergConverter_EmptyDOCX_ReturnsSentinel(t *testing.T) {
	c := generator.NewGotenbergConverter("http://example.invalid", 1*time.Second)
	_, err := c.Convert(context.Background(), nil, "x.docx")
	if !errors.Is(err, generator.ErrEmptyTemplate) {
		t.Fatalf("err=%v, want errors.Is generator.ErrEmptyTemplate", err)
	}
}

// TestGotenbergConverter_PassesFilename verifies that the filename
// argument lands in the multipart form so Gotenberg's libreoffice
// engine knows which extension to interpret. Without it, libreoffice
// guesses by content sniffing — works for DOCX, but fails for ODT
// / RTF templates (future formats).
func TestGotenbergConverter_PassesFilename(t *testing.T) {
	var gotFilename string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseMultipartForm(10 << 20)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		for _, fhs := range r.MultipartForm.File {
			for _, fh := range fhs {
				gotFilename = fh.Filename
			}
		}
		w.Header().Set("Content-Type", "application/pdf")
		_, _ = w.Write([]byte("%PDF-1.4"))
	}))
	defer srv.Close()

	c := generator.NewGotenbergConverter(srv.URL, 5*time.Second)
	if _, err := c.Convert(context.Background(), []byte("DOCX"), "report-2026-05-05.docx"); err != nil {
		t.Fatalf("Convert: %v", err)
	}
	if gotFilename != "report-2026-05-05.docx" {
		t.Errorf("filename in multipart=%q, want report-2026-05-05.docx", gotFilename)
	}
}

// drainBody is a tiny helper for the rare test that needs to peek
// at the response body for assertion.
//
//nolint:unused // available for future tests
func drainBody(t *testing.T, body io.Reader) []byte {
	t.Helper()
	data, err := io.ReadAll(body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return data
}
