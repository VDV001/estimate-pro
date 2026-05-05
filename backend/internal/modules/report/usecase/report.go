// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package usecase

import (
	"context"
	"fmt"
	"strings"

	estimationdomain "github.com/VDV001/estimate-pro/backend/internal/modules/estimation/domain"
	reportdomain "github.com/VDV001/estimate-pro/backend/internal/modules/report/domain"
	"github.com/VDV001/estimate-pro/backend/internal/shared/generator"
)

// Reporter is the use-case entry point. Constructed at the
// composition root with the three ports.
type Reporter struct {
	projects   ProjectMetadataReader
	aggregator EstimationAggregator
	renderer   Renderer
}

// NewReporter wires the three collaborators. None may be nil — a
// missing dependency surfaces as a programmer error on the first
// call rather than as a confusing nil pointer panic deep inside
// the rendering path.
func NewReporter(projects ProjectMetadataReader, aggregator EstimationAggregator, renderer Renderer) *Reporter {
	return &Reporter{projects: projects, aggregator: aggregator, renderer: renderer}
}

// RenderEstimationReport produces the rendered report bytes for the
// given project and format. Validation order: format first (cheap,
// no I/O), then project, then aggregation. The aggregation must be
// non-empty — an empty PERT view has nothing to render.
func (r *Reporter) RenderEstimationReport(ctx context.Context, projectID string, format reportdomain.Format) ([]byte, error) {
	if !format.IsValid() {
		return nil, fmt.Errorf("%w: %q", reportdomain.ErrInvalidFormat, format)
	}
	project, err := r.projects.GetByID(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("report: load project: %w", err)
	}
	aggregated, err := r.aggregator.GetAggregated(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("report: load aggregated estimation: %w", err)
	}
	if len(aggregated.Items) == 0 {
		return nil, fmt.Errorf("%w: project %s", reportdomain.ErrEmptyEstimation, projectID)
	}

	input := buildGenerationInput(project.Name, aggregated)
	return r.renderer.Render(ctx, format, input)
}

// buildGenerationInput maps the project + aggregated PERT view onto
// the generator's GenerationInput shape. Title carries the project
// name; Meta carries project name + total hours; one Section per
// aggregated task lists the PERT / min / max / estimator count.
func buildGenerationInput(projectName string, agg *estimationdomain.AggregatedResult) generator.GenerationInput {
	sections := make([]generator.GenerationSection, 0, len(agg.Items))
	for _, item := range agg.Items {
		var content strings.Builder
		fmt.Fprintf(&content, "PERT: %.1f h\n", item.AvgPERTHours)
		fmt.Fprintf(&content, "Min: %.1f h\n", item.MinOfMins)
		fmt.Fprintf(&content, "Max: %.1f h\n", item.MaxOfMaxes)
		fmt.Fprintf(&content, "Estimator count: %d\n", item.EstimatorCount)
		sections = append(sections, generator.GenerationSection{
			Title:   item.TaskName,
			Content: content.String(),
		})
	}

	return generator.GenerationInput{
		Title: fmt.Sprintf("Estimation Report — %s", projectName),
		Meta: []generator.MetaEntry{
			{Key: "Project", Value: projectName},
			{Key: "Total hours", Value: fmt.Sprintf("%.1f", agg.TotalHours)},
		},
		Sections: sections,
	}
}
