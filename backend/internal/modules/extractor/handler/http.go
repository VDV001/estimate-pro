// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

// Package handler exposes the extractor module over HTTP. DTOs live
// here so the domain stays free of transport tags; route wiring is
// in RegisterRoutes (mounted on /api/v1).
package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/VDV001/estimate-pro/backend/internal/modules/extractor/domain"
	"github.com/VDV001/estimate-pro/backend/internal/modules/extractor/usecase"
	sharedErrors "github.com/VDV001/estimate-pro/backend/internal/shared/errors"
	"github.com/VDV001/estimate-pro/backend/internal/shared/middleware"
	"github.com/VDV001/estimate-pro/backend/internal/shared/response"
	"github.com/VDV001/estimate-pro/backend/pkg/jwt"
)

// Handler wires Extractor use-cases onto chi routes.
type Handler struct {
	uc *usecase.Extractor
}

func New(uc *usecase.Extractor) *Handler {
	return &Handler{uc: uc}
}

// Register mounts /api/v1 routes with JWT auth middleware in front of
// every endpoint and an optional project-membership middleware in
// front of project-scoped routes. Mirrors the pattern used by other
// modules' Register methods.
func (h *Handler) Register(r chi.Router, jwtService *jwt.Service, membershipMW ...func(http.Handler) http.Handler) {
	r.Route("/api/v1", func(r chi.Router) {
		r.Use(middleware.Auth(jwtService))
		h.mountAuthed(r, membershipMW...)
	})
}

// RegisterRoutes attaches the routes to a router that already has
// auth handled (handler tests and feature-flag-bypass paths use
// this; the production wiring goes through Register).
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/api/v1", func(r chi.Router) {
		h.mountAuthed(r)
	})
}

func (h *Handler) mountAuthed(r chi.Router, membershipMW ...func(http.Handler) http.Handler) {
	r.Route("/projects/{projectId}", func(r chi.Router) {
		if len(membershipMW) > 0 {
			r.Use(membershipMW[0])
		}
		r.Get("/extractions", h.ListByProject)
		r.Post("/documents/{docId}/versions/{versionId}/extractions", h.RequestExtraction)
	})

	// Extraction-scoped routes are JWT-required; ownership is
	// enforced at the use-case level in PR-B3 once the worker brings
	// project context onto the extraction (TODO).
	r.Route("/extractions/{extractionId}", func(r chi.Router) {
		r.Get("/", h.GetExtraction)
		r.Post("/cancel", h.CancelExtraction)
		r.Post("/retry", h.RetryExtraction)
	})
}

// ---------- DTOs ----------

type extractionDTO struct {
	ID                string             `json:"id"`
	DocumentID        string             `json:"document_id"`
	DocumentVersionID string             `json:"document_version_id"`
	Status            string             `json:"status"`
	Tasks             []extractedTaskDTO `json:"tasks"`
	FailureReason     string             `json:"failure_reason,omitempty"`
	CreatedAt         time.Time          `json:"created_at"`
	UpdatedAt         time.Time          `json:"updated_at"`
	StartedAt         *time.Time         `json:"started_at,omitempty"`
	CompletedAt       *time.Time         `json:"completed_at,omitempty"`
}

type extractedTaskDTO struct {
	Name         string `json:"name"`
	EstimateHint string `json:"estimate_hint,omitempty"`
}

type extractionEventDTO struct {
	ID           string    `json:"id"`
	ExtractionID string    `json:"extraction_id"`
	FromStatus   string    `json:"from_status"`
	ToStatus     string    `json:"to_status"`
	ErrorMessage string    `json:"error_message,omitempty"`
	Actor        string    `json:"actor"`
	CreatedAt    time.Time `json:"created_at"`
}

func toExtractionDTO(e *domain.Extraction) extractionDTO {
	tasks := make([]extractedTaskDTO, len(e.Tasks))
	for i, t := range e.Tasks {
		tasks[i] = extractedTaskDTO{Name: t.Name, EstimateHint: t.EstimateHint}
	}
	return extractionDTO{
		ID:                e.ID,
		DocumentID:        e.DocumentID,
		DocumentVersionID: e.DocumentVersionID,
		Status:            string(e.Status),
		Tasks:             tasks,
		FailureReason:     e.FailureReason,
		CreatedAt:         e.CreatedAt,
		UpdatedAt:         e.UpdatedAt,
		StartedAt:         e.StartedAt,
		CompletedAt:       e.CompletedAt,
	}
}

func toExtractionEventDTO(ev *domain.ExtractionEvent) extractionEventDTO {
	return extractionEventDTO{
		ID:           ev.ID,
		ExtractionID: ev.ExtractionID,
		FromStatus:   string(ev.FromStatus),
		ToStatus:     string(ev.ToStatus),
		ErrorMessage: ev.ErrorMessage,
		Actor:        ev.Actor,
		CreatedAt:    ev.CreatedAt,
	}
}

// ---------- Handlers ----------

type requestExtractionBody struct {
	FileSize int64 `json:"file_size"`
}

func (h *Handler) RequestExtraction(w http.ResponseWriter, r *http.Request) {
	docID := chi.URLParam(r, "docId")
	versionID := chi.URLParam(r, "versionId")
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		sharedErrors.Unauthorized(w, "missing user context")
		return
	}

	var body requestExtractionBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		sharedErrors.BadRequest(w, "invalid request body")
		return
	}

	ext, err := h.uc.RequestExtraction(r.Context(), docID, versionID, body.FileSize, fmt.Sprintf("user:%s", userID))
	if err != nil {
		h.mapError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusCreated, toExtractionDTO(ext))
}

type extractionEnvelope struct {
	Extraction extractionDTO        `json:"extraction"`
	Events     []extractionEventDTO `json:"events"`
}

func (h *Handler) GetExtraction(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "extractionId")
	ext, events, err := h.uc.GetExtraction(r.Context(), id)
	if err != nil {
		h.mapError(w, err)
		return
	}
	eventDTOs := make([]extractionEventDTO, len(events))
	for i, ev := range events {
		eventDTOs[i] = toExtractionEventDTO(ev)
	}
	response.WriteJSON(w, http.StatusOK, extractionEnvelope{
		Extraction: toExtractionDTO(ext),
		Events:     eventDTOs,
	})
}

func (h *Handler) CancelExtraction(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "extractionId")
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		sharedErrors.Unauthorized(w, "missing user context")
		return
	}
	if err := h.uc.CancelExtraction(r.Context(), id, fmt.Sprintf("user:%s", userID)); err != nil {
		h.mapError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) RetryExtraction(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "extractionId")
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		sharedErrors.Unauthorized(w, "missing user context")
		return
	}
	if err := h.uc.RetryExtraction(r.Context(), id, fmt.Sprintf("user:%s", userID)); err != nil {
		h.mapError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) ListByProject(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "projectId")
	extractions, err := h.uc.ListByProject(r.Context(), projectID)
	if err != nil {
		sharedErrors.InternalError(w, "failed to list extractions")
		return
	}
	dtos := make([]extractionDTO, len(extractions))
	for i, e := range extractions {
		dtos[i] = toExtractionDTO(e)
	}
	response.WriteJSON(w, http.StatusOK, dtos)
}

// mapError funnels every domain sentinel onto the right HTTP status
// in one place — keeping handlers free of mapping noise and guarding
// against drift between endpoints.
func (h *Handler) mapError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrExtractionNotFound):
		sharedErrors.NotFound(w, "extraction not found")
	case errors.Is(err, domain.ErrDocumentTooLarge):
		sharedErrors.PayloadTooLarge(w, "document exceeds maximum allowed size")
	case errors.Is(err, domain.ErrInvalidStatusTransition),
		errors.Is(err, domain.ErrAlreadyCompleted):
		sharedErrors.Conflict(w, err.Error())
	case errors.Is(err, domain.ErrMissingDocument),
		errors.Is(err, domain.ErrMissingDocumentVersion):
		sharedErrors.BadRequest(w, err.Error())
	default:
		sharedErrors.InternalError(w, "internal error")
	}
}
