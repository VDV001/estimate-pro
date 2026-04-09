// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/VDV001/estimate-pro/backend/internal/modules/project/domain"
)

type ProjectUsecase struct {
	projectRepo   domain.ProjectRepository
	workspaceRepo domain.WorkspaceRepository
	memberRepo    domain.MemberRepository
}

func New(projectRepo domain.ProjectRepository, workspaceRepo domain.WorkspaceRepository, memberRepo domain.MemberRepository) *ProjectUsecase {
	return &ProjectUsecase{projectRepo: projectRepo, workspaceRepo: workspaceRepo, memberRepo: memberRepo}
}

type CreateProjectInput struct {
	WorkspaceID string
	Name        string
	Description string
	UserID      string
}

func (uc *ProjectUsecase) Create(ctx context.Context, input CreateProjectInput) (*domain.Project, error) {
	if _, err := uc.workspaceRepo.GetByID(ctx, input.WorkspaceID); err != nil {
		return nil, fmt.Errorf("project.Create: %w", err)
	}

	now := time.Now()
	project := &domain.Project{
		ID:          uuid.New().String(),
		WorkspaceID: input.WorkspaceID,
		Name:        input.Name,
		Description: input.Description,
		Status:      domain.ProjectStatusActive,
		CreatedBy:   input.UserID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := uc.projectRepo.Create(ctx, project); err != nil {
		return nil, fmt.Errorf("project.Create: %w", err)
	}

	member := &domain.Member{
		ProjectID: project.ID,
		UserID:    input.UserID,
		Role:      domain.RoleAdmin,
	}
	if err := uc.memberRepo.Add(ctx, member); err != nil {
		return nil, fmt.Errorf("project.Create add member: %w", err)
	}

	return project, nil
}

type ListProjectsInput struct {
	WorkspaceID string
	Limit       int
	Offset      int
}

type ListProjectsOutput struct {
	Projects []*domain.Project
	Total    int
}

func (uc *ProjectUsecase) List(ctx context.Context, input ListProjectsInput) (*ListProjectsOutput, error) {
	projects, total, err := uc.projectRepo.ListByWorkspace(ctx, input.WorkspaceID, input.Limit, input.Offset)
	if err != nil {
		return nil, fmt.Errorf("project.List: %w", err)
	}
	return &ListProjectsOutput{Projects: projects, Total: total}, nil
}

type ListByUserInput struct {
	UserID string
	Limit  int
	Offset int
}

func (uc *ProjectUsecase) ListByUser(ctx context.Context, input ListByUserInput) (*ListProjectsOutput, error) {
	projects, total, err := uc.projectRepo.ListByUser(ctx, input.UserID, input.Limit, input.Offset)
	if err != nil {
		return nil, fmt.Errorf("project.ListByUser: %w", err)
	}
	return &ListProjectsOutput{Projects: projects, Total: total}, nil
}

func (uc *ProjectUsecase) GetByID(ctx context.Context, id string) (*domain.Project, error) {
	return uc.projectRepo.GetByID(ctx, id)
}

type UpdateProjectInput struct {
	ID          string
	Name        string
	Description string
	UserID      string
}

func (uc *ProjectUsecase) Update(ctx context.Context, input UpdateProjectInput) (*domain.Project, error) {
	role, err := uc.memberRepo.GetRole(ctx, input.ID, input.UserID)
	if err != nil {
		return nil, fmt.Errorf("project.Update: %w", err)
	}
	if !role.CanManageMembers() {
		return nil, fmt.Errorf("project.Update: %w", domain.ErrInsufficientRole)
	}

	project, err := uc.projectRepo.GetByID(ctx, input.ID)
	if err != nil {
		return nil, fmt.Errorf("project.Update: %w", err)
	}

	if input.Name != "" {
		project.Name = input.Name
	}
	if input.Description != "" {
		project.Description = input.Description
	}
	project.UpdatedAt = time.Now()

	if err := uc.projectRepo.Update(ctx, project); err != nil {
		return nil, fmt.Errorf("project.Update: %w", err)
	}
	return project, nil
}

func (uc *ProjectUsecase) Archive(ctx context.Context, id, userID string) (*domain.Project, error) {
	role, err := uc.memberRepo.GetRole(ctx, id, userID)
	if err != nil {
		return nil, fmt.Errorf("project.Archive: %w", err)
	}
	if !role.CanManageMembers() {
		return nil, fmt.Errorf("project.Archive: %w", domain.ErrInsufficientRole)
	}

	project, err := uc.projectRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("project.Archive: %w", err)
	}

	project.Status = domain.ProjectStatusArchived
	project.UpdatedAt = time.Now()

	if err := uc.projectRepo.Update(ctx, project); err != nil {
		return nil, fmt.Errorf("project.Archive: %w", err)
	}
	return project, nil
}

func (uc *ProjectUsecase) Restore(ctx context.Context, id, userID string) (*domain.Project, error) {
	role, err := uc.memberRepo.GetRole(ctx, id, userID)
	if err != nil {
		return nil, fmt.Errorf("project.Restore: %w", err)
	}
	if !role.CanManageMembers() {
		return nil, fmt.Errorf("project.Restore: %w", domain.ErrInsufficientRole)
	}

	project, err := uc.projectRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("project.Restore: %w", err)
	}

	project.Status = domain.ProjectStatusActive
	project.UpdatedAt = time.Now()

	if err := uc.projectRepo.Update(ctx, project); err != nil {
		return nil, fmt.Errorf("project.Restore: %w", err)
	}
	return project, nil
}
