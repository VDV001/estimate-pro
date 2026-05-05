// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

// Package handler exposes the report module over HTTP. The single
// route GET /api/v1/projects/{projectId}/report?format=md|pdf|docx
// returns the rendered bytes with the right Content-Type and a
// Content-Disposition header that hints the filename to the
// browser / curl.
package handler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	reportdomain "github.com/VDV001/estimate-pro/backend/internal/modules/report/domain"
	sharedErrors "github.com/VDV001/estimate-pro/backend/internal/shared/errors"
	"github.com/VDV001/estimate-pro/backend/internal/shared/middleware"
	"github.com/VDV001/estimate-pro/backend/pkg/jwt"
)

// ReportRenderer is the use-case-shaped port the handler depends
// on. The concrete *usecase.Reporter satisfies it structurally; the
// interface keeps handler tests free of fake-port boilerplate.
type ReportRenderer interface {
	RenderEstimationReport(ctx context.Context, projectID string, format reportdomain.Format) ([]byte, error)
}

// Handler wires the report use case onto a single chi route.
type Handler struct {
	uc ReportRenderer
}

func New(uc ReportRenderer) *Handler {
	return &Handler{uc: uc}
}

// Register mounts /api/v1/projects/{projectId}/report behind JWT
// auth. The optional membershipMW (project-membership middleware
// from main.go) scopes the route to project members only.
func (h *Handler) Register(r chi.Router, jwtService *jwt.Service, membershipMW ...func(http.Handler) http.Handler) {
	r.Route("/api/v1", func(r chi.Router) {
		r.Use(middleware.Auth(jwtService))
		h.mountAuthed(r, membershipMW...)
	})
}

// RegisterRoutes attaches the route to a router that already has
// auth wired (handler-level integration tests use this path).
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
		r.Get("/report", h.RenderReport)
	})
}

// RenderReport handles GET /api/v1/projects/{projectId}/report.
// Stub returns 501 Not Implemented so the partner test fails.
func (h *Handler) RenderReport(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
	_, _ = w.Write([]byte("RenderReport: not implemented"))
}

// contentTypeFor picks the MIME type for the rendered bytes. The
// content-disposition filename comes from the same format. Both
// helpers stay package-private because no other module needs them.
func contentTypeFor(f reportdomain.Format) string {
	switch f {
	case reportdomain.FormatMD:
		return "text/markdown; charset=utf-8"
	case reportdomain.FormatPDF:
		return "application/pdf"
	case reportdomain.FormatDOCX:
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	}
	return "application/octet-stream"
}

func filenameFor(projectID string, f reportdomain.Format) string {
	return fmt.Sprintf("report-%s.%s", projectID, f)
}

// mapError funnels every domain sentinel onto the right HTTP status.
func (h *Handler) mapError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, reportdomain.ErrInvalidFormat):
		sharedErrors.BadRequest(w, "invalid format")
	case errors.Is(err, reportdomain.ErrEmptyEstimation):
		sharedErrors.Conflict(w, "no submitted estimations to aggregate")
	default:
		slog.Error("report.handler: unmapped error", "error", err)
		sharedErrors.InternalError(w, "internal error")
	}
}
