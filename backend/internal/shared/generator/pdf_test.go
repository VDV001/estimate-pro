// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package generator_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/VDV001/estimate-pro/backend/internal/shared/generator"
)

// TestPDFGenerator_ProducesValidPDF pins the basic PDF generation
// contract: any GenerationInput must produce a non-empty byte
// stream whose first 4 bytes are the PDF magic header `%PDF`.
// Reading the rendered PDF for content assertions is overkill in
// unit tests — we rely on maroto's own coverage for layout, the
// integration / golden-file test lands in PR-B7 alongside the
// estimation report use case.
func TestPDFGenerator_ProducesValidPDF(t *testing.T) {
	input := generator.GenerationInput{
		Title: "Test Estimate Report",
		Meta: []generator.MetaEntry{
			{Key: "project", Value: "EstimatePro"},
			{Key: "date", Value: "2026-05-05"},
		},
		Sections: []generator.GenerationSection{
			{Title: "Tasks", Content: "- Task 1 — 5h\n- Task 2 — 3h"},
			{Title: "Summary", Content: "Total: 8h"},
		},
	}

	g, err := generator.NewPDFGenerator()
	if err != nil {
		t.Fatalf("NewPDFGenerator: %v", err)
	}
	out, err := g.Render(context.Background(), input)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}

	if len(out) < 1024 {
		t.Errorf("PDF too small (%d bytes); maroto should produce at least a few KB", len(out))
	}
	if !bytes.HasPrefix(out, []byte("%PDF-")) {
		t.Errorf("output does not start with PDF magic; first 16 bytes: %q", out[:min(16, len(out))])
	}
}

// TestPDFGenerator_HandlesEmptyInput keeps the renderer fault-tolerant:
// an empty GenerationInput must not panic — title falls back to
// defaultTitle, sections list is empty, and we still get a valid
// (single-page) PDF. Generators are infrastructure; rejecting
// inputs is the use case's job, not theirs.
func TestPDFGenerator_HandlesEmptyInput(t *testing.T) {
	g, err := generator.NewPDFGenerator()
	if err != nil {
		t.Fatalf("NewPDFGenerator: %v", err)
	}
	out, err := g.Render(context.Background(), generator.GenerationInput{})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !bytes.HasPrefix(out, []byte("%PDF-")) {
		t.Fatalf("empty input produced invalid PDF; first bytes: %q", out[:min(16, len(out))])
	}
}

// TestPDFGenerator_StatelessConcurrency confirms that the generator
// can be reused across goroutines safely — the composition root
// hands one instance to many consumers (worker, future report use
// case) and we do not want to allocate per-call. The font repository
// is loaded once at construction, so reuse must work.
func TestPDFGenerator_StatelessConcurrency(t *testing.T) {
	g, err := generator.NewPDFGenerator()
	if err != nil {
		t.Fatalf("NewPDFGenerator: %v", err)
	}

	input := generator.GenerationInput{
		Title:    "Stress",
		Sections: []generator.GenerationSection{{Title: "x", Content: "y"}},
	}

	const N = 4
	errs := make(chan error, N)
	for i := 0; i < N; i++ {
		go func() {
			_, err := g.Render(context.Background(), input)
			errs <- err
		}()
	}
	for i := 0; i < N; i++ {
		if err := <-errs; err != nil {
			t.Errorf("concurrent Render[%d]: %v", i, err)
		}
	}
}
