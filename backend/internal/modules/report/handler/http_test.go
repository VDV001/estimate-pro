// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package handler_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	reportdomain "github.com/VDV001/estimate-pro/backend/internal/modules/report/domain"
	"github.com/VDV001/estimate-pro/backend/internal/modules/report/handler"
)

// stubReporter satisfies handler.ReportRenderer.
type stubReporter struct {
	calls       int
	gotProject  string
	gotFormat   reportdomain.Format
	respBytes   []byte
	respErr     error
}

func (s *stubReporter) RenderEstimationReport(_ context.Context, projectID string, format reportdomain.Format) ([]byte, error) {
	s.calls++
	s.gotProject = projectID
	s.gotFormat = format
	return s.respBytes, s.respErr
}

func newServer(s *stubReporter) *httptest.Server {
	r := chi.NewRouter()
	handler.New(s).RegisterRoutes(r)
	return httptest.NewServer(r)
}

func TestRenderReport_PDF_ReturnsBytesWithProperHeaders(t *testing.T) {
	s := &stubReporter{respBytes: []byte("%PDF-1.4 ...")}
	srv := newServer(s)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/v1/projects/p1/report?format=pdf")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status=%d, want 200", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "%PDF-1.4 ..." {
		t.Errorf("body=%q", body)
	}
	if got := resp.Header.Get("Content-Type"); !strings.HasPrefix(got, "application/pdf") {
		t.Errorf("Content-Type=%q, want application/pdf", got)
	}
	if got := resp.Header.Get("Content-Disposition"); !strings.Contains(got, "report-p1.pdf") {
		t.Errorf("Content-Disposition=%q, want filename report-p1.pdf", got)
	}

	if s.calls != 1 || s.gotProject != "p1" || s.gotFormat != reportdomain.FormatPDF {
		t.Errorf("uc.calls=%d project=%q format=%q", s.calls, s.gotProject, s.gotFormat)
	}
}

func TestRenderReport_MD_DOCX_PickRightContentType(t *testing.T) {
	cases := []struct {
		name        string
		format      string
		wantCT      string
		wantSuffix  string
	}{
		{"md", "md", "text/markdown", "report-p1.md"},
		{"docx", "docx", "application/vnd.openxmlformats-officedocument.wordprocessingml.document", "report-p1.docx"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := &stubReporter{respBytes: []byte("BODY")}
			srv := newServer(s)
			defer srv.Close()

			resp, err := http.Get(fmt.Sprintf("%s/api/v1/projects/p1/report?format=%s", srv.URL, tc.format))
			if err != nil {
				t.Fatalf("GET: %v", err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				t.Errorf("status=%d, want 200", resp.StatusCode)
			}
			if got := resp.Header.Get("Content-Type"); !strings.HasPrefix(got, tc.wantCT) {
				t.Errorf("Content-Type=%q, want prefix %q", got, tc.wantCT)
			}
			if got := resp.Header.Get("Content-Disposition"); !strings.Contains(got, tc.wantSuffix) {
				t.Errorf("Content-Disposition=%q, want %q", got, tc.wantSuffix)
			}
		})
	}
}

func TestRenderReport_DefaultsToPDF_WhenFormatMissing(t *testing.T) {
	s := &stubReporter{respBytes: []byte("%PDF-1.4 ...")}
	srv := newServer(s)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/v1/projects/p1/report")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status=%d, want 200", resp.StatusCode)
	}
	if s.gotFormat != reportdomain.FormatPDF {
		t.Errorf("default format=%q, want pdf", s.gotFormat)
	}
}

func TestRenderReport_400_OnInvalidFormat(t *testing.T) {
	s := &stubReporter{respErr: fmt.Errorf("wrap: %w", reportdomain.ErrInvalidFormat)}
	srv := newServer(s)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/v1/projects/p1/report?format=yaml")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status=%d, want 400", resp.StatusCode)
	}
}

func TestRenderReport_409_OnEmptyEstimation(t *testing.T) {
	s := &stubReporter{respErr: fmt.Errorf("wrap: %w", reportdomain.ErrEmptyEstimation)}
	srv := newServer(s)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/v1/projects/p1/report?format=pdf")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		t.Errorf("status=%d, want 409", resp.StatusCode)
	}
}

func TestRenderReport_500_OnUnknownError(t *testing.T) {
	s := &stubReporter{respErr: errors.New("db: oh no")}
	srv := newServer(s)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/v1/projects/p1/report?format=pdf")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("status=%d, want 500", resp.StatusCode)
	}
}
