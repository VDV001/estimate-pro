package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/daniilrusanov/estimate-pro/backend/internal/modules/document/domain"
	"github.com/daniilrusanov/estimate-pro/backend/internal/modules/document/usecase"
	sharedErrors "github.com/daniilrusanov/estimate-pro/backend/internal/shared/errors"
	"github.com/daniilrusanov/estimate-pro/backend/internal/shared/middleware"
	"github.com/daniilrusanov/estimate-pro/backend/pkg/jwt"
)

type Handler struct {
	uc *usecase.DocumentUsecase
}

func New(uc *usecase.DocumentUsecase) *Handler {
	return &Handler{uc: uc}
}

func (h *Handler) Register(r chi.Router, jwtService *jwt.Service) {
	r.Route("/api/v1/projects/{projectId}/documents", func(r chi.Router) {
		r.Use(middleware.Auth(jwtService))

		r.Get("/", h.ListDocuments)
		r.Post("/", h.UploadDocument)
		r.Route("/{docId}", func(r chi.Router) {
			r.Get("/", h.GetDocument)
			r.Get("/download", h.DownloadDocument)
			r.Delete("/", h.DeleteDocument)
		})
	})
}

func (h *Handler) ListDocuments(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "projectId")

	docs, err := h.uc.List(r.Context(), projectID)
	if err != nil {
		sharedErrors.InternalError(w, "failed to list documents")
		return
	}

	writeJSON(w, http.StatusOK, docs)
}

func (h *Handler) UploadDocument(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "projectId")
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		sharedErrors.Unauthorized(w, "missing user context")
		return
	}

	if err := r.ParseMultipartForm(50 << 20); err != nil {
		sharedErrors.BadRequest(w, "failed to parse multipart form")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		sharedErrors.BadRequest(w, "file field is required")
		return
	}
	defer file.Close()

	ext := strings.TrimPrefix(filepath.Ext(header.Filename), ".")
	fileType := domain.FileType(strings.ToLower(ext))

	title := r.FormValue("title")
	if title == "" {
		title = strings.TrimSuffix(header.Filename, filepath.Ext(header.Filename))
	}

	doc, version, err := h.uc.Upload(r.Context(), usecase.UploadInput{
		ProjectID: projectID,
		Title:     title,
		FileName:  header.Filename,
		FileSize:  header.Size,
		FileType:  fileType,
		Content:   file,
		UserID:    userID,
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrUnsupportedFileType):
			sharedErrors.BadRequest(w, "unsupported file type")
		case errors.Is(err, domain.ErrFileTooLarge):
			sharedErrors.BadRequest(w, "file exceeds maximum allowed size (50MB)")
		default:
			sharedErrors.InternalError(w, "failed to upload document")
		}
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"document": doc,
		"version":  version,
	})
}

func (h *Handler) GetDocument(w http.ResponseWriter, r *http.Request) {
	docID := chi.URLParam(r, "docId")

	result, err := h.uc.Get(r.Context(), docID)
	if err != nil {
		if errors.Is(err, domain.ErrDocumentNotFound) {
			sharedErrors.NotFound(w, "document not found")
			return
		}
		sharedErrors.InternalError(w, "failed to get document")
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) DownloadDocument(w http.ResponseWriter, r *http.Request) {
	docID := chi.URLParam(r, "docId")

	reader, fileKey, err := h.uc.Download(r.Context(), docID)
	if err != nil {
		if errors.Is(err, domain.ErrVersionNotFound) {
			sharedErrors.NotFound(w, "no versions found for document")
			return
		}
		sharedErrors.InternalError(w, "failed to download document")
		return
	}
	defer reader.Close()

	fileName := filepath.Base(fileKey)
	w.Header().Set("Content-Disposition", "attachment; filename=\""+fileName+"\"")
	w.Header().Set("Content-Type", "application/octet-stream")

	if _, err := io.Copy(w, reader); err != nil {
		return
	}
}

func (h *Handler) DeleteDocument(w http.ResponseWriter, r *http.Request) {
	docID := chi.URLParam(r, "docId")
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		sharedErrors.Unauthorized(w, "missing user context")
		return
	}

	if err := h.uc.Delete(r.Context(), docID, userID); err != nil {
		if errors.Is(err, domain.ErrDocumentNotFound) {
			sharedErrors.NotFound(w, "document not found")
			return
		}
		sharedErrors.InternalError(w, "failed to delete document")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

