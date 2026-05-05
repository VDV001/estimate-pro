// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package generator

import (
	"context"
	"errors"
)

// DOCXRenderer produces a DOCX byte stream from a GenerationInput
// by assembling a minimal Office Open XML package — no external
// template, no third-party library. Stateless and safe for
// concurrent reuse, mirroring MDRenderer / PDFGenerator.
//
// Distinct from DOCXTemplateFiller: the filler substitutes named
// placeholders inside an opaque template (the layout lives in the
// template), whereas this renderer emits the layout itself from
// {Title, Meta, Sections}. Both implement Generator semantics in
// principle, but only this one is wired through Composite.Generate.
type DOCXRenderer struct{}

// NewDOCXRenderer returns a stateless renderer.
func NewDOCXRenderer() *DOCXRenderer {
	return &DOCXRenderer{}
}

// Render emits a single-document DOCX zip in this order:
//   - H1-styled title (defaultTitle if empty)
//   - one paragraph per Meta entry as "Key: Value"
//   - for each Section: H2-styled title, then content split on
//     newlines into one paragraph per line
//
// XML special characters in any text are escaped so user-provided
// strings cannot break the document XML.
func (r *DOCXRenderer) Render(_ context.Context, _ GenerationInput) ([]byte, error) {
	return nil, errors.New("DOCXRenderer.Render: not implemented")
}
