// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/VDV001/estimate-pro/backend/internal/modules/estimation/domain"
)

type EstimationUsecase struct {
	estRepo  domain.EstimationRepository
	itemRepo domain.ItemRepository
}

func New(estRepo domain.EstimationRepository, itemRepo domain.ItemRepository) *EstimationUsecase {
	return &EstimationUsecase{estRepo: estRepo, itemRepo: itemRepo}
}

type CreateInput struct {
	ProjectID         string
	DocumentVersionID string
	UserID            string
	Items             []*domain.EstimationItem
}

func (uc *EstimationUsecase) Create(ctx context.Context, input CreateInput) (*domain.EstimationWithItems, error) {
	if len(input.Items) == 0 {
		return nil, fmt.Errorf("estimation.Create: %w", domain.ErrEmptyItems)
	}

	for _, item := range input.Items {
		if err := validateItem(item); err != nil {
			return nil, fmt.Errorf("estimation.Create: %w", err)
		}
	}

	now := time.Now()
	est := &domain.Estimation{
		ID:                uuid.New().String(),
		ProjectID:         input.ProjectID,
		DocumentVersionID: input.DocumentVersionID,
		SubmittedBy:       input.UserID,
		Status:            domain.StatusDraft,
		CreatedAt:         now,
	}

	if err := uc.estRepo.Create(ctx, est); err != nil {
		return nil, fmt.Errorf("estimation.Create: %w", err)
	}

	for _, item := range input.Items {
		item.ID = uuid.New().String()
		item.EstimationID = est.ID
	}

	if err := uc.itemRepo.CreateBatch(ctx, input.Items); err != nil {
		return nil, fmt.Errorf("estimation.Create items: %w", err)
	}

	return &domain.EstimationWithItems{Estimation: est, Items: input.Items}, nil
}

func (uc *EstimationUsecase) GetByID(ctx context.Context, id string) (*domain.EstimationWithItems, error) {
	est, err := uc.estRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("estimation.GetByID: %w", err)
	}

	items, err := uc.itemRepo.ListByEstimation(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("estimation.GetByID items: %w", err)
	}

	return &domain.EstimationWithItems{Estimation: est, Items: items}, nil
}

func (uc *EstimationUsecase) ListByProject(ctx context.Context, projectID string) ([]*domain.Estimation, error) {
	estimations, err := uc.estRepo.ListByProject(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("estimation.ListByProject: %w", err)
	}
	return estimations, nil
}

func (uc *EstimationUsecase) Submit(ctx context.Context, id, userID string) error {
	est, err := uc.estRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("estimation.Submit: %w", err)
	}

	if est.SubmittedBy != userID {
		return fmt.Errorf("estimation.Submit: only the author can submit")
	}

	if est.IsSubmitted() {
		return fmt.Errorf("estimation.Submit: %w", domain.ErrAlreadySubmitted)
	}

	if err := uc.estRepo.UpdateStatus(ctx, id, domain.StatusSubmitted); err != nil {
		return fmt.Errorf("estimation.Submit: %w", err)
	}

	return nil
}

func (uc *EstimationUsecase) Delete(ctx context.Context, id, userID string) error {
	est, err := uc.estRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("estimation.Delete: %w", err)
	}

	if est.SubmittedBy != userID {
		return fmt.Errorf("estimation.Delete: only the author can delete")
	}

	if est.IsSubmitted() {
		return fmt.Errorf("estimation.Delete: %w", domain.ErrNotDraft)
	}

	if err := uc.itemRepo.DeleteByEstimation(ctx, id); err != nil {
		return fmt.Errorf("estimation.Delete items: %w", err)
	}

	if err := uc.estRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("estimation.Delete: %w", err)
	}

	return nil
}

func (uc *EstimationUsecase) GetAggregated(ctx context.Context, projectID string) (*domain.AggregatedResult, error) {
	estimations, err := uc.estRepo.ListByProject(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("estimation.GetAggregated: %w", err)
	}

	var withItems []*domain.EstimationWithItems
	for _, est := range estimations {
		if est.Status != domain.StatusSubmitted {
			continue
		}
		items, err := uc.itemRepo.ListByEstimation(ctx, est.ID)
		if err != nil {
			return nil, fmt.Errorf("estimation.GetAggregated items: %w", err)
		}
		withItems = append(withItems, &domain.EstimationWithItems{Estimation: est, Items: items})
	}

	return domain.Aggregate(withItems), nil
}

func validateItem(item *domain.EstimationItem) error {
	if item.MinHours < 0 || item.LikelyHours < 0 || item.MaxHours < 0 {
		return domain.ErrInvalidHours
	}
	if item.MinHours > item.LikelyHours || item.LikelyHours > item.MaxHours {
		return domain.ErrInvalidHours
	}
	if item.TaskName == "" {
		return fmt.Errorf("task name is required")
	}
	if len(item.TaskName) > 255 {
		return fmt.Errorf("task name too long (max 255)")
	}
	return nil
}
