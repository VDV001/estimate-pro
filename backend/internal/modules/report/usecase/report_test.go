// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package usecase_test

import (
	"context"
	"strings"
	"testing"

	estimationdomain "github.com/VDV001/estimate-pro/backend/internal/modules/estimation/domain"
	projectdomain "github.com/VDV001/estimate-pro/backend/internal/modules/project/domain"
	reportdomain "github.com/VDV001/estimate-pro/backend/internal/modules/report/domain"
	"github.com/VDV001/estimate-pro/backend/internal/modules/report/usecase"
	"github.com/VDV001/estimate-pro/backend/internal/shared/generator"
)

// stubProject implements ProjectMetadataReader.
type stubProject struct {
	resp *projectdomain.Project
	err  error
}

func (s *stubProject) GetByID(_ context.Context, _ string) (*projectdomain.Project, error) {
	return s.resp, s.err
}

// stubAggregator implements EstimationAggregator.
type stubAggregator struct {
	resp *estimationdomain.AggregatedResult
	err  error
}

func (s *stubAggregator) GetAggregated(_ context.Context, _ string) (*estimationdomain.AggregatedResult, error) {
	return s.resp, s.err
}

// recordingRenderer captures Render args for assertion.
type recordingRenderer struct {
	calls       int
	gotFormat   reportdomain.Format
	gotInput    generator.GenerationInput
	respBytes   []byte
	respErr     error
}

func (r *recordingRenderer) Render(_ context.Context, format reportdomain.Format, input generator.GenerationInput) ([]byte, error) {
	r.calls++
	r.gotFormat = format
	r.gotInput = input
	return r.respBytes, r.respErr
}

func TestReporter_RendersInRequestedFormat(t *testing.T) {
	project := &projectdomain.Project{
		ID:          "p1",
		Name:        "Alpha Project",
		Description: "Build a thing",
	}
	aggregated := &estimationdomain.AggregatedResult{
		Items: []*estimationdomain.AggregatedItem{
			{TaskName: "Implement login", AvgPERTHours: 4.5, MinOfMins: 3, MaxOfMaxes: 6, EstimatorCount: 2},
			{TaskName: "Wire OAuth", AvgPERTHours: 8.0, MinOfMins: 6, MaxOfMaxes: 12, EstimatorCount: 2},
		},
		TotalHours: 12.5,
	}

	rec := &recordingRenderer{respBytes: []byte("RENDERED")}
	uc := usecase.NewReporter(&stubProject{resp: project}, &stubAggregator{resp: aggregated}, rec)

	out, err := uc.RenderEstimationReport(context.Background(), "p1", reportdomain.FormatPDF)
	if err != nil {
		t.Fatalf("RenderEstimationReport: %v", err)
	}
	if string(out) != "RENDERED" {
		t.Errorf("output=%q, want 'RENDERED'", out)
	}
	if rec.calls != 1 {
		t.Fatalf("renderer.calls=%d, want 1", rec.calls)
	}
	if rec.gotFormat != reportdomain.FormatPDF {
		t.Errorf("renderer received format=%q, want pdf", rec.gotFormat)
	}
}

func TestReporter_BuildsGenerationInputFromProjectAndAggregation(t *testing.T) {
	project := &projectdomain.Project{
		ID:          "p1",
		Name:        "Alpha Project",
		Description: "Build a thing",
	}
	aggregated := &estimationdomain.AggregatedResult{
		Items: []*estimationdomain.AggregatedItem{
			{TaskName: "Implement login", AvgPERTHours: 4.5, MinOfMins: 3, MaxOfMaxes: 6, EstimatorCount: 2},
			{TaskName: "Wire OAuth", AvgPERTHours: 8.0, MinOfMins: 6, MaxOfMaxes: 12, EstimatorCount: 2},
		},
		TotalHours: 12.5,
	}

	rec := &recordingRenderer{respBytes: []byte("OK")}
	uc := usecase.NewReporter(&stubProject{resp: project}, &stubAggregator{resp: aggregated}, rec)

	if _, err := uc.RenderEstimationReport(context.Background(), "p1", reportdomain.FormatMD); err != nil {
		t.Fatalf("RenderEstimationReport: %v", err)
	}

	input := rec.gotInput
	if !strings.Contains(input.Title, project.Name) {
		t.Errorf("title=%q, want contains project name %q", input.Title, project.Name)
	}

	// Meta should carry at least project name and total hours.
	metaText := ""
	for _, m := range input.Meta {
		metaText += m.Key + "=" + m.Value + ";"
	}
	for _, want := range []string{project.Name, "12.5"} {
		if !strings.Contains(metaText, want) {
			t.Errorf("meta missing %q; got %s", want, metaText)
		}
	}

	// Sections should contain one per aggregated task plus content
	// referencing PERT / min / max / estimator count.
	if len(input.Sections) == 0 {
		t.Fatalf("sections empty, expected at least one per task")
	}
	allContent := ""
	for _, s := range input.Sections {
		allContent += s.Title + "\n" + s.Content + "\n"
	}
	for _, want := range []string{"Implement login", "Wire OAuth", "4.5", "8.0", "estimator"} {
		if !strings.Contains(strings.ToLower(allContent), strings.ToLower(want)) {
			t.Errorf("sections missing %q; got:\n%s", want, allContent)
		}
	}
}
