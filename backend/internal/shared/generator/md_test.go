// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package generator_test

import (
	"context"
	"strings"
	"testing"

	"github.com/VDV001/estimate-pro/backend/internal/shared/generator"
)

// TestMDRenderer_RendersTitleAndSections pins the basic markdown
// rendering contract: a GenerationInput with Title + Sections must
// produce a markdown byte stream containing the H1 title, H2 section
// headers, and verbatim section content. This is the smallest slice
// of the generator package — no external deps, no fonts, no HTTP.
func TestMDRenderer_RendersTitleAndSections(t *testing.T) {
	input := generator.GenerationInput{
		Title: "Test Estimate Report",
		Sections: []generator.GenerationSection{
			{Title: "Tasks", Content: "Task 1 — 5h\nTask 2 — 3h"},
			{Title: "Summary", Content: "Total: 8h"},
		},
	}

	r := generator.NewMDRenderer()
	out, err := r.Render(context.Background(), input)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}

	s := string(out)
	for _, want := range []string{
		"# Test Estimate Report",
		"## Tasks",
		"Task 1 — 5h",
		"Task 2 — 3h",
		"## Summary",
		"Total: 8h",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("rendered output missing %q; got:\n%s", want, s)
		}
	}
}

// TestMDRenderer_RendersMetaTable verifies that meta key/value pairs
// emit a leading bullet list in stable insertion order before the
// section content. Operators reading the report top-down see the
// project / date / aggregate metadata before drilling into tasks.
func TestMDRenderer_RendersMetaTable(t *testing.T) {
	input := generator.GenerationInput{
		Title: "Estimate",
		Meta: []generator.MetaEntry{
			{Key: "project", Value: "EstimatePro"},
			{Key: "date", Value: "2026-05-05"},
		},
		Sections: []generator.GenerationSection{
			{Title: "Tasks", Content: "—"},
		},
	}

	out, err := generator.NewMDRenderer().Render(context.Background(), input)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}

	s := string(out)
	if !strings.Contains(s, "- **project:** EstimatePro") {
		t.Errorf("missing project meta line; got:\n%s", s)
	}
	if !strings.Contains(s, "- **date:** 2026-05-05") {
		t.Errorf("missing date meta line; got:\n%s", s)
	}
	// Meta block must precede the first section header.
	metaIdx := strings.Index(s, "- **project:**")
	sectionIdx := strings.Index(s, "## Tasks")
	if metaIdx < 0 || sectionIdx < 0 || metaIdx > sectionIdx {
		t.Errorf("meta must appear before sections; metaIdx=%d sectionIdx=%d", metaIdx, sectionIdx)
	}
}

// TestMDRenderer_EmptyTitle_FallsBackToDefault keeps the rendering
// fault-tolerant: a missing Title yields a sensible default heading
// rather than a bare "# " line. Generators are infrastructure —
// they should not reject inputs that the use case may have produced
// from incomplete domain data.
func TestMDRenderer_EmptyTitle_FallsBackToDefault(t *testing.T) {
	input := generator.GenerationInput{
		Sections: []generator.GenerationSection{{Title: "X", Content: "y"}},
	}
	out, err := generator.NewMDRenderer().Render(context.Background(), input)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	s := string(out)
	if strings.HasPrefix(s, "# \n") {
		t.Errorf("empty title produced bare '# '; got:\n%s", s)
	}
}
