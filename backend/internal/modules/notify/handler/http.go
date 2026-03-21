// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/VDV001/estimate-pro/backend/internal/modules/notify/domain"
	"github.com/VDV001/estimate-pro/backend/internal/modules/notify/usecase"
	sharedErrors "github.com/VDV001/estimate-pro/backend/internal/shared/errors"
	"github.com/VDV001/estimate-pro/backend/internal/shared/middleware"
	"github.com/VDV001/estimate-pro/backend/internal/shared/pagination"
	"github.com/VDV001/estimate-pro/backend/internal/shared/response"
	"github.com/VDV001/estimate-pro/backend/pkg/jwt"
)

type Handler struct {
	uc *usecase.NotificationUsecase
}

func New(uc *usecase.NotificationUsecase) *Handler {
	return &Handler{uc: uc}
}

func (h *Handler) Register(r chi.Router, jwtService *jwt.Service) {
	r.Route("/api/v1/notifications", func(r chi.Router) {
		r.Use(middleware.Auth(jwtService))
		r.Get("/", h.List)
		r.Get("/unread-count", h.UnreadCount)
		r.Patch("/read-all", h.MarkAllRead)
		r.Patch("/{id}/read", h.MarkRead)
		r.Get("/preferences", h.GetPreferences)
		r.Put("/preferences", h.SetPreference)
	})
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		sharedErrors.Unauthorized(w, "missing user context")
		return
	}

	p := pagination.Parse(r)
	notifications, total, err := h.uc.List(r.Context(), userID, p.Limit, p.Offset())
	if err != nil {
		sharedErrors.InternalError(w, "failed to list notifications")
		return
	}

	response.WriteJSON(w, http.StatusOK, map[string]any{
		"notifications": notifications,
		"meta":          pagination.Meta{Total: total, Page: p.Page, Limit: p.Limit},
	})
}

func (h *Handler) UnreadCount(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		sharedErrors.Unauthorized(w, "missing user context")
		return
	}

	count, err := h.uc.CountUnread(r.Context(), userID)
	if err != nil {
		sharedErrors.InternalError(w, "failed to count unread")
		return
	}

	response.WriteJSON(w, http.StatusOK, map[string]int{"count": count})
}

func (h *Handler) MarkRead(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		sharedErrors.Unauthorized(w, "missing user context")
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		sharedErrors.BadRequest(w, "missing notification id")
		return
	}

	err := h.uc.MarkRead(r.Context(), userID, id)
	if errors.Is(err, domain.ErrNotificationNotFound) {
		sharedErrors.NotFound(w, "notification not found")
		return
	}
	if err != nil {
		sharedErrors.InternalError(w, "failed to mark read")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) MarkAllRead(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		sharedErrors.Unauthorized(w, "missing user context")
		return
	}

	if err := h.uc.MarkAllRead(r.Context(), userID); err != nil {
		sharedErrors.InternalError(w, "failed to mark all read")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) GetPreferences(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		sharedErrors.Unauthorized(w, "missing user context")
		return
	}

	prefs, err := h.uc.GetPreferences(r.Context(), userID)
	if err != nil {
		sharedErrors.InternalError(w, "failed to get preferences")
		return
	}

	response.WriteJSON(w, http.StatusOK, map[string]any{"preferences": prefs})
}

type setPreferenceRequest struct {
	Channel string `json:"channel"`
	Enabled bool   `json:"enabled"`
}

func (h *Handler) SetPreference(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		sharedErrors.Unauthorized(w, "missing user context")
		return
	}

	var req setPreferenceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sharedErrors.BadRequest(w, "invalid request body")
		return
	}

	err := h.uc.SetPreference(r.Context(), userID, domain.Channel(req.Channel), req.Enabled)
	if errors.Is(err, domain.ErrInvalidChannel) {
		sharedErrors.BadRequest(w, "invalid channel")
		return
	}
	if err != nil {
		sharedErrors.InternalError(w, "failed to set preference")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
