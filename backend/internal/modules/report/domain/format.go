// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

// Package domain holds the Report module's domain model: the Format
// value object, sentinel errors, and the typed enum the use case
// validates at the boundary. Persistence, HTTP, and rendering
// concerns live in sibling packages or shared infrastructure.
package domain

// Format names the report output format the consumer asks for.
// Mirrors the three formats the shared/generator Composite can
// produce from a GenerationInput (md / pdf / docx-from-scratch).
type Format string

const (
	FormatMD   Format = "md"
	FormatPDF  Format = "pdf"
	FormatDOCX Format = "docx"
)

// IsValid reports whether the receiver is one of the supported
// constants. Callers use this to guard input boundaries (HTTP
// query param, bot intent param) before handing the value to the
// use case — keeping invalid formats out of the rendering path.
func (f Format) IsValid() bool {
	return false
}
