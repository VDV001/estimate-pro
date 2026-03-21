// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package domain

import "context"

type EstimationRepository interface {
	Create(ctx context.Context, estimation *Estimation) error
	GetByID(ctx context.Context, id string) (*Estimation, error)
	ListByProject(ctx context.Context, projectID string) ([]*Estimation, error)
	UpdateStatus(ctx context.Context, id string, status Status) error
	Delete(ctx context.Context, id string) error
}

type ItemRepository interface {
	CreateBatch(ctx context.Context, items []*EstimationItem) error
	ListByEstimation(ctx context.Context, estimationID string) ([]*EstimationItem, error)
	DeleteByEstimation(ctx context.Context, estimationID string) error
}
