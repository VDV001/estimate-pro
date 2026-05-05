// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

// Package generator hosts pure-infrastructure document generators
// (Markdown / PDF / DOCX) and a Gotenberg-backed converter for
// DOCX→PDF rendering. Each component is a thin adapter over a
// MIT-licensed third-party library (or stdlib) and exposes a single
// Render method on a stable input contract — domain modules wire
// their estimation/report data into a GenerationInput and pick the
// right Generator at the composition root.
//
// This package is deliberately free of cross-module imports and
// owns no business invariants. Validation of estimation data
// belongs in the consumer use case (PR-B7); the generator only
// needs to be tolerant to whatever it receives.
package generator

import (
	"context"
	"errors"
)

// GenerationInput is the contract every Generator accepts. The
// fields are intentionally generic — Title for the document
// heading, Meta for a leading key/value table, and Sections for the
// body. Consumer use cases map domain aggregates onto this shape.
//
// Public fields with no constructor: this is a DTO, not a domain
// VO — the generator does not enforce invariants on its input.
type GenerationInput struct {
	Title    string
	Meta     []MetaEntry
	Sections []GenerationSection
}

// MetaEntry is one row of the leading meta table. Order is
// preserved by using a slice instead of a map so consumers control
// the rendering order without leaking map-iteration randomness
// into the output.
type MetaEntry struct {
	Key   string
	Value string
}

// GenerationSection is one body block: an H2-style title and its
// verbatim content. Content is rendered as-is by each Generator
// (markdown emits the runes; PDF wraps as paragraphs; DOCX inserts
// into the template's section placeholders).
type GenerationSection struct {
	Title   string
	Content string
}

// Generator is the contract every concrete generator implements.
// One method, one return shape: bytes of the rendered document and
// an error. Concrete implementations (md / pdf / docx) decide the
// MIME type implicitly — the composition root knows which generator
// to call for which target format.
type Generator interface {
	Render(ctx context.Context, input GenerationInput) ([]byte, error)
}

// defaultTitle is the fallback heading when GenerationInput.Title
// is empty. Consumers that omit the title (e.g. quick draft from a
// partial domain object) still get a structurally valid document.
const defaultTitle = "Документ"

// TemplateFiller is the contract for placeholder-based document
// generation: take an opaque template body (typically DOCX bytes),
// substitute named placeholders, and return the filled bytes.
// Distinct from Generator because the input shape differs — a
// template carries the layout, the params carry the values.
type TemplateFiller interface {
	Fill(ctx context.Context, template []byte, params map[string]string) ([]byte, error)
}

// Converter renders one document format into another by delegating
// to an external service (Gotenberg sidecar). The contract is
// bytes-in / bytes-out; the filename hint lets the engine pick the
// correct input parser when the bytes alone are ambiguous (DOCX
// vs ODT vs RTF). Distinct from Generator (which builds from
// structured input) and TemplateFiller (which substitutes into a
// template).
type Converter interface {
	Convert(ctx context.Context, input []byte, filename string) ([]byte, error)
}

// Format names the supported output document type. Composite's
// Generate dispatches on this value; any value outside the
// constants below surfaces ErrUnsupportedFormat.
type Format string

// Supported formats. FormatDOCX is included only as a sentinel for
// the unsupported-via-Generate path — DOCX flows through
// Composite.FillTemplate which has a different input shape.
const (
	FormatMD   Format = "md"
	FormatPDF  Format = "pdf"
	FormatDOCX Format = "docx"
)

// Sentinel errors. ADR-014 — every sentinel ships with a consumer
// branch that returns it. Tests in this package supply the consumer
// for the input-validation slices; the production consumer is the
// PR-B7 report use case.
var (
	ErrEmptyTemplate          = errors.New("generator: template is empty")
	ErrInvalidTemplate        = errors.New("generator: template is not a valid DOCX (zip parse failed)")
	ErrGotenbergUnavailable   = errors.New("generator: gotenberg service unavailable")
	ErrInvalidConversionInput = errors.New("generator: gotenberg rejected conversion input")
	ErrUnsupportedFormat      = errors.New("generator: unsupported output format")
)
