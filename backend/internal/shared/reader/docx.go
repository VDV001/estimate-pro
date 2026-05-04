package reader

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"strings"
)

// DOCXReader extracts plain text from DOCX bytes via stdlib zip+xml,
// avoiding AGPL Office libraries.
type DOCXReader struct{}

func NewDOCXReader() *DOCXReader { return &DOCXReader{} }

func (r *DOCXReader) Supports(ft FileType) bool { return ft == FileTypeDOCX }

func (r *DOCXReader) Parse(_ context.Context, filename string, data []byte) (string, error) {
	if len(data) == 0 {
		return "", fmt.Errorf("docx %q: %w", filename, ErrEmptyContent)
	}

	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", fmt.Errorf("docx %q: %w (%v)", filename, ErrCorruptedFile, err)
	}

	for _, f := range zr.File {
		if f.Name == "word/document.xml" {
			text, err := readDocxBody(f)
			if err != nil {
				return "", fmt.Errorf("docx %q: %w (%v)", filename, ErrCorruptedFile, err)
			}
			return text, nil
		}
	}

	return "", fmt.Errorf("docx %q: %w (missing word/document.xml)", filename, ErrCorruptedFile)
}

type docxBody struct {
	Paragraphs []docxParagraph `xml:"body>p"`
}

type docxParagraph struct {
	Runs []docxRun `xml:"r"`
}

type docxRun struct {
	Text string `xml:"t"`
}

func readDocxBody(f *zip.File) (string, error) {
	rc, err := f.Open()
	if err != nil {
		return "", err
	}
	defer func() { _ = rc.Close() }()

	var doc docxBody
	if err := xml.NewDecoder(rc).Decode(&doc); err != nil {
		return "", err
	}

	var sb strings.Builder
	for i, p := range doc.Paragraphs {
		if i > 0 {
			sb.WriteByte('\n')
		}
		for _, run := range p.Runs {
			sb.WriteString(run.Text)
		}
	}
	return strings.TrimSpace(sb.String()), nil
}
