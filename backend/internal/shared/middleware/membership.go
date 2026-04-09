// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package middleware

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/VDV001/estimate-pro/backend/internal/shared/errors"
)

// RoleGetter checks if a user is a member of a project and returns their role.
type RoleGetter interface {
	GetRole(ctx context.Context, projectID, userID string) (string, error)
}

type roleContextKey string

const ProjectRoleKey roleContextKey = "project_role"

// RequireProjectMember ensures the authenticated user is a member of the project
// identified by the {id} or {projectId} URL param. Returns 403 if not a member.
func RequireProjectMember(roleGetter RoleGetter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, ok := UserIDFromContext(r.Context())
			if !ok {
				errors.Unauthorized(w, "missing user context")
				return
			}

			// Try {projectId} first (estimation routes), then {id} (project routes).
			projectID := chi.URLParam(r, "projectId")
			if projectID == "" {
				projectID = chi.URLParam(r, "id")
			}
			if projectID == "" {
				errors.BadRequest(w, "missing project ID")
				return
			}

			role, err := roleGetter.GetRole(r.Context(), projectID, userID)
			if err != nil {
				errors.Forbidden(w, "not a member of this project")
				return
			}

			ctx := context.WithValue(r.Context(), ProjectRoleKey, role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ProjectRoleFromContext extracts the project role from the context.
func ProjectRoleFromContext(ctx context.Context) (string, bool) {
	role, ok := ctx.Value(ProjectRoleKey).(string)
	return role, ok
}
