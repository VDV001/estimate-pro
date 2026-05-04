package reader

import (
	"context"
	"fmt"
)

// MDReader returns Markdown bytes as text verbatim. Downstream LLM
// consumers handle the markup themselves, so no parser is involved.
// Unlike TXTReader, MDReader does not enforce UTF-8 validity:
// embedded code fences may legitimately carry arbitrary bytes
// (base64, binary samples) and rejecting them would harm real-world
// extraction more than it would protect callers.
type MDReader struct{}

func NewMDReader() *MDReader { return &MDReader{} }

func (r *MDReader) Supports(ft FileType) bool { return ft == FileTypeMD }

func (r *MDReader) Parse(ctx context.Context, filename string, data []byte) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if len(data) == 0 {
		return "", fmt.Errorf("md %q: %w", filename, ErrEmptyContent)
	}
	return string(data), nil
}
