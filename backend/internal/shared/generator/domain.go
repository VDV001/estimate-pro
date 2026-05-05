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

import "context"

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
