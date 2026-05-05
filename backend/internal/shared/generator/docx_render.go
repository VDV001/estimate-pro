// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package generator

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"html"
	"strings"
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
func (r *DOCXRenderer) Render(ctx context.Context, input GenerationInput) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	title := strings.TrimSpace(input.Title)
	if title == "" {
		title = defaultTitle
	}

	var body bytes.Buffer
	body.WriteString(documentHeader)
	writeParagraph(&body, title, "Heading1")

	for _, m := range input.Meta {
		writeParagraph(&body, fmt.Sprintf("%s: %s", m.Key, m.Value), "")
	}

	for _, sec := range input.Sections {
		if strings.TrimSpace(sec.Title) != "" {
			writeParagraph(&body, sec.Title, "Heading2")
		}
		for _, line := range strings.Split(sec.Content, "\n") {
			writeParagraph(&body, line, "")
		}
	}
	body.WriteString(documentFooter)

	var buf bytes.Buffer
	z := zip.NewWriter(&buf)
	parts := []struct {
		name, content string
	}{
		{"[Content_Types].xml", contentTypesXML},
		{"_rels/.rels", packageRelsXML},
		{"word/_rels/document.xml.rels", documentRelsXML},
		{"word/document.xml", body.String()},
	}
	for _, p := range parts {
		w, err := z.Create(p.name)
		if err != nil {
			return nil, fmt.Errorf("docx: create zip entry %s: %w", p.name, err)
		}
		if _, err := w.Write([]byte(p.content)); err != nil {
			return nil, fmt.Errorf("docx: write zip entry %s: %w", p.name, err)
		}
	}
	if err := z.Close(); err != nil {
		return nil, fmt.Errorf("docx: close zip: %w", err)
	}
	return buf.Bytes(), nil
}

// writeParagraph appends a single <w:p> with the given text and
// optional pStyle. text is XML-escaped via html.EscapeString — DOCX
// uses the same five XML predefined entities, and html.EscapeString
// covers all of them without the html package's other quirks.
func writeParagraph(b *bytes.Buffer, text, style string) {
	b.WriteString("<w:p>")
	if style != "" {
		fmt.Fprintf(b, `<w:pPr><w:pStyle w:val="%s"/></w:pPr>`, style)
	}
	b.WriteString(`<w:r><w:t xml:space="preserve">`)
	b.WriteString(html.EscapeString(text))
	b.WriteString(`</w:t></w:r></w:p>`)
}

const documentHeader = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
<w:body>`

const documentFooter = `</w:body>
</w:document>`

const contentTypesXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
<Default Extension="xml" ContentType="application/xml"/>
<Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
</Types>`

const packageRelsXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
</Relationships>`

const documentRelsXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
</Relationships>`
