// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/VDV001/estimate-pro/backend/internal/modules/document/domain"
	"github.com/VDV001/estimate-pro/backend/internal/modules/document/usecase"
	sharedErrors "github.com/VDV001/estimate-pro/backend/internal/shared/errors"
	"github.com/VDV001/estimate-pro/backend/internal/shared/middleware"
	"github.com/VDV001/estimate-pro/backend/internal/shared/response"
	"github.com/VDV001/estimate-pro/backend/pkg/jwt"
)

type OnEvent func(eventType, projectID, userID string)

type Handler struct {
	uc      *usecase.DocumentUsecase
	onEvent OnEvent
}

func New(uc *usecase.DocumentUsecase, onEvent ...OnEvent) *Handler {
	h := &Handler{uc: uc}
	if len(onEvent) > 0 {
		h.onEvent = onEvent[0]
	}
	return h
}

func (h *Handler) SetOnEvent(fn OnEvent) { h.onEvent = fn }

func (h *Handler) emit(eventType, projectID, userID string) {
	if h.onEvent != nil {
		h.onEvent(eventType, projectID, userID)
	}
}

func (h *Handler) Register(r chi.Router, jwtService *jwt.Service, membershipMW ...func(http.Handler) http.Handler) {
	r.Route("/api/v1/projects/{projectId}/documents", func(r chi.Router) {
		r.Use(middleware.Auth(jwtService))
		if len(membershipMW) > 0 {
			r.Use(membershipMW[0])
		}

		r.Get("/", h.ListDocuments)
		r.Post("/", h.UploadDocument)
		r.Route("/{docId}", func(r chi.Router) {
			r.Get("/", h.GetDocument)
			r.Get("/download", h.DownloadDocument)
			r.Delete("/", h.DeleteDocument)
			r.Patch("/versions/{versionId}/flags", h.UpdateVersionFlags)
			r.Put("/versions/{versionId}/tags", h.SetVersionTags)
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

	response.WriteJSON(w, http.StatusOK, docs)
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

	h.emit("document.uploaded", projectID, userID)
	response.WriteJSON(w, http.StatusCreated, map[string]any{
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

	response.WriteJSON(w, http.StatusOK, result)
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

type updateFlagsRequest struct {
	IsSigned bool `json:"is_signed"`
	IsFinal  bool `json:"is_final"`
}

func (h *Handler) UpdateVersionFlags(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "projectId")
	versionID := chi.URLParam(r, "versionId")

	var req updateFlagsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sharedErrors.BadRequest(w, "invalid request body")
		return
	}

	if err := h.uc.UpdateVersionFlags(r.Context(), projectID, versionID, req.IsSigned, req.IsFinal); err != nil {
		sharedErrors.InternalError(w, "failed to update version flags")
		return
	}

	response.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type setTagsRequest struct {
	Tags []string `json:"tags"`
}

func (h *Handler) SetVersionTags(w http.ResponseWriter, r *http.Request) {
	versionID := chi.URLParam(r, "versionId")

	var req setTagsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sharedErrors.BadRequest(w, "invalid request body")
		return
	}

	if len(req.Tags) > domain.MaxTagsPerVersion {
		sharedErrors.BadRequest(w, "maximum 3 tags allowed")
		return
	}

	if err := h.uc.SetVersionTags(r.Context(), versionID, req.Tags); err != nil {
		sharedErrors.InternalError(w, "failed to set tags")
		return
	}

	response.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}


