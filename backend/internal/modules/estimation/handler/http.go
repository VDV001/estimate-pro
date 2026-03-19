package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/daniilrusanov/estimate-pro/backend/internal/modules/estimation/domain"
	"github.com/daniilrusanov/estimate-pro/backend/internal/modules/estimation/usecase"
	sharedErrors "github.com/daniilrusanov/estimate-pro/backend/internal/shared/errors"
	"github.com/daniilrusanov/estimate-pro/backend/internal/shared/middleware"
	"github.com/daniilrusanov/estimate-pro/backend/internal/shared/response"
	"github.com/daniilrusanov/estimate-pro/backend/pkg/jwt"
)

// RoleChecker checks if a user can perform estimation actions in a project.
type RoleChecker interface {
	CanEstimate(ctx context.Context, projectID, userID string) bool
}

// OnEvent is a callback for real-time event broadcasting.
type OnEvent func(eventType, projectID string)

type Handler struct {
	uc          *usecase.EstimationUsecase
	roleChecker RoleChecker
	onEvent     OnEvent
}

func New(uc *usecase.EstimationUsecase, roleChecker RoleChecker, onEvent ...OnEvent) *Handler {
	h := &Handler{uc: uc, roleChecker: roleChecker}
	if len(onEvent) > 0 {
		h.onEvent = onEvent[0]
	}
	return h
}

func (h *Handler) SetOnEvent(fn OnEvent) { h.onEvent = fn }

func (h *Handler) emit(eventType, projectID string) {
	if h.onEvent != nil {
		h.onEvent(eventType, projectID)
	}
}

func (h *Handler) Register(r chi.Router, jwtService *jwt.Service) {
	r.Route("/api/v1/projects/{projectId}/estimations", func(r chi.Router) {
		r.Use(middleware.Auth(jwtService))

		r.Post("/", h.CreateEstimation)
		r.Get("/", h.ListEstimations)
		r.Get("/aggregated", h.GetAggregated)
		r.Route("/{estId}", func(r chi.Router) {
			r.Get("/", h.GetEstimation)
			r.Put("/submit", h.SubmitEstimation)
			r.Delete("/", h.DeleteEstimation)
		})
	})
}

type createEstimationRequest struct {
	DocumentVersionID string             `json:"document_version_id,omitempty"`
	Items             []estimationItemDTO `json:"items"`
}

type estimationItemDTO struct {
	TaskName    string  `json:"task_name"`
	MinHours    float64 `json:"min_hours"`
	LikelyHours float64 `json:"likely_hours"`
	MaxHours    float64 `json:"max_hours"`
	SortOrder   int     `json:"sort_order"`
	Note        string  `json:"note,omitempty"`
}

func (h *Handler) CreateEstimation(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "projectId")
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		sharedErrors.Unauthorized(w, "missing user context")
		return
	}

	// Observer cannot create estimations
	if h.roleChecker != nil && !h.roleChecker.CanEstimate(r.Context(), projectID, userID) {
		sharedErrors.Forbidden(w, "observers cannot create estimations")
		return
	}

	var req createEstimationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sharedErrors.BadRequest(w, "invalid request body")
		return
	}

	items := make([]*domain.EstimationItem, len(req.Items))
	for i, dto := range req.Items {
		items[i] = &domain.EstimationItem{
			TaskName:    dto.TaskName,
			MinHours:    dto.MinHours,
			LikelyHours: dto.LikelyHours,
			MaxHours:    dto.MaxHours,
			SortOrder:   dto.SortOrder,
			Note:        dto.Note,
		}
	}

	result, err := h.uc.Create(r.Context(), usecase.CreateInput{
		ProjectID:         projectID,
		DocumentVersionID: req.DocumentVersionID,
		UserID:            userID,
		Items:             items,
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrEmptyItems):
			sharedErrors.BadRequest(w, "estimation must have at least one item")
		case errors.Is(err, domain.ErrInvalidHours):
			sharedErrors.BadRequest(w, "invalid hours: must be non-negative and min <= likely <= max")
		default:
			slog.Error("failed to create estimation", "error", err)
			sharedErrors.InternalError(w, "failed to create estimation")
		}
		return
	}

	h.emit("estimation.created", projectID)
	response.WriteJSON(w, http.StatusCreated, result)
}

func (h *Handler) ListEstimations(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "projectId")

	estimations, err := h.uc.ListByProject(r.Context(), projectID)
	if err != nil {
		sharedErrors.InternalError(w, "failed to list estimations")
		return
	}

	// Filter by current user if ?mine=true
	if r.URL.Query().Get("mine") == "true" {
		userID, ok := middleware.UserIDFromContext(r.Context())
		if ok {
			filtered := make([]*domain.Estimation, 0, len(estimations))
			for _, e := range estimations {
				if e.SubmittedBy == userID {
					filtered = append(filtered, e)
				}
			}
			estimations = filtered
		}
	}

	response.WriteJSON(w, http.StatusOK, estimations)
}

func (h *Handler) GetEstimation(w http.ResponseWriter, r *http.Request) {
	estID := chi.URLParam(r, "estId")

	result, err := h.uc.GetByID(r.Context(), estID)
	if err != nil {
		if errors.Is(err, domain.ErrEstimationNotFound) {
			sharedErrors.NotFound(w, "estimation not found")
			return
		}
		sharedErrors.InternalError(w, "failed to get estimation")
		return
	}

	response.WriteJSON(w, http.StatusOK, result)
}

func (h *Handler) GetAggregated(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "projectId")

	result, err := h.uc.GetAggregated(r.Context(), projectID)
	if err != nil {
		sharedErrors.InternalError(w, "failed to get aggregated estimations")
		return
	}

	response.WriteJSON(w, http.StatusOK, result)
}

func (h *Handler) SubmitEstimation(w http.ResponseWriter, r *http.Request) {
	estID := chi.URLParam(r, "estId")
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		sharedErrors.Unauthorized(w, "missing user context")
		return
	}

	err := h.uc.Submit(r.Context(), estID, userID)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrEstimationNotFound):
			sharedErrors.NotFound(w, "estimation not found")
		case errors.Is(err, domain.ErrAlreadySubmitted):
			sharedErrors.BadRequest(w, "estimation already submitted")
		default:
			slog.Error("failed to submit estimation", "error", err)
			sharedErrors.InternalError(w, "failed to submit estimation")
		}
		return
	}

	h.emit("estimation.submitted", chi.URLParam(r, "projectId"))
	response.WriteJSON(w, http.StatusOK, map[string]string{"status": "submitted"})
}

func (h *Handler) DeleteEstimation(w http.ResponseWriter, r *http.Request) {
	estID := chi.URLParam(r, "estId")
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		sharedErrors.Unauthorized(w, "missing user context")
		return
	}

	err := h.uc.Delete(r.Context(), estID, userID)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrEstimationNotFound):
			sharedErrors.NotFound(w, "estimation not found")
		case errors.Is(err, domain.ErrNotDraft):
			sharedErrors.BadRequest(w, "only draft estimations can be deleted")
		default:
			slog.Error("failed to delete estimation", "error", err)
			sharedErrors.InternalError(w, "failed to delete estimation")
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

