package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/daniilrusanov/estimate-pro/backend/internal/modules/project/domain"
	"github.com/daniilrusanov/estimate-pro/backend/internal/modules/project/usecase"
	sharedErrors "github.com/daniilrusanov/estimate-pro/backend/internal/shared/errors"
	"github.com/daniilrusanov/estimate-pro/backend/internal/shared/middleware"
	"github.com/daniilrusanov/estimate-pro/backend/internal/shared/pagination"
	"github.com/daniilrusanov/estimate-pro/backend/pkg/jwt"
)

type Handler struct {
	uc            *usecase.ProjectUsecase
	memberUC      *usecase.MemberUsecase
	workspaceRepo domain.WorkspaceRepository
}

func New(uc *usecase.ProjectUsecase, memberUC *usecase.MemberUsecase, workspaceRepo domain.WorkspaceRepository) *Handler {
	return &Handler{uc: uc, memberUC: memberUC, workspaceRepo: workspaceRepo}
}

func (h *Handler) Register(r chi.Router, jwtService *jwt.Service) {
	r.Route("/api/v1", func(r chi.Router) {
		r.Use(middleware.Auth(jwtService))

		r.Route("/workspaces", func(r chi.Router) {
			r.Get("/", h.ListWorkspaces)
			r.Post("/", h.CreateWorkspace)
		})

		r.Route("/projects", func(r chi.Router) {
			r.Get("/", h.ListProjects)
			r.Post("/", h.CreateProject)
			r.Route("/{id}", func(r chi.Router) {
				r.Get("/", h.GetProject)
				r.Patch("/", h.UpdateProject)
				r.Delete("/", h.DeleteProject)
				r.Post("/restore", h.RestoreProject)

				r.Get("/members", h.ListMembers)
				r.Post("/members", h.AddMember)
				r.Delete("/members/{userId}", h.RemoveMember)
			})
		})
	})
}

func (h *Handler) ListWorkspaces(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		sharedErrors.Unauthorized(w, "missing user context")
		return
	}

	workspaces, err := h.workspaceRepo.ListByUser(r.Context(), userID)
	if err != nil {
		sharedErrors.InternalError(w, "failed to list workspaces")
		return
	}

	writeJSON(w, http.StatusOK, workspaces)
}

func (h *Handler) CreateWorkspace(w http.ResponseWriter, r *http.Request) {
	sharedErrors.InternalError(w, "not implemented")
}

type createProjectRequest struct {
	WorkspaceID string `json:"workspace_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (h *Handler) CreateProject(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		sharedErrors.Unauthorized(w, "missing user context")
		return
	}

	var req createProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sharedErrors.BadRequest(w, "invalid request body")
		return
	}

	if req.WorkspaceID == "" || req.Name == "" {
		sharedErrors.BadRequest(w, "workspace_id and name are required")
		return
	}

	project, err := h.uc.Create(r.Context(), usecase.CreateProjectInput{
		WorkspaceID: req.WorkspaceID,
		Name:        req.Name,
		Description: req.Description,
		UserID:      userID,
	})
	if err != nil {
		sharedErrors.InternalError(w, "failed to create project")
		return
	}

	writeJSON(w, http.StatusCreated, project)
}

type projectListResponse struct {
	Projects []*domain.Project `json:"projects"`
	Meta     pagination.Meta   `json:"meta"`
}

func (h *Handler) ListProjects(w http.ResponseWriter, r *http.Request) {
	workspaceID := r.URL.Query().Get("workspace_id")
	if workspaceID == "" {
		sharedErrors.BadRequest(w, "workspace_id query param required")
		return
	}

	p := pagination.Parse(r)

	result, err := h.uc.List(r.Context(), usecase.ListProjectsInput{
		WorkspaceID: workspaceID,
		Limit:       p.Limit,
		Offset:      p.Offset(),
	})
	if err != nil {
		sharedErrors.InternalError(w, "failed to list projects")
		return
	}

	writeJSON(w, http.StatusOK, projectListResponse{
		Projects: result.Projects,
		Meta: pagination.Meta{
			Total: result.Total,
			Page:  p.Page,
			Limit: p.Limit,
		},
	})
}

func (h *Handler) GetProject(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	project, err := h.uc.GetByID(r.Context(), id)
	if err != nil {
		sharedErrors.NotFound(w, "project not found")
		return
	}

	writeJSON(w, http.StatusOK, project)
}

type updateProjectRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (h *Handler) UpdateProject(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		sharedErrors.Unauthorized(w, "missing user context")
		return
	}

	var req updateProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sharedErrors.BadRequest(w, "invalid request body")
		return
	}

	project, err := h.uc.Update(r.Context(), usecase.UpdateProjectInput{
		ID:          id,
		Name:        req.Name,
		Description: req.Description,
		UserID:      userID,
	})
	if err != nil {
		sharedErrors.InternalError(w, "failed to update project")
		return
	}

	writeJSON(w, http.StatusOK, project)
}

func (h *Handler) DeleteProject(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		sharedErrors.Unauthorized(w, "missing user context")
		return
	}

	project, err := h.uc.Archive(r.Context(), id, userID)
	if err != nil {
		sharedErrors.InternalError(w, "failed to archive project")
		return
	}

	writeJSON(w, http.StatusOK, project)
}

func (h *Handler) RestoreProject(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		sharedErrors.Unauthorized(w, "missing user context")
		return
	}

	project, err := h.uc.Restore(r.Context(), id, userID)
	if err != nil {
		sharedErrors.InternalError(w, "failed to restore project")
		return
	}

	writeJSON(w, http.StatusOK, project)
}

type addMemberRequest struct {
	Email string      `json:"email"`
	Role  domain.Role `json:"role"`
}

func (h *Handler) AddMember(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "id")
	callerID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		sharedErrors.Unauthorized(w, "missing user context")
		return
	}

	var req addMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sharedErrors.BadRequest(w, "invalid request body")
		return
	}
	if req.Email == "" || req.Role == "" {
		sharedErrors.BadRequest(w, "email and role are required")
		return
	}

	err := h.memberUC.AddMemberByEmail(r.Context(), usecase.AddMemberByEmailInput{
		ProjectID: projectID,
		Email:     req.Email,
		Role:      req.Role,
		CallerID:  callerID,
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrInsufficientRole):
			sharedErrors.Forbidden(w, "insufficient role to manage members")
		case errors.Is(err, domain.ErrMemberAlreadyAdded):
			sharedErrors.Conflict(w, "member already added")
		case errors.Is(err, domain.ErrProjectNotFound):
			sharedErrors.NotFound(w, "project not found")
		default:
			sharedErrors.InternalError(w, "failed to add member")
		}
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{"status": "ok"})
}

func (h *Handler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "id")
	targetUserID := chi.URLParam(r, "userId")
	callerID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		sharedErrors.Unauthorized(w, "missing user context")
		return
	}

	err := h.memberUC.RemoveMember(r.Context(), usecase.RemoveMemberInput{
		ProjectID: projectID,
		UserID:    targetUserID,
		CallerID:  callerID,
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrInsufficientRole):
			sharedErrors.Forbidden(w, "insufficient role")
		case errors.Is(err, domain.ErrMemberNotFound):
			sharedErrors.NotFound(w, "member not found")
		case errors.Is(err, usecase.ErrLastAdmin):
			sharedErrors.BadRequest(w, "cannot remove last admin")
		default:
			sharedErrors.InternalError(w, "failed to remove member")
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) ListMembers(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "id")

	members, err := h.memberUC.ListMembersWithUsers(r.Context(), projectID)
	if err != nil {
		sharedErrors.InternalError(w, "failed to list members")
		return
	}

	writeJSON(w, http.StatusOK, members)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
