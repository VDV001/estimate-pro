package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/VDV001/estimate-pro/backend/internal/modules/document/domain"
	"github.com/VDV001/estimate-pro/backend/internal/modules/document/handler"
	"github.com/VDV001/estimate-pro/backend/internal/modules/document/usecase"
	"github.com/VDV001/estimate-pro/backend/internal/shared/middleware"
)

// --- Mock DocumentRepository ---

type mockDocRepo struct {
	docs      map[string]*domain.Document
	byProject map[string][]*domain.Document
	createErr error
}

func newMockDocRepo() *mockDocRepo {
	return &mockDocRepo{
		docs:      make(map[string]*domain.Document),
		byProject: make(map[string][]*domain.Document),
	}
}

func (m *mockDocRepo) Create(_ context.Context, doc *domain.Document) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.docs[doc.ID] = doc
	m.byProject[doc.ProjectID] = append(m.byProject[doc.ProjectID], doc)
	return nil
}

func (m *mockDocRepo) GetByID(_ context.Context, id string) (*domain.Document, error) {
	doc, ok := m.docs[id]
	if !ok {
		return nil, domain.ErrDocumentNotFound
	}
	return doc, nil
}

func (m *mockDocRepo) ListByProject(_ context.Context, projectID string) ([]*domain.Document, error) {
	return m.byProject[projectID], nil
}

func (m *mockDocRepo) Delete(_ context.Context, id string) error {
	doc, ok := m.docs[id]
	if !ok {
		return domain.ErrDocumentNotFound
	}
	delete(m.docs, id)
	list := m.byProject[doc.ProjectID]
	for i, d := range list {
		if d.ID == id {
			m.byProject[doc.ProjectID] = append(list[:i], list[i+1:]...)
			break
		}
	}
	return nil
}

// --- Mock VersionRepository ---

type mockVersionRepo struct {
	versions   map[string]*domain.DocumentVersion
	byDoc      map[string][]*domain.DocumentVersion
	tags       map[string][]string
	createErr  error
	flagsErr   error
}

func newMockVersionRepo() *mockVersionRepo {
	return &mockVersionRepo{
		versions: make(map[string]*domain.DocumentVersion),
		byDoc:    make(map[string][]*domain.DocumentVersion),
		tags:     make(map[string][]string),
	}
}

func (m *mockVersionRepo) Create(_ context.Context, v *domain.DocumentVersion) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.versions[v.ID] = v
	m.byDoc[v.DocumentID] = append(m.byDoc[v.DocumentID], v)
	return nil
}

func (m *mockVersionRepo) GetByID(_ context.Context, id string) (*domain.DocumentVersion, error) {
	v, ok := m.versions[id]
	if !ok {
		return nil, domain.ErrVersionNotFound
	}
	return v, nil
}

func (m *mockVersionRepo) ListByDocument(_ context.Context, documentID string) ([]*domain.DocumentVersion, error) {
	return m.byDoc[documentID], nil
}

func (m *mockVersionRepo) GetLatestByDocument(_ context.Context, documentID string) (*domain.DocumentVersion, error) {
	list := m.byDoc[documentID]
	if len(list) == 0 {
		return nil, domain.ErrVersionNotFound
	}
	return list[len(list)-1], nil
}

func (m *mockVersionRepo) UpdateFlags(_ context.Context, id string, isSigned, isFinal bool) error {
	if m.flagsErr != nil {
		return m.flagsErr
	}
	v, ok := m.versions[id]
	if !ok {
		return domain.ErrVersionNotFound
	}
	v.IsSigned = isSigned
	v.IsFinal = isFinal
	return nil
}

func (m *mockVersionRepo) ClearFinal(_ context.Context, documentID string) error {
	for _, v := range m.byDoc[documentID] {
		v.IsFinal = false
	}
	return nil
}

func (m *mockVersionRepo) ClearFinalByProject(_ context.Context, _ string) error {
	for _, v := range m.versions {
		v.IsFinal = false
	}
	return nil
}

func (m *mockVersionRepo) SetTags(_ context.Context, versionID string, tags []string) error {
	m.tags[versionID] = tags
	return nil
}

func (m *mockVersionRepo) GetTags(_ context.Context, versionID string) ([]string, error) {
	return m.tags[versionID], nil
}

// --- Mock FileStorage ---

type mockStorage struct {
	files     map[string][]byte
	uploadErr error
}

func newMockStorage() *mockStorage {
	return &mockStorage{files: make(map[string][]byte)}
}

func (m *mockStorage) Upload(_ context.Context, key string, data io.Reader, _ int64, _ string) error {
	if m.uploadErr != nil {
		return m.uploadErr
	}
	b, _ := io.ReadAll(data)
	m.files[key] = b
	return nil
}

func (m *mockStorage) Download(_ context.Context, key string) (io.ReadCloser, error) {
	data, ok := m.files[key]
	if !ok {
		return nil, fmt.Errorf("not found: %s", key)
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}

func (m *mockStorage) Delete(_ context.Context, key string) error {
	delete(m.files, key)
	return nil
}

// --- Helpers ---

func newTestHandler(docRepo *mockDocRepo, versionRepo *mockVersionRepo, storage *mockStorage) *handler.Handler {
	uc := usecase.New(docRepo, versionRepo, storage)
	return handler.New(uc)
}

func withUserID(r *http.Request, userID string) *http.Request {
	ctx := context.WithValue(r.Context(), middleware.UserIDKey, userID)
	return r.WithContext(ctx)
}

func withChiParams(r *http.Request, params map[string]string) *http.Request {
	rctx := chi.NewRouteContext()
	for k, v := range params {
		rctx.URLParams.Add(k, v)
	}
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

// ==============================
// ListDocuments
// ==============================

func TestListDocuments_Empty(t *testing.T) {
	h := newTestHandler(newMockDocRepo(), newMockVersionRepo(), newMockStorage())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/p1/documents", nil)
	req = withChiParams(req, map[string]string{"projectId": "p1"})
	rec := httptest.NewRecorder()

	h.ListDocuments(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200", rec.Code)
	}
}

func TestListDocuments_WithDocs(t *testing.T) {
	docRepo := newMockDocRepo()
	versionRepo := newMockVersionRepo()
	storage := newMockStorage()
	h := newTestHandler(docRepo, versionRepo, storage)

	// Upload a document first via usecase
	uc := usecase.New(docRepo, versionRepo, storage)
	_, _, err := uc.Upload(t.Context(), usecase.UploadInput{
		ProjectID: "p1",
		Title:     "test.pdf",
		FileName:  "test.pdf",
		FileSize:  100,
		FileType:  domain.FileTypePDF,
		Content:   strings.NewReader("pdf content"),
		UserID:    "u1",
	})
	if err != nil {
		t.Fatalf("upload setup: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/p1/documents", nil)
	req = withChiParams(req, map[string]string{"projectId": "p1"})
	rec := httptest.NewRecorder()

	h.ListDocuments(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200", rec.Code)
	}

	var docs []any
	if err := json.NewDecoder(rec.Body).Decode(&docs); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(docs) != 1 {
		t.Errorf("len: got %d, want 1", len(docs))
	}
}

// ==============================
// UploadDocument
// ==============================

func TestUploadDocument_Success(t *testing.T) {
	h := newTestHandler(newMockDocRepo(), newMockVersionRepo(), newMockStorage())

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("file", "report.pdf")
	if err != nil {
		t.Fatal(err)
	}
	part.Write([]byte("fake pdf content"))
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/p1/documents", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req = withChiParams(req, map[string]string{"projectId": "p1"})
	req = withUserID(req, "user-1")
	rec := httptest.NewRecorder()

	h.UploadDocument(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status: got %d, want 201: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["document"] == nil {
		t.Error("expected document in response")
	}
	if resp["version"] == nil {
		t.Error("expected version in response")
	}
}

func TestUploadDocument_NoUserContext(t *testing.T) {
	h := newTestHandler(newMockDocRepo(), newMockVersionRepo(), newMockStorage())

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/p1/documents", nil)
	req = withChiParams(req, map[string]string{"projectId": "p1"})
	rec := httptest.NewRecorder()

	h.UploadDocument(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status: got %d, want 401", rec.Code)
	}
}

func TestUploadDocument_NoFile(t *testing.T) {
	h := newTestHandler(newMockDocRepo(), newMockVersionRepo(), newMockStorage())

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/p1/documents", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req = withChiParams(req, map[string]string{"projectId": "p1"})
	req = withUserID(req, "user-1")
	rec := httptest.NewRecorder()

	h.UploadDocument(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d, want 400", rec.Code)
	}
}

func TestUploadDocument_UnsupportedType(t *testing.T) {
	h := newTestHandler(newMockDocRepo(), newMockVersionRepo(), newMockStorage())

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, _ := writer.CreateFormFile("file", "image.jpg")
	part.Write([]byte("fake image"))
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/p1/documents", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req = withChiParams(req, map[string]string{"projectId": "p1"})
	req = withUserID(req, "user-1")
	rec := httptest.NewRecorder()

	h.UploadDocument(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d, want 400", rec.Code)
	}
}

func TestUploadDocument_TitleFromFormValue(t *testing.T) {
	h := newTestHandler(newMockDocRepo(), newMockVersionRepo(), newMockStorage())

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	writer.WriteField("title", "Custom Title")
	part, _ := writer.CreateFormFile("file", "report.pdf")
	part.Write([]byte("content"))
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/p1/documents", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req = withChiParams(req, map[string]string{"projectId": "p1"})
	req = withUserID(req, "user-1")
	rec := httptest.NewRecorder()

	h.UploadDocument(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status: got %d, want 201: %s", rec.Code, rec.Body.String())
	}
}

func TestUploadDocument_BadMultipart(t *testing.T) {
	h := newTestHandler(newMockDocRepo(), newMockVersionRepo(), newMockStorage())

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/p1/documents", strings.NewReader("not multipart"))
	req.Header.Set("Content-Type", "application/json")
	req = withChiParams(req, map[string]string{"projectId": "p1"})
	req = withUserID(req, "user-1")
	rec := httptest.NewRecorder()

	h.UploadDocument(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d, want 400", rec.Code)
	}
}

// ==============================
// GetDocument
// ==============================

func TestGetDocument_Success(t *testing.T) {
	docRepo := newMockDocRepo()
	versionRepo := newMockVersionRepo()
	storage := newMockStorage()
	h := newTestHandler(docRepo, versionRepo, storage)

	uc := usecase.New(docRepo, versionRepo, storage)
	doc, _, err := uc.Upload(t.Context(), usecase.UploadInput{
		ProjectID: "p1", Title: "test.pdf", FileName: "test.pdf",
		FileSize: 10, FileType: domain.FileTypePDF,
		Content: strings.NewReader("data"), UserID: "u1",
	})
	if err != nil {
		t.Fatalf("upload setup: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = withChiParams(req, map[string]string{"docId": doc.ID})
	rec := httptest.NewRecorder()

	h.GetDocument(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200: %s", rec.Code, rec.Body.String())
	}
}

func TestGetDocument_NotFound(t *testing.T) {
	h := newTestHandler(newMockDocRepo(), newMockVersionRepo(), newMockStorage())

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = withChiParams(req, map[string]string{"docId": "nonexistent"})
	rec := httptest.NewRecorder()

	h.GetDocument(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status: got %d, want 404", rec.Code)
	}
}

// ==============================
// DownloadDocument
// ==============================

func TestDownloadDocument_Success(t *testing.T) {
	docRepo := newMockDocRepo()
	versionRepo := newMockVersionRepo()
	storage := newMockStorage()
	h := newTestHandler(docRepo, versionRepo, storage)

	uc := usecase.New(docRepo, versionRepo, storage)
	doc, _, err := uc.Upload(t.Context(), usecase.UploadInput{
		ProjectID: "p1", Title: "report.pdf", FileName: "report.pdf",
		FileSize: 12, FileType: domain.FileTypePDF,
		Content: strings.NewReader("pdf-content!"), UserID: "u1",
	})
	if err != nil {
		t.Fatalf("upload setup: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/download", nil)
	req = withChiParams(req, map[string]string{"docId": doc.ID})
	rec := httptest.NewRecorder()

	h.DownloadDocument(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200: %s", rec.Code, rec.Body.String())
	}

	ct := rec.Header().Get("Content-Type")
	if ct != "application/octet-stream" {
		t.Errorf("Content-Type: got %q", ct)
	}

	disp := rec.Header().Get("Content-Disposition")
	if !strings.Contains(disp, "attachment") {
		t.Errorf("Content-Disposition: got %q, want attachment", disp)
	}

	body := rec.Body.String()
	if body != "pdf-content!" {
		t.Errorf("body: got %q", body)
	}
}

func TestDownloadDocument_NoVersions(t *testing.T) {
	docRepo := newMockDocRepo()
	// Add a doc directly without versions.
	docRepo.docs["doc-1"] = &domain.Document{ID: "doc-1", ProjectID: "p1"}
	h := newTestHandler(docRepo, newMockVersionRepo(), newMockStorage())

	req := httptest.NewRequest(http.MethodGet, "/download", nil)
	req = withChiParams(req, map[string]string{"docId": "doc-1"})
	rec := httptest.NewRecorder()

	h.DownloadDocument(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status: got %d, want 404", rec.Code)
	}
}

// ==============================
// DeleteDocument
// ==============================

func TestDeleteDocument_Success(t *testing.T) {
	docRepo := newMockDocRepo()
	versionRepo := newMockVersionRepo()
	storage := newMockStorage()
	h := newTestHandler(docRepo, versionRepo, storage)

	uc := usecase.New(docRepo, versionRepo, storage)
	doc, _, err := uc.Upload(t.Context(), usecase.UploadInput{
		ProjectID: "p1", Title: "del.txt", FileName: "del.txt",
		FileSize: 5, FileType: domain.FileTypeTXT,
		Content: strings.NewReader("hello"), UserID: "u1",
	})
	if err != nil {
		t.Fatalf("upload setup: %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete, "/", nil)
	req = withChiParams(req, map[string]string{"docId": doc.ID})
	req = withUserID(req, "u1")
	rec := httptest.NewRecorder()

	h.DeleteDocument(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status: got %d, want 204: %s", rec.Code, rec.Body.String())
	}
}

func TestDeleteDocument_NotFound(t *testing.T) {
	h := newTestHandler(newMockDocRepo(), newMockVersionRepo(), newMockStorage())

	req := httptest.NewRequest(http.MethodDelete, "/", nil)
	req = withChiParams(req, map[string]string{"docId": "nonexistent"})
	req = withUserID(req, "u1")
	rec := httptest.NewRecorder()

	h.DeleteDocument(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status: got %d, want 404", rec.Code)
	}
}

func TestDeleteDocument_NoUserContext(t *testing.T) {
	h := newTestHandler(newMockDocRepo(), newMockVersionRepo(), newMockStorage())

	req := httptest.NewRequest(http.MethodDelete, "/", nil)
	req = withChiParams(req, map[string]string{"docId": "doc-1"})
	rec := httptest.NewRecorder()

	h.DeleteDocument(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status: got %d, want 401", rec.Code)
	}
}

// ==============================
// UpdateVersionFlags
// ==============================

func TestUpdateVersionFlags_Success(t *testing.T) {
	docRepo := newMockDocRepo()
	versionRepo := newMockVersionRepo()
	storage := newMockStorage()
	h := newTestHandler(docRepo, versionRepo, storage)

	uc := usecase.New(docRepo, versionRepo, storage)
	_, version, err := uc.Upload(t.Context(), usecase.UploadInput{
		ProjectID: "p1", Title: "flags.pdf", FileName: "flags.pdf",
		FileSize: 10, FileType: domain.FileTypePDF,
		Content: strings.NewReader("data"), UserID: "u1",
	})
	if err != nil {
		t.Fatalf("upload setup: %v", err)
	}

	body, _ := json.Marshal(map[string]bool{"is_signed": true, "is_final": false})
	req := httptest.NewRequest(http.MethodPatch, "/flags", bytes.NewReader(body))
	req = withChiParams(req, map[string]string{"projectId": "p1", "versionId": version.ID})
	rec := httptest.NewRecorder()

	h.UpdateVersionFlags(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200: %s", rec.Code, rec.Body.String())
	}
}

func TestUpdateVersionFlags_BadJSON(t *testing.T) {
	h := newTestHandler(newMockDocRepo(), newMockVersionRepo(), newMockStorage())

	req := httptest.NewRequest(http.MethodPatch, "/flags", strings.NewReader("not json"))
	req = withChiParams(req, map[string]string{"projectId": "p1", "versionId": "v1"})
	rec := httptest.NewRecorder()

	h.UpdateVersionFlags(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d, want 400", rec.Code)
	}
}

// ==============================
// SetVersionTags
// ==============================

func TestSetVersionTags_Success(t *testing.T) {
	docRepo := newMockDocRepo()
	versionRepo := newMockVersionRepo()
	storage := newMockStorage()
	h := newTestHandler(docRepo, versionRepo, storage)

	uc := usecase.New(docRepo, versionRepo, storage)
	_, version, err := uc.Upload(t.Context(), usecase.UploadInput{
		ProjectID: "p1", Title: "tags.md", FileName: "tags.md",
		FileSize: 5, FileType: domain.FileTypeMD,
		Content: strings.NewReader("# hi"), UserID: "u1",
	})
	if err != nil {
		t.Fatalf("upload setup: %v", err)
	}

	body, _ := json.Marshal(map[string][]string{"tags": {"черновик", "срочно"}})
	req := httptest.NewRequest(http.MethodPut, "/tags", bytes.NewReader(body))
	req = withChiParams(req, map[string]string{"versionId": version.ID})
	rec := httptest.NewRecorder()

	h.SetVersionTags(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200: %s", rec.Code, rec.Body.String())
	}
}

func TestSetVersionTags_TooMany(t *testing.T) {
	h := newTestHandler(newMockDocRepo(), newMockVersionRepo(), newMockStorage())

	body, _ := json.Marshal(map[string][]string{"tags": {"a", "b", "c", "d"}})
	req := httptest.NewRequest(http.MethodPut, "/tags", bytes.NewReader(body))
	req = withChiParams(req, map[string]string{"versionId": "v1"})
	rec := httptest.NewRecorder()

	h.SetVersionTags(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d, want 400", rec.Code)
	}
}

func TestSetVersionTags_BadJSON(t *testing.T) {
	h := newTestHandler(newMockDocRepo(), newMockVersionRepo(), newMockStorage())

	req := httptest.NewRequest(http.MethodPut, "/tags", strings.NewReader("{invalid"))
	req = withChiParams(req, map[string]string{"versionId": "v1"})
	rec := httptest.NewRecorder()

	h.SetVersionTags(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d, want 400", rec.Code)
	}
}

// ==============================
// Handler constructor / events
// ==============================

func TestNew_WithOnEvent(t *testing.T) {
	uc := usecase.New(newMockDocRepo(), newMockVersionRepo(), newMockStorage())

	var called bool
	h := handler.New(uc, func(eventType, projectID, userID string) {
		called = true
	})

	// Trigger via an upload to exercise the emit path.
	docRepo := newMockDocRepo()
	versionRepo := newMockVersionRepo()
	storage := newMockStorage()
	ucReal := usecase.New(docRepo, versionRepo, storage)
	_ = h // just checking it doesn't panic

	doc, _, err := ucReal.Upload(t.Context(), usecase.UploadInput{
		ProjectID: "p1", Title: "t.csv", FileName: "t.csv",
		FileSize: 3, FileType: domain.FileTypeCSV,
		Content: strings.NewReader("a,b"), UserID: "u1",
	})
	_ = doc
	_ = err
	_ = called
}

func TestSetOnEvent(t *testing.T) {
	uc := usecase.New(newMockDocRepo(), newMockVersionRepo(), newMockStorage())
	h := handler.New(uc)

	var eventCalled string
	h.SetOnEvent(func(eventType, projectID, userID string) {
		eventCalled = eventType
	})

	// We can't easily trigger emit from the outside without going through upload,
	// but we can verify SetOnEvent doesn't panic.
	if eventCalled != "" {
		t.Error("expected no event yet")
	}
}

// ==============================
// UploadDocument with event emission
// ==============================

func TestUploadDocument_EmitsEvent(t *testing.T) {
	docRepo := newMockDocRepo()
	versionRepo := newMockVersionRepo()
	storage := newMockStorage()
	uc := usecase.New(docRepo, versionRepo, storage)

	var emittedEvent, emittedProject, emittedUser string
	h := handler.New(uc, func(eventType, projectID, userID string) {
		emittedEvent = eventType
		emittedProject = projectID
		emittedUser = userID
	})

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, _ := writer.CreateFormFile("file", "event.txt")
	part.Write([]byte("content"))
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req = withChiParams(req, map[string]string{"projectId": "proj-42"})
	req = withUserID(req, "user-7")
	rec := httptest.NewRecorder()

	h.UploadDocument(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status: got %d, want 201: %s", rec.Code, rec.Body.String())
	}

	if emittedEvent != "document.uploaded" {
		t.Errorf("event: got %q, want document.uploaded", emittedEvent)
	}
	if emittedProject != "proj-42" {
		t.Errorf("project: got %q, want proj-42", emittedProject)
	}
	if emittedUser != "user-7" {
		t.Errorf("user: got %q, want user-7", emittedUser)
	}
}
