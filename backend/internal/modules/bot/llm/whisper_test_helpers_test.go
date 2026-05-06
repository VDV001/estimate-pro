// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package llm_test

import "mime"

// mimeParseMedia wraps mime.ParseMediaType so the test file does not
// need a direct stdlib import (keeps the multipart-handling closure
// readable).
func mimeParseMedia(value string) (string, map[string]string, error) {
	return mime.ParseMediaType(value)
}
