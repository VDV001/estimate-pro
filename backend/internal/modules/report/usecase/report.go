// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package usecase

import (
	"context"
	"errors"

	reportdomain "github.com/VDV001/estimate-pro/backend/internal/modules/report/domain"
)

// Reporter is the use-case entry point. Constructed at the
// composition root with the three ports above.
type Reporter struct {
	projects     ProjectMetadataReader
	aggregator   EstimationAggregator
	renderer     Renderer
}

// NewReporter wires the three collaborators. None may be nil — a
// missing dependency surfaces as a programmer error on the first
// call rather than as a confusing nil pointer panic deep inside
// the rendering path.
func NewReporter(projects ProjectMetadataReader, aggregator EstimationAggregator, renderer Renderer) *Reporter {
	return &Reporter{projects: projects, aggregator: aggregator, renderer: renderer}
}

// RenderEstimationReport produces the rendered report bytes for the
// given project and format. Stub returns an "unimplemented" error
// so the GREEN partner of the RED test makes it pass.
func (r *Reporter) RenderEstimationReport(_ context.Context, _ string, _ reportdomain.Format) ([]byte, error) {
	return nil, errors.New("report.RenderEstimationReport: not implemented")
}
