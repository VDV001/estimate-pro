// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package generator

import (
	"context"
	"fmt"
)

// Composite bundles the three generation paths (structured
// rendering for MD / PDF, template filling for DOCX, byte
// conversion for DOCX→PDF) into a single dependency for
// consumers. The composition root in cmd/server/main.go builds
// one Composite and hands it to whichever module needs document
// generation (PR-B7 report use case).
//
// The converter is optional: local-dev environments without the
// Gotenberg sidecar can pass nil; FillTemplate / Generate keep
// working, ConvertToPDF returns ErrGotenbergUnavailable so the
// caller decides whether to retry or surface a configuration
// error.
type Composite struct {
	md        Generator
	pdf       Generator
	docx      TemplateFiller
	converter Converter
}

// NewComposite wires the four collaborators. md and pdf must be
// non-nil — they have no fallback and a nil pointer here is a
// programmer error that surfaces on the first Generate call.
// The converter is allowed to be nil; see Composite docs.
func NewComposite(md, pdf Generator, docx TemplateFiller, converter Converter) *Composite {
	return &Composite{md: md, pdf: pdf, docx: docx, converter: converter}
}

// Generate dispatches on the requested format. FormatMD / FormatPDF
// land on the corresponding Generator; FormatDOCX (and any unknown
// format) returns ErrUnsupportedFormat — DOCX flows through
// FillTemplate, not Generate.
func (c *Composite) Generate(ctx context.Context, format Format, input GenerationInput) ([]byte, error) {
	switch format {
	case FormatMD:
		return c.md.Render(ctx, input)
	case FormatPDF:
		return c.pdf.Render(ctx, input)
	default:
		return nil, fmt.Errorf("composite: %w: %q", ErrUnsupportedFormat, format)
	}
}

// FillTemplate forwards verbatim to the DOCX template filler.
// Kept on the Composite so downstream consumers depend on a
// single struct, not three interfaces.
func (c *Composite) FillTemplate(ctx context.Context, template []byte, params map[string]string) ([]byte, error) {
	return c.docx.Fill(ctx, template, params)
}

// ConvertToPDF forwards to the Gotenberg converter when wired,
// otherwise surfaces ErrGotenbergUnavailable. The caller should
// branch on errors.Is to decide retry vs configuration-error UX.
func (c *Composite) ConvertToPDF(ctx context.Context, docx []byte, filename string) ([]byte, error) {
	if c.converter == nil {
		return nil, fmt.Errorf("composite: converter not configured: %w", ErrGotenbergUnavailable)
	}
	return c.converter.Convert(ctx, docx, filename)
}
