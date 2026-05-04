package reader

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
)

// Composite dispatches Parse to the first registered DocumentReader
// that Supports the file's FileType. It enforces an optional pre-dispatch
// size guard (maxBytes <= 0 disables the check).
type Composite struct {
	readers  []DocumentReader
	maxBytes int64
}

func NewComposite(maxBytes int64, readers ...DocumentReader) *Composite {
	return &Composite{readers: readers, maxBytes: maxBytes}
}

func (c *Composite) Supports(ft FileType) bool {
	for _, r := range c.readers {
		if r.Supports(ft) {
			return true
		}
	}
	return false
}

func (c *Composite) Parse(ctx context.Context, filename string, data []byte) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if c.maxBytes > 0 && int64(len(data)) > c.maxBytes {
		return "", fmt.Errorf("composite %q: %w (size=%d, max=%d)", filename, ErrFileTooLarge, len(data), c.maxBytes)
	}

	ext := strings.TrimPrefix(filepath.Ext(filename), ".")
	ft, err := ParseFileType(ext)
	if err != nil {
		return "", fmt.Errorf("composite %q: %w (ext=%q)", filename, err, ext)
	}

	for _, r := range c.readers {
		if r.Supports(ft) {
			return r.Parse(ctx, filename, data)
		}
	}
	return "", fmt.Errorf("composite %q: %w (no reader for %s)", filename, ErrUnsupportedFormat, ft)
}
