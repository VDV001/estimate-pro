// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/VDV001/estimate-pro/backend/internal/modules/extractor/domain"
	"github.com/VDV001/estimate-pro/backend/internal/modules/extractor/handler"
	"github.com/VDV001/estimate-pro/backend/internal/modules/extractor/usecase"
	"github.com/VDV001/estimate-pro/backend/internal/shared/middleware"
)

// fakeRepo (handler-package copy) — keeps tests local; the
// authoritative repository contract is exercised in
// repository_integration_test.go and the usecase fakeRepo unit
// tests.
type fakeRepo struct {
	extractions map[string]*domain.Extraction
	events      []*domain.ExtractionEvent
	listErr     error
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{extractions: make(map[string]*domain.Extraction)}
}

func (f *fakeRepo) Create(_ context.Context, ext *domain.Extraction) error {
	f.extractions[ext.ID] = ext
	return nil
}

func (f *fakeRepo) GetByID(_ context.Context, id string) (*domain.Extraction, error) {
	e, ok := f.extractions[id]
	if !ok {
		return nil, domain.ErrExtractionNotFound
	}
	return e, nil
}

func (f *fakeRepo) GetActiveByDocumentVersion(_ context.Context, doc, ver string) (*domain.Extraction, error) {
	for _, e := range f.extractions {
		if e.DocumentID == doc && e.DocumentVersionID == ver {
			switch e.Status {
			case domain.StatusPending, domain.StatusProcessing, domain.StatusCompleted:
				return e, nil
			}
		}
	}
	return nil, domain.ErrExtractionNotFound
}

func (f *fakeRepo) UpdateStatus(_ context.Context, ext *domain.Extraction, ev *domain.ExtractionEvent) error {
	if _, ok := f.extractions[ext.ID]; !ok {
		return domain.ErrExtractionNotFound
	}
	f.extractions[ext.ID] = ext
	f.events = append(f.events, ev)
	return nil
}

func (f *fakeRepo) SaveTasks(_ context.Context, id string, tasks []domain.ExtractedTask) error {
	e, ok := f.extractions[id]
	if !ok {
		return domain.ErrExtractionNotFound
	}
	e.Tasks = tasks
	return nil
}

func (f *fakeRepo) GetEvents(_ context.Context, id string) ([]*domain.ExtractionEvent, error) {
	var out []*domain.ExtractionEvent
	for _, ev := range f.events {
		if ev.ExtractionID == id {
			out = append(out, ev)
		}
	}
	return out, nil
}

func (f *fakeRepo) ListByProject(_ context.Context, _ string) ([]*domain.Extraction, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	out := make([]*domain.Extraction, 0, len(f.extractions))
	for _, e := range f.extractions {
		out = append(out, e)
	}
	return out, nil
}

// helpers

func newTestHandler(t *testing.T) (*handler.Handler, *fakeRepo) {
	t.Helper()
	repo := newFakeRepo()
	uc := usecase.NewExtractor(repo, 0)
	return handler.New(uc), repo
}

func newRouter(h *handler.Handler) chi.Router {
	r := chi.NewRouter()
	// Inject auth context on every request — bypasses JWT middleware
	// for handler-only tests; full JWT wiring is exercised in
	// auth/middleware tests.
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ctx := context.WithValue(req.Context(), middleware.UserIDKey, "user-test")
			next.ServeHTTP(w, req.WithContext(ctx))
		})
	})
	h.RegisterRoutes(r)
	return r
}

func mustNewExtraction(t *testing.T, docID, versionID string) *domain.Extraction {
	t.Helper()
	ext, err := domain.NewExtraction(docID, versionID)
	if err != nil {
		t.Fatalf("NewExtraction: %v", err)
	}
	return ext
}

// ---------- RequestExtraction ----------

func TestHandler_RequestExtraction_Created(t *testing.T) {
	h, repo := newTestHandler(t)
	r := newRouter(h)

	body := strings.NewReader(`{"file_size": 1024}`)
	req := httptest.NewRequest(http.MethodPost,
		"/api/v1/projects/proj-1/documents/doc-1/versions/ver-1/extractions", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("status=%d, want 201; body=%s", rr.Code, rr.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["status"] != "pending" {
		t.Errorf("status=%v, want pending", resp["status"])
	}
	if resp["document_id"] != "doc-1" {
		t.Errorf("document_id=%v, want doc-1", resp["document_id"])
	}
	if len(repo.extractions) != 1 {
		t.Errorf("repo count=%d, want 1", len(repo.extractions))
	}
}

func TestHandler_RequestExtraction_BadBody(t *testing.T) {
	h, _ := newTestHandler(t)
	r := newRouter(h)

	req := httptest.NewRequest(http.MethodPost,
		"/api/v1/projects/proj-1/documents/doc-1/versions/ver-1/extractions",
		strings.NewReader(`not json`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status=%d, want 400", rr.Code)
	}
}

func TestHandler_RequestExtraction_PayloadTooLarge(t *testing.T) {
	repo := newFakeRepo()
	const maxBytes int64 = 1024
	uc := usecase.NewExtractor(repo, maxBytes)
	h := handler.New(uc)
	r := newRouter(h)

	body := bytes.NewBufferString(`{"file_size": 99999}`)
	req := httptest.NewRequest(http.MethodPost,
		"/api/v1/projects/proj-1/documents/doc-1/versions/ver-1/extractions", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status=%d, want 413; body=%s", rr.Code, rr.Body.String())
	}
}

// ---------- GetExtraction ----------

func TestHandler_GetExtraction_OK(t *testing.T) {
	h, repo := newTestHandler(t)
	r := newRouter(h)

	ext := mustNewExtraction(t, "doc-1", "ver-1")
	repo.extractions[ext.ID] = ext

	req := httptest.NewRequest(http.MethodGet, "/api/v1/extractions/"+ext.ID, nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d, want 200; body=%s", rr.Code, rr.Body.String())
	}
	var resp struct {
		Extraction map[string]any   `json:"extraction"`
		Events     []map[string]any `json:"events"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Extraction["id"] != ext.ID {
		t.Errorf("id=%v, want %s", resp.Extraction["id"], ext.ID)
	}
}

func TestHandler_GetExtraction_NotFound(t *testing.T) {
	h, _ := newTestHandler(t)
	r := newRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/extractions/ghost", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("status=%d, want 404", rr.Code)
	}
}

// ---------- Cancel / Retry ----------

func TestHandler_CancelExtraction_NoContent(t *testing.T) {
	h, repo := newTestHandler(t)
	r := newRouter(h)

	ext := mustNewExtraction(t, "doc-1", "ver-1")
	repo.extractions[ext.ID] = ext

	req := httptest.NewRequest(http.MethodPost, "/api/v1/extractions/"+ext.ID+"/cancel", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("status=%d, want 204; body=%s", rr.Code, rr.Body.String())
	}
	if ext.Status != domain.StatusCancelled {
		t.Errorf("Status=%q, want cancelled", ext.Status)
	}
}

func TestHandler_CancelExtraction_Conflict(t *testing.T) {
	h, repo := newTestHandler(t)
	r := newRouter(h)

	ext := mustNewExtraction(t, "doc-1", "ver-1")
	_ = ext.MarkProcessing()
	_ = ext.MarkCompleted(nil)
	repo.extractions[ext.ID] = ext

	req := httptest.NewRequest(http.MethodPost, "/api/v1/extractions/"+ext.ID+"/cancel", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusConflict {
		t.Errorf("status=%d, want 409", rr.Code)
	}
}

func TestHandler_RetryExtraction_NoContent(t *testing.T) {
	h, repo := newTestHandler(t)
	r := newRouter(h)

	ext := mustNewExtraction(t, "doc-1", "ver-1")
	_ = ext.MarkProcessing()
	_ = ext.MarkFailed("transient")
	repo.extractions[ext.ID] = ext

	req := httptest.NewRequest(http.MethodPost, "/api/v1/extractions/"+ext.ID+"/retry", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Errorf("status=%d, want 204; body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandler_RetryExtraction_NotFound(t *testing.T) {
	h, _ := newTestHandler(t)
	r := newRouter(h)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/extractions/ghost/retry", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("status=%d, want 404", rr.Code)
	}
}

// ---------- ListByProject ----------

func TestHandler_ListByProject_OK(t *testing.T) {
	h, repo := newTestHandler(t)
	r := newRouter(h)

	ext1 := mustNewExtraction(t, "doc-1", "ver-1")
	ext2 := mustNewExtraction(t, "doc-2", "ver-2")
	repo.extractions[ext1.ID] = ext1
	repo.extractions[ext2.ID] = ext2

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/proj-1/extractions", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d, want 200; body=%s", rr.Code, rr.Body.String())
	}
	var dtos []map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &dtos); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(dtos) != 2 {
		t.Errorf("len=%d, want 2", len(dtos))
	}
}
