// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package generator_test

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/VDV001/estimate-pro/backend/internal/shared/generator"
)

// buildDOCXFixture writes a minimal DOCX (zip with word/document.xml)
// containing the provided body XML. The body is wrapped in the
// minimum OOXML envelope that LibreOffice / Word will open without
// complaint. Tests use this instead of committing a binary fixture
// to the repo — keeps the diff small and the fixture verbatim
// auditable.
func buildDOCXFixture(t *testing.T, bodyXML string) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)

	doc := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>` + bodyXML + `</w:body>
</w:document>`
	f, err := w.Create("word/document.xml")
	if err != nil {
		t.Fatalf("zip create: %v", err)
	}
	if _, err := f.Write([]byte(doc)); err != nil {
		t.Fatalf("zip write: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("zip close: %v", err)
	}
	return buf.Bytes()
}

// readDOCXBody extracts word/document.xml from a generated DOCX
// for assertion. The returned string is verbatim XML.
func readDOCXBody(t *testing.T, docx []byte) string {
	t.Helper()
	r, err := zip.NewReader(bytes.NewReader(docx), int64(len(docx)))
	if err != nil {
		t.Fatalf("zip open: %v", err)
	}
	for _, f := range r.File {
		if f.Name == "word/document.xml" {
			rc, err := f.Open()
			if err != nil {
				t.Fatalf("zip entry open: %v", err)
			}
			defer rc.Close()
			data, err := io.ReadAll(rc)
			if err != nil {
				t.Fatalf("zip entry read: %v", err)
			}
			return string(data)
		}
	}
	t.Fatal("word/document.xml missing in fixture")
	return ""
}

// TestDOCXTemplateFiller_ReplacesSimplePlaceholder pins the basic
// substitution: `{{key}}` in a single <w:r><w:t> run gets replaced
// with the corresponding params value.
func TestDOCXTemplateFiller_ReplacesSimplePlaceholder(t *testing.T) {
	template := buildDOCXFixture(t, `<w:p><w:r><w:t>Hello {{name}}!</w:t></w:r></w:p>`)
	out, err := generator.NewDOCXTemplateFiller().Fill(context.Background(), template, map[string]string{
		"name": "World",
	})
	if err != nil {
		t.Fatalf("Fill: %v", err)
	}
	body := readDOCXBody(t, out)
	if !strings.Contains(body, "Hello World!") {
		t.Errorf("missing substituted text; body=%s", body)
	}
	if strings.Contains(body, "{{name}}") {
		t.Errorf("placeholder not replaced; body=%s", body)
	}
}

// TestDOCXTemplateFiller_EscapesXMLSpecials confirms that values
// with `<`, `>`, `&`, `"`, `'` are properly XML-escaped — without
// this, a malicious or accidental special char would corrupt the
// document and crash Word/LibreOffice on open.
func TestDOCXTemplateFiller_EscapesXMLSpecials(t *testing.T) {
	template := buildDOCXFixture(t, `<w:p><w:r><w:t>{{snippet}}</w:t></w:r></w:p>`)
	out, err := generator.NewDOCXTemplateFiller().Fill(context.Background(), template, map[string]string{
		"snippet": `<script>alert("x")</script>`,
	})
	if err != nil {
		t.Fatalf("Fill: %v", err)
	}
	body := readDOCXBody(t, out)
	if strings.Contains(body, "<script>") {
		t.Errorf("raw < / > leaked into XML; body=%s", body)
	}
	if !strings.Contains(body, "&lt;script&gt;") {
		t.Errorf("expected XML-escaped <script>; body=%s", body)
	}
	if !strings.Contains(body, "&quot;x&quot;") {
		t.Errorf("expected XML-escaped quotes; body=%s", body)
	}
}

// TestDOCXTemplateFiller_MergesSplitRuns covers the OOXML quirk
// where Word may split a `{{name}}` token across multiple <w:r>
// runs (e.g. <w:r><w:t>{{</w:t></w:r><w:r><w:t>name}}</w:t></w:r>).
// The filler must normalise these into a single run before the
// string-replace step, otherwise the substitution silently misses.
func TestDOCXTemplateFiller_MergesSplitRuns(t *testing.T) {
	template := buildDOCXFixture(t,
		`<w:p><w:r><w:t>{{</w:t></w:r><w:r><w:t>name</w:t></w:r><w:r><w:t>}}</w:t></w:r></w:p>`)
	out, err := generator.NewDOCXTemplateFiller().Fill(context.Background(), template, map[string]string{
		"name": "Estimate",
	})
	if err != nil {
		t.Fatalf("Fill: %v", err)
	}
	body := readDOCXBody(t, out)
	if !strings.Contains(body, "Estimate") {
		t.Errorf("split-run placeholder not merged + replaced; body=%s", body)
	}
}

// TestDOCXTemplateFiller_EmptyTemplate_ReturnsSentinel keeps the
// API safe under degenerate input: a zero-length template surfaces
// generator.ErrEmptyTemplate via errors.Is so callers can branch
// on the misuse case rather than parse error strings.
func TestDOCXTemplateFiller_EmptyTemplate_ReturnsSentinel(t *testing.T) {
	_, err := generator.NewDOCXTemplateFiller().Fill(context.Background(), nil, map[string]string{})
	if !errors.Is(err, generator.ErrEmptyTemplate) {
		t.Fatalf("err=%v, want errors.Is generator.ErrEmptyTemplate", err)
	}
}

// TestDOCXTemplateFiller_InvalidZip_ReturnsSentinel covers the
// other input-validation slice: random bytes that do not parse as
// a zip archive surface ErrInvalidTemplate. Callers (template
// upload handlers) can return 400 instead of leaking zip parse
// internals.
func TestDOCXTemplateFiller_InvalidZip_ReturnsSentinel(t *testing.T) {
	_, err := generator.NewDOCXTemplateFiller().Fill(context.Background(),
		[]byte("not a zip"), map[string]string{})
	if !errors.Is(err, generator.ErrInvalidTemplate) {
		t.Fatalf("err=%v, want errors.Is generator.ErrInvalidTemplate", err)
	}
}
