// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package generator

import (
	"context"
	"fmt"
	"strings"
)

// MDRenderer produces a CommonMark-flavoured markdown byte stream
// from a GenerationInput. No external deps — pure stdlib string
// building. Adapter pattern from deal-sense md_renderer.go,
// adapted to the EstimatePro GenerationInput shape (slice-of-pairs
// Meta instead of map for stable ordering).
type MDRenderer struct{}

// NewMDRenderer returns a stateless renderer. Stateless means safe
// for concurrent reuse from any goroutine — the composition root
// can hand the same instance to every consumer.
func NewMDRenderer() *MDRenderer {
	return &MDRenderer{}
}

// Render emits the document in this order: H1 title (or fallback),
// optional meta bullet block followed by a horizontal rule, then
// each section as H2 + verbatim content. Trailing newline so the
// stream is POSIX-friendly.
func (r *MDRenderer) Render(_ context.Context, input GenerationInput) ([]byte, error) {
	var b strings.Builder

	title := input.Title
	if strings.TrimSpace(title) == "" {
		title = defaultTitle
	}
	fmt.Fprintf(&b, "# %s\n\n", title)

	if len(input.Meta) > 0 {
		for _, m := range input.Meta {
			fmt.Fprintf(&b, "- **%s:** %s\n", m.Key, m.Value)
		}
		b.WriteString("\n---\n\n")
	}

	for _, sec := range input.Sections {
		fmt.Fprintf(&b, "## %s\n\n%s\n\n", sec.Title, sec.Content)
	}

	return []byte(b.String()), nil
}

// Compile-time assertion that MDRenderer satisfies the Generator
// contract. If the interface evolves (e.g. adds a Format() method),
// every implementation breaks at build time, not at runtime.
var _ Generator = (*MDRenderer)(nil)
