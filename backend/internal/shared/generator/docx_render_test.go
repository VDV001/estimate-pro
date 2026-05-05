// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package generator_test

import (
	"archive/zip"
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"github.com/VDV001/estimate-pro/backend/internal/shared/generator"
)

// readDocumentXML opens the rendered DOCX bytes as a zip and returns
// the concatenated text content of word/document.xml. Tests on this
// helper assert against the visible text rather than the XML
// envelope — the envelope is verified once in TestDOCXRenderer_ZipShape.
func readDocumentXML(t *testing.T, raw []byte) string {
	t.Helper()
	r, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		t.Fatalf("zip.NewReader: %v", err)
	}
	for _, f := range r.File {
		if f.Name != "word/document.xml" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			t.Fatalf("open document.xml: %v", err)
		}
		defer rc.Close()
		body, err := io.ReadAll(rc)
		if err != nil {
			t.Fatalf("read document.xml: %v", err)
		}
		return string(body)
	}
	t.Fatalf("word/document.xml missing from zip")
	return ""
}

func TestDOCXRenderer_RendersTitleAndSections(t *testing.T) {
	input := generator.GenerationInput{
		Title: "Test Estimate Report",
		Sections: []generator.GenerationSection{
			{Title: "Tasks", Content: "Task 1 — 5h\nTask 2 — 3h"},
			{Title: "Summary", Content: "Total: 8h"},
		},
	}

	r := generator.NewDOCXRenderer()
	out, err := r.Render(context.Background(), input)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}

	doc := readDocumentXML(t, out)
	for _, want := range []string{
		"Test Estimate Report",
		"Tasks",
		"Task 1 — 5h",
		"Task 2 — 3h",
		"Summary",
		"Total: 8h",
	} {
		if !strings.Contains(doc, want) {
			t.Errorf("document.xml missing %q; got:\n%s", want, doc)
		}
	}
}

func TestDOCXRenderer_RendersMeta(t *testing.T) {
	input := generator.GenerationInput{
		Title: "Estimate",
		Meta: []generator.MetaEntry{
			{Key: "project", Value: "EstimatePro"},
			{Key: "date", Value: "2026-05-05"},
		},
		Sections: []generator.GenerationSection{
			{Title: "Tasks", Content: "Task 1"},
		},
	}

	r := generator.NewDOCXRenderer()
	out, err := r.Render(context.Background(), input)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}

	doc := readDocumentXML(t, out)
	for _, want := range []string{"project", "EstimatePro", "date", "2026-05-05"} {
		if !strings.Contains(doc, want) {
			t.Errorf("document.xml missing meta %q; got:\n%s", want, doc)
		}
	}
}

func TestDOCXRenderer_FallbackTitleWhenEmpty(t *testing.T) {
	input := generator.GenerationInput{
		Title:    "",
		Sections: []generator.GenerationSection{{Title: "S", Content: "c"}},
	}
	out, err := generator.NewDOCXRenderer().Render(context.Background(), input)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	doc := readDocumentXML(t, out)
	// defaultTitle is package-private; assert non-empty by checking
	// the document doesn't open with the section header alone.
	if !strings.Contains(doc, "Документ") {
		t.Errorf("expected fallback title 'Документ', got:\n%s", doc)
	}
}

func TestDOCXRenderer_EscapesXMLSpecialChars(t *testing.T) {
	input := generator.GenerationInput{
		Title: "Title with <tag> & \"quote\"",
		Sections: []generator.GenerationSection{
			{Title: "Bad <h2>", Content: "5 < 10 & \"safe\""},
		},
	}
	out, err := generator.NewDOCXRenderer().Render(context.Background(), input)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}

	// Result must still be a valid zip with parseable document.xml.
	doc := readDocumentXML(t, out)

	// Raw '<tag>' as bytes would have been parsed as XML if not escaped.
	if strings.Contains(doc, "<tag>") {
		t.Errorf("raw <tag> leaked into document.xml — escaping failed")
	}
	if !strings.Contains(doc, "&lt;tag&gt;") {
		t.Errorf("expected escaped &lt;tag&gt;; got:\n%s", doc)
	}
	if !strings.Contains(doc, "&amp;") {
		t.Errorf("ampersand not escaped to &amp;; got:\n%s", doc)
	}
}

func TestDOCXRenderer_ZipShape(t *testing.T) {
	out, err := generator.NewDOCXRenderer().Render(context.Background(), generator.GenerationInput{
		Title:    "T",
		Sections: []generator.GenerationSection{{Title: "S", Content: "c"}},
	})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}

	r, err := zip.NewReader(bytes.NewReader(out), int64(len(out)))
	if err != nil {
		t.Fatalf("zip.NewReader: %v", err)
	}

	required := map[string]bool{
		"[Content_Types].xml":              false,
		"_rels/.rels":                      false,
		"word/document.xml":                false,
		"word/_rels/document.xml.rels":     false,
	}
	for _, f := range r.File {
		if _, ok := required[f.Name]; ok {
			required[f.Name] = true
		}
	}
	for name, present := range required {
		if !present {
			t.Errorf("zip missing required entry %q", name)
		}
	}
}
