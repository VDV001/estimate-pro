// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package usecase

import (
	"context"
	"fmt"

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
	Items             []CreateItemInput
}

// CreateItemInput — транспорт из handler в usecase. Не является доменной
// сущностью; валидируется в NewEstimationItem.
type CreateItemInput struct {
	TaskName    string
	MinHours    float64
	LikelyHours float64
	MaxHours    float64
	Note        string
}

func (uc *EstimationUsecase) Create(ctx context.Context, input CreateInput) (*domain.EstimationWithItems, error) {
	if len(input.Items) == 0 {
		return nil, domain.ErrEmptyItems
	}

	est, err := domain.NewEstimation(input.ProjectID, input.UserID, input.DocumentVersionID)
	if err != nil {
		return nil, err
	}

	// Validate items via domain constructor before touching persistence.
	validated := make([]*domain.EstimationItem, 0, len(input.Items))
	for i, raw := range input.Items {
		item, err := domain.NewEstimationItem(raw.TaskName, raw.MinHours, raw.LikelyHours, raw.MaxHours, raw.Note)
		if err != nil {
			return nil, err
		}
		item.AttachTo(est.ID, i)
		validated = append(validated, item)
	}

	if err := uc.estRepo.Create(ctx, est); err != nil {
		return nil, fmt.Errorf("estimation.Create: %w", err)
	}

	if err := uc.itemRepo.CreateBatch(ctx, validated); err != nil {
		return nil, fmt.Errorf("estimation.Create items: %w", err)
	}

	return &domain.EstimationWithItems{Estimation: est, Items: validated}, nil
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

	if err := est.AuthorizeAuthor(userID); err != nil {
		return err
	}

	if err := est.Submit(); err != nil {
		return err
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

	if err := est.AuthorizeAuthor(userID); err != nil {
		return err
	}

	if est.IsSubmitted() {
		return domain.ErrNotDraft
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

