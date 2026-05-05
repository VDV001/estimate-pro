// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package generator

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"regexp"
	"strings"
)

// DOCXTemplateFiller substitutes {{key}} placeholders in DOCX
// templates with values from a params map. Pure stdlib zip+xml —
// no third-party DOCX library, no AGPL surface. Adapter pattern
// from deal-sense docx_template.go (also stdlib).
//
// Stateless: safe for concurrent reuse. The composition root hands
// one instance to every consumer.
type DOCXTemplateFiller struct{}

// NewDOCXTemplateFiller returns a stateless filler.
func NewDOCXTemplateFiller() *DOCXTemplateFiller {
	return &DOCXTemplateFiller{}
}

// Fill walks every entry of the input zip; for entries that look
// like body / header / footer XML it merges run-split placeholders
// (Word's authoring quirk where {{name}} can land across multiple
// <w:r> runs), then string-replaces every `{{key}}` occurrence
// with the XML-escaped params value. Other zip entries (relations,
// styles, embedded media) pass through verbatim.
//
// Errors: ErrEmptyTemplate for nil/zero-length input, ErrInvalidTemplate
// when the bytes do not parse as a zip archive. Both sentinels match
// via errors.Is for caller-side branching.
func (f *DOCXTemplateFiller) Fill(_ context.Context, template []byte, params map[string]string) ([]byte, error) {
	if len(template) == 0 {
		return nil, fmt.Errorf("docx: %w", ErrEmptyTemplate)
	}

	r, err := zip.NewReader(bytes.NewReader(template), int64(len(template)))
	if err != nil {
		return nil, fmt.Errorf("docx: %w: %v", ErrInvalidTemplate, err)
	}

	var buf bytes.Buffer
	w := zip.NewWriter(&buf)

	for _, entry := range r.File {
		content, err := readZipEntry(entry)
		if err != nil {
			return nil, fmt.Errorf("docx: read entry %q: %w", entry.Name, err)
		}

		if isDocxXML(entry.Name) {
			xml := mergePlaceholderRuns(string(content))
			for k, v := range params {
				xml = strings.ReplaceAll(xml, "{{"+k+"}}", escapeXML(v))
			}
			content = []byte(xml)
		}

		header := entry.FileHeader
		header.UncompressedSize64 = uint64(len(content))
		fw, err := w.CreateHeader(&header)
		if err != nil {
			return nil, fmt.Errorf("docx: write header %q: %w", entry.Name, err)
		}
		if _, err := fw.Write(content); err != nil {
			return nil, fmt.Errorf("docx: write entry %q: %w", entry.Name, err)
		}
	}

	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("docx: zip close: %w", err)
	}
	return buf.Bytes(), nil
}

// readZipEntry reads the full contents of a zip entry. Surfaced
// errors are surfaced upward — the caller wraps with the entry
// name for traceability.
func readZipEntry(f *zip.File) ([]byte, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(rc)
}

// isDocxXML reports whether a zip entry holds a substitution-eligible
// document part. The OOXML standard pins these names — we match
// the body and the optional headers/footers; styles, relations,
// embedded media are off-limits for placeholder substitution.
func isDocxXML(name string) bool {
	switch name {
	case "word/document.xml",
		"word/header1.xml", "word/header2.xml", "word/header3.xml",
		"word/footer1.xml", "word/footer2.xml", "word/footer3.xml":
		return true
	}
	return false
}

// escapeXML escapes the five XML-significant characters in the
// substitution value. Without this, an authored value containing
// `<` or `&` would corrupt the document — Word/LibreOffice rejects
// the whole file on open.
func escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}

// runRe matches a single <w:r>...</w:r> element (non-greedy). Used
// by mergePlaceholderRuns to walk run boundaries.
var runRe = regexp.MustCompile(`<w:r\b[^>]*>[\s\S]*?</w:r>`)

// textRe extracts the text content from <w:t ...>...</w:t>.
var textRe = regexp.MustCompile(`<w:t[^>]*>([\s\S]*?)</w:t>`)

// mergePlaceholderRuns finds {{placeholder}} tokens split across
// consecutive <w:r> runs in OOXML and merges them into one run.
// Word/LibreOffice may arbitrarily split a token like "{{name}}"
// into multiple runs ("<w:r><w:t>{{</w:t></w:r><w:r><w:t>name}}</w:t></w:r>");
// this normalisation precedes the string replacement step so the
// substitution doesn't silently miss those splits.
func mergePlaceholderRuns(xml string) string {
	runs := runRe.FindAllStringIndex(xml, -1)
	if len(runs) < 2 {
		return xml
	}

	type runInfo struct {
		start, end int
		text       string
		full       string
	}

	infos := make([]runInfo, len(runs))
	for i, loc := range runs {
		full := xml[loc[0]:loc[1]]
		text := ""
		if m := textRe.FindStringSubmatch(full); len(m) > 1 {
			text = m[1]
		}
		infos[i] = runInfo{start: loc[0], end: loc[1], text: text, full: full}
	}

	var result strings.Builder
	prev := 0

	for i := 0; i < len(infos); {
		ri := infos[i]
		openIdx := strings.LastIndex(ri.text, "{{")
		closeIdx := strings.LastIndex(ri.text, "}}")

		if openIdx == -1 || (closeIdx != -1 && closeIdx > openIdx) {
			result.WriteString(xml[prev:ri.end])
			prev = ri.end
			i++
			continue
		}

		// Unmatched {{ — accumulate subsequent runs until }}.
		merged := ri.text
		lastMerged := i
		for j := i + 1; j < len(infos); j++ {
			merged += infos[j].text
			lastMerged = j
			if strings.Contains(infos[j].text, "}}") {
				break
			}
		}

		result.WriteString(xml[prev:ri.start])
		firstRun := ri.full
		newRun := textRe.ReplaceAllStringFunc(firstRun, func(_ string) string {
			orig := textRe.FindString(firstRun)
			tagEnd := strings.Index(orig, ">")
			openTag := orig[:tagEnd+1]
			return openTag + merged + "</w:t>"
		})
		result.WriteString(newRun)

		prev = infos[lastMerged].end
		i = lastMerged + 1
	}

	result.WriteString(xml[prev:])
	return result.String()
}

// Compile-time assertion that DOCXTemplateFiller satisfies the
// TemplateFiller contract.
var _ TemplateFiller = (*DOCXTemplateFiller)(nil)
