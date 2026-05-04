package reader

import "context"

// DocumentReader extracts plain text from in-memory document bytes.
// Implementations must be stateless and safe for concurrent use.
type DocumentReader interface {
	Supports(ft FileType) bool
	Parse(ctx context.Context, filename string, data []byte) (string, error)
}
