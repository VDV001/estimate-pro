// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package domain

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

const maxProjectNameLen = 255

// NewProject constructs a Project enforcing domain invariants:
// non-empty workspace, non-empty name (1..255 after trim), non-empty creator.
// Status defaults to active. Caller must not bypass this constructor with a
// struct literal outside the domain package.
func NewProject(workspaceID, name, description, createdBy string) (*Project, error) {
	if workspaceID == "" {
		return nil, ErrMissingWorkspace
	}
	trimmed := strings.TrimSpace(name)
	if trimmed == "" || len(trimmed) > maxProjectNameLen {
		return nil, ErrInvalidProjectName
	}
	if createdBy == "" {
		return nil, ErrMissingCreator
	}
	now := time.Now()
	return &Project{
		ID:          uuid.New().String(),
		WorkspaceID: workspaceID,
		Name:        trimmed,
		Description: description,
		Status:      ProjectStatusActive,
		CreatedBy:   createdBy,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

// UpdateDetails applies partial updates. Empty name is treated as "keep current".
// If a non-empty name is invalid, no field is mutated. UpdatedAt advances on
// successful mutation.
func (p *Project) UpdateDetails(name, description string) error {
	var newName string
	if name != "" {
		trimmed := strings.TrimSpace(name)
		if trimmed == "" || len(trimmed) > maxProjectNameLen {
			return ErrInvalidProjectName
		}
		newName = trimmed
	}
	if newName != "" {
		p.Name = newName
	}
	if description != "" {
		p.Description = description
	}
	p.UpdatedAt = time.Now()
	return nil
}

// Archive marks the project as archived.
func (p *Project) Archive() {
	p.Status = ProjectStatusArchived
	p.UpdatedAt = time.Now()
}

// Restore marks the project as active.
func (p *Project) Restore() {
	p.Status = ProjectStatusActive
	p.UpdatedAt = time.Now()
}

// NewMember constructs a project Member enforcing invariants:
// non-empty project, non-empty user, valid role. AddedBy may be empty for
// system-initiated creation (e.g. initial admin at project creation).
func NewMember(projectID, userID string, role Role, addedBy string) (*Member, error) {
	if projectID == "" {
		return nil, ErrMissingProject
	}
	if userID == "" {
		return nil, ErrMissingUser
	}
	if !role.IsValid() {
		return nil, ErrInvalidRole
	}
	return &Member{
		ProjectID: projectID,
		UserID:    userID,
		Role:      role,
		AddedBy:   addedBy,
		AddedAt:   time.Now(),
	}, nil
}
