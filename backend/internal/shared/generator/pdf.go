// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package generator

import (
	"context"
	"embed"
	"fmt"
	"strings"

	"github.com/johnfercher/maroto/v2"
	"github.com/johnfercher/maroto/v2/pkg/components/col"
	"github.com/johnfercher/maroto/v2/pkg/components/row"
	"github.com/johnfercher/maroto/v2/pkg/components/text"
	"github.com/johnfercher/maroto/v2/pkg/config"
	"github.com/johnfercher/maroto/v2/pkg/consts/align"
	"github.com/johnfercher/maroto/v2/pkg/consts/fontstyle"
	"github.com/johnfercher/maroto/v2/pkg/core"
	"github.com/johnfercher/maroto/v2/pkg/core/entity"
	"github.com/johnfercher/maroto/v2/pkg/props"
	"github.com/johnfercher/maroto/v2/pkg/repository"
)

//go:embed fonts/Roboto-Regular.ttf fonts/Roboto-Bold.ttf
var fontsFS embed.FS

const (
	fontFamily   = "Roboto"
	lineHeight   = 5.0
	bulletIndent = 8.0
	charsPerLine = 90.0
)

// PDFGenerator renders a GenerationInput as a PDF document via
// maroto/v2 (MIT-licensed). Fonts (Roboto Regular + Bold) are
// embedded into the binary, so the generator is self-contained —
// no filesystem path / runtime config dependency.
//
// One instance per process; the constructor loads the font
// repository once. Render is goroutine-safe — every call builds a
// fresh maroto.Maroto from the shared cfg.
type PDFGenerator struct {
	cfg *entity.Config
}

// NewPDFGenerator loads the embedded fonts and returns a configured
// generator. Font load failure is the only construction error and
// surfaces as a wrapped error so the composition root can fail-fast.
func NewPDFGenerator() (*PDFGenerator, error) {
	regularBytes, err := fontsFS.ReadFile("fonts/Roboto-Regular.ttf")
	if err != nil {
		return nil, fmt.Errorf("pdf: read regular font: %w", err)
	}
	boldBytes, err := fontsFS.ReadFile("fonts/Roboto-Bold.ttf")
	if err != nil {
		return nil, fmt.Errorf("pdf: read bold font: %w", err)
	}

	customFonts, err := repository.New().
		AddUTF8FontFromBytes(fontFamily, fontstyle.Normal, regularBytes).
		AddUTF8FontFromBytes(fontFamily, fontstyle.Bold, boldBytes).
		AddUTF8FontFromBytes(fontFamily, fontstyle.Italic, regularBytes).
		AddUTF8FontFromBytes(fontFamily, fontstyle.BoldItalic, boldBytes).
		Load()
	if err != nil {
		return nil, fmt.Errorf("pdf: load fonts: %w", err)
	}

	cfg := config.NewBuilder().
		WithPageNumber().
		WithLeftMargin(15).
		WithTopMargin(15).
		WithRightMargin(15).
		WithCustomFonts(customFonts).
		WithDefaultFont(&props.Font{Family: fontFamily, Size: 10}).
		Build()

	return &PDFGenerator{cfg: cfg}, nil
}

// Render builds a fresh maroto document, populates header / meta /
// sections, and returns the raw PDF bytes. Per-call allocation is
// intentional — maroto's builder is not safe for concurrent reuse.
func (g *PDFGenerator) Render(_ context.Context, input GenerationInput) ([]byte, error) {
	m := maroto.New(g.cfg)
	g.addHeader(m, input)

	for _, sec := range input.Sections {
		m.AddRows(text.NewRow(8, sec.Title, props.Text{
			Size:  12,
			Style: fontstyle.Bold,
			Top:   4,
		}))
		g.addContentLines(m, sec.Content)
		m.AddRows(row.New(3))
	}

	doc, err := m.Generate()
	if err != nil {
		return nil, fmt.Errorf("pdf: generate: %w", err)
	}
	return doc.GetBytes(), nil
}

// addHeader emits the title row with optional sub-line (project /
// client meta lookup) and a right-aligned date column. Meta
// entries beyond the recognised keys still appear via the body
// flow — the header surface is reserved for the most-prominent
// three.
func (g *PDFGenerator) addHeader(m core.Maroto, input GenerationInput) {
	title := input.Title
	if strings.TrimSpace(title) == "" {
		title = defaultTitle
	}

	var clientLine, dateLine string
	for _, m := range input.Meta {
		switch m.Key {
		case "client":
			clientLine = m.Value
		case "date":
			dateLine = m.Value
		}
	}

	_ = m.RegisterHeader(
		row.New(16).Add(
			col.New(8).Add(
				text.New(title, props.Text{
					Size:  14,
					Style: fontstyle.Bold,
				}),
				text.New(clientLine, props.Text{
					Top:   8,
					Size:  10,
					Color: &props.Color{Red: 80, Green: 80, Blue: 80},
				}),
			),
			col.New(4).Add(
				text.New(dateLine, props.Text{
					Size:  10,
					Align: align.Right,
				}),
			),
		),
	)
}

// addContentLines walks each line of section content and emits it
// as a heading / bullet / paragraph row. Markdown markers (`#`,
// `- `, `* `) are recognised but stripped — generators are not a
// markdown engine, just a thin renderer that survives the most
// common conventions in upstream LLM output.
func (g *PDFGenerator) addContentLines(m core.Maroto, content string) {
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		isHeading := strings.HasPrefix(trimmed, "#")
		isBullet := strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ")

		cleaned := stripMarkdownPrefix(trimmed)
		if cleaned == "" {
			continue
		}

		switch {
		case isHeading:
			m.AddRows(text.NewRow(estimateHeight(cleaned), cleaned, props.Text{
				Size:  11,
				Style: fontstyle.Bold,
				Top:   3,
			}))
		case isBullet:
			itemText := "•  " + cleaned
			m.AddRows(text.NewRow(estimateHeight(itemText), itemText, props.Text{
				Size: 10,
				Left: bulletIndent,
				Top:  1,
			}))
		default:
			m.AddRows(text.NewRow(estimateHeight(cleaned), cleaned, props.Text{
				Size: 10,
				Top:  1,
			}))
		}
	}
}

// stripMarkdownPrefix removes leading `#` / `- ` / `* ` markers
// and trims the residue. Inline emphasis markers (`**`, `*`, `_`)
// are left in — they are uncommon in LLM-generated estimation
// reports and stripping them would require a full markdown
// tokeniser. Acceptable trade-off for an MVP renderer.
func stripMarkdownPrefix(line string) string {
	line = strings.TrimLeft(line, "#")
	line = strings.TrimSpace(line)
	if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") {
		line = strings.TrimSpace(line[2:])
	}
	return line
}

// estimateHeight approximates the row height needed to fit a line
// of given length. The constant charsPerLine matches an A4 page at
// the configured margins + 10pt Roboto font; values shorter than
// lineHeight clamp upward so single-line rows still render.
func estimateHeight(s string) float64 {
	wrappedLines := float64(len(s))/charsPerLine + 1
	h := wrappedLines * lineHeight
	if h < lineHeight {
		return lineHeight
	}
	return h
}

// Compile-time assertion that PDFGenerator satisfies the contract.
var _ Generator = (*PDFGenerator)(nil)
