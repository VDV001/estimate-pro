// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package domain

import "context"

type ProjectRepository interface {
	Create(ctx context.Context, project *Project) error
	GetByID(ctx context.Context, id string) (*Project, error)
	ListByWorkspace(ctx context.Context, workspaceID string, limit, offset int) ([]*Project, int, error)
	ListByUser(ctx context.Context, userID string, limit, offset int) ([]*Project, int, error)
	Update(ctx context.Context, project *Project) error
	Delete(ctx context.Context, id string) error
}

type MemberRepository interface {
	Add(ctx context.Context, member *Member) error
	Remove(ctx context.Context, projectID, userID string) error
	ListByProject(ctx context.Context, projectID string) ([]*Member, error)
	GetRole(ctx context.Context, projectID, userID string) (Role, error)
	ListByProjectWithUsers(ctx context.Context, projectID string) ([]*MemberWithUser, error)
}

type UserFinder interface {
	FindByEmail(ctx context.Context, email string) (userID string, err error)
}

type WorkspaceRepository interface {
	Create(ctx context.Context, workspace *Workspace) error
	GetByID(ctx context.Context, id string) (*Workspace, error)
	ListByUser(ctx context.Context, userID string) ([]*Workspace, error)
}
