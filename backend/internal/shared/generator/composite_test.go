// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package generator_test

import (
	"context"
	"errors"
	"testing"

	"github.com/VDV001/estimate-pro/backend/internal/shared/generator"
)

// recordingGenerator captures the last Render call for assertion.
type recordingGenerator struct {
	calls       int
	gotInput    generator.GenerationInput
	respBytes   []byte
	respErr     error
}

func (r *recordingGenerator) Render(_ context.Context, in generator.GenerationInput) ([]byte, error) {
	r.calls++
	r.gotInput = in
	return r.respBytes, r.respErr
}

// recordingFiller captures the last Fill call for assertion.
type recordingFiller struct {
	calls       int
	gotTemplate []byte
	gotParams   map[string]string
	respBytes   []byte
	respErr     error
}

func (r *recordingFiller) Fill(_ context.Context, t []byte, p map[string]string) ([]byte, error) {
	r.calls++
	r.gotTemplate = t
	r.gotParams = p
	return r.respBytes, r.respErr
}

// recordingConverter captures the last Convert call.
type recordingConverter struct {
	calls       int
	gotInput    []byte
	gotFilename string
	respBytes   []byte
	respErr     error
}

func (r *recordingConverter) Convert(_ context.Context, in []byte, filename string) ([]byte, error) {
	r.calls++
	r.gotInput = in
	r.gotFilename = filename
	return r.respBytes, r.respErr
}

// TestComposite_GenerateDispatchesByFormat pins the routing
// contract: FormatMD lands on the md generator, FormatPDF lands
// on the pdf generator. Wrong-target generators must not be
// touched (count assertions).
func TestComposite_GenerateDispatchesByFormat(t *testing.T) {
	md := &recordingGenerator{respBytes: []byte("# md")}
	pdf := &recordingGenerator{respBytes: []byte("%PDF-1.4")}
	docx := &recordingFiller{}
	conv := &recordingConverter{}

	c := generator.NewComposite(md, pdf, docx, conv)
	input := generator.GenerationInput{Title: "T"}

	out, err := c.Generate(context.Background(), generator.FormatMD, input)
	if err != nil {
		t.Fatalf("Generate(md): %v", err)
	}
	if string(out) != "# md" {
		t.Errorf("md output=%q, want '# md'", out)
	}
	if md.calls != 1 || pdf.calls != 0 {
		t.Errorf("md.calls=%d pdf.calls=%d, want 1/0", md.calls, pdf.calls)
	}

	out, err = c.Generate(context.Background(), generator.FormatPDF, input)
	if err != nil {
		t.Fatalf("Generate(pdf): %v", err)
	}
	if string(out) != "%PDF-1.4" {
		t.Errorf("pdf output=%q", out)
	}
	if md.calls != 1 || pdf.calls != 1 {
		t.Errorf("md.calls=%d pdf.calls=%d, want 1/1", md.calls, pdf.calls)
	}
}

// TestComposite_GenerateUnsupportedFormat covers the failure
// branch: DOCX (templated) and unknown formats surface
// ErrUnsupportedFormat — DOCX does not match the structured-input
// contract, the caller must use FillTemplate instead.
func TestComposite_GenerateUnsupportedFormat(t *testing.T) {
	c := generator.NewComposite(&recordingGenerator{}, &recordingGenerator{}, &recordingFiller{}, &recordingConverter{})

	for _, f := range []generator.Format{generator.FormatDOCX, generator.Format("xls"), generator.Format("")} {
		_, err := c.Generate(context.Background(), f, generator.GenerationInput{})
		if !errors.Is(err, generator.ErrUnsupportedFormat) {
			t.Errorf("Generate(%q) err=%v, want errors.Is ErrUnsupportedFormat", f, err)
		}
	}
}

// TestComposite_FillTemplateDelegates covers the template path:
// FillTemplate forwards verbatim to the DOCX filler.
func TestComposite_FillTemplateDelegates(t *testing.T) {
	docx := &recordingFiller{respBytes: []byte("FILLED-DOCX")}
	c := generator.NewComposite(&recordingGenerator{}, &recordingGenerator{}, docx, &recordingConverter{})

	out, err := c.FillTemplate(context.Background(),
		[]byte("TEMPLATE"),
		map[string]string{"k": "v"})
	if err != nil {
		t.Fatalf("FillTemplate: %v", err)
	}
	if string(out) != "FILLED-DOCX" {
		t.Errorf("output=%q", out)
	}
	if docx.calls != 1 {
		t.Errorf("docx.calls=%d, want 1", docx.calls)
	}
	if string(docx.gotTemplate) != "TEMPLATE" || docx.gotParams["k"] != "v" {
		t.Errorf("docx received template=%q params=%v", docx.gotTemplate, docx.gotParams)
	}
}

// TestComposite_ConvertToPDFDelegates covers the converter path.
func TestComposite_ConvertToPDFDelegates(t *testing.T) {
	conv := &recordingConverter{respBytes: []byte("%PDF-CONVERTED")}
	c := generator.NewComposite(&recordingGenerator{}, &recordingGenerator{}, &recordingFiller{}, conv)

	out, err := c.ConvertToPDF(context.Background(), []byte("DOCX"), "x.docx")
	if err != nil {
		t.Fatalf("ConvertToPDF: %v", err)
	}
	if string(out) != "%PDF-CONVERTED" {
		t.Errorf("output=%q", out)
	}
	if conv.calls != 1 || string(conv.gotInput) != "DOCX" || conv.gotFilename != "x.docx" {
		t.Errorf("converter received input=%q filename=%q calls=%d",
			conv.gotInput, conv.gotFilename, conv.calls)
	}
}

// TestComposite_NilConverter_FillTemplateOK verifies graceful
// degradation: even if the Gotenberg converter is nil (e.g. local
// dev without sidecar), the rest of the composite stays usable —
// MD / PDF / DOCX-fill all work, and only ConvertToPDF rejects
// the call with a sentinel.
func TestComposite_NilConverter_FillTemplateOK(t *testing.T) {
	c := generator.NewComposite(&recordingGenerator{}, &recordingGenerator{}, &recordingFiller{respBytes: []byte("ok")}, nil)

	if _, err := c.FillTemplate(context.Background(), []byte("T"), nil); err != nil {
		t.Errorf("FillTemplate with nil converter: %v", err)
	}
	_, err := c.ConvertToPDF(context.Background(), []byte("DOCX"), "x.docx")
	if !errors.Is(err, generator.ErrGotenbergUnavailable) {
		t.Fatalf("ConvertToPDF nil converter err=%v, want errors.Is ErrGotenbergUnavailable", err)
	}
}
