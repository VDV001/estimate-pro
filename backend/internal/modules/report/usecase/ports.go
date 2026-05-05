// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

// Package usecase orchestrates the report module's business flow:
// validate format → fetch project → fetch aggregated estimation →
// build GenerationInput → render. Ports live next to the use case
// per the consumer-side DIP rule; concrete implementations are wired
// in cmd/server/main.go from existing modules (project, estimation)
// and shared/generator.
package usecase

import (
	"context"

	estimationdomain "github.com/VDV001/estimate-pro/backend/internal/modules/estimation/domain"
	projectdomain "github.com/VDV001/estimate-pro/backend/internal/modules/project/domain"
	reportdomain "github.com/VDV001/estimate-pro/backend/internal/modules/report/domain"
	"github.com/VDV001/estimate-pro/backend/internal/shared/generator"
)

// ProjectMetadataReader fetches the project entity for header /
// meta-table content. The use case only reads — write paths stay
// in the project module.
type ProjectMetadataReader interface {
	GetByID(ctx context.Context, id string) (*projectdomain.Project, error)
}

// EstimationAggregator returns the cross-team aggregated PERT view
// for a project. The estimation module already owns this logic;
// this port is the seam that lets the report use case stay free
// of estimation persistence details.
type EstimationAggregator interface {
	GetAggregated(ctx context.Context, projectID string) (*estimationdomain.AggregatedResult, error)
}

// Renderer is a thin facade over shared/generator.Composite so the
// use case stays free of generator's wider surface (FillTemplate,
// ConvertToPDF). The composition root in main.go adapts the
// Composite to this single-method shape.
type Renderer interface {
	Render(ctx context.Context, format reportdomain.Format, input generator.GenerationInput) ([]byte, error)
}
