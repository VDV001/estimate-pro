// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package domain

import "time"

type Role string

const (
	RoleAdmin     Role = "admin"
	RolePM        Role = "pm"
	RoleTechLead  Role = "tech_lead"
	RoleDeveloper Role = "developer"
	RoleObserver  Role = "observer"
)

func (r Role) CanManageMembers() bool {
	return r == RoleAdmin || r == RolePM
}

func (r Role) CanEstimate() bool {
	return r != RoleObserver
}

func (r Role) IsValid() bool {
	switch r {
	case RoleAdmin, RolePM, RoleTechLead, RoleDeveloper, RoleObserver:
		return true
	}
	return false
}

type ProjectStatus string

const (
	ProjectStatusActive   ProjectStatus = "active"
	ProjectStatusArchived ProjectStatus = "archived"
)

type Project struct {
	ID          string        `json:"id"`
	WorkspaceID string        `json:"workspace_id"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Status      ProjectStatus `json:"status"`
	CreatedBy   string        `json:"created_by"`
	CreatedAt   time.Time     `json:"created_at,omitzero"`
	UpdatedAt   time.Time     `json:"updated_at,omitzero"`
}

type Member struct {
	ProjectID string    `json:"project_id"`
	UserID    string    `json:"user_id"`
	Role      Role      `json:"role"`
	AddedBy   string    `json:"added_by,omitempty"`
	AddedAt   time.Time `json:"added_at,omitzero"`
}

type MemberWithUser struct {
	ProjectID string `json:"project_id"`
	UserID    string `json:"user_id"`
	Role      Role   `json:"role"`
	UserName  string `json:"user_name"`
	UserEmail string `json:"user_email"`
}

type Workspace struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	OwnerID   string    `json:"owner_id"`
	CreatedAt time.Time `json:"created_at,omitzero"`
}

type WorkspaceMember struct {
	WorkspaceID string    `json:"workspace_id"`
	UserID      string    `json:"user_id"`
	Role        Role      `json:"role"`
	InvitedAt   time.Time `json:"invited_at,omitzero"`
	JoinedAt    time.Time `json:"joined_at,omitzero"`
}
