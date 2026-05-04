// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package llm

import (
	"strings"
	"testing"
	"time"
)

func TestBodyPreview_TruncatesLargeBody(t *testing.T) {
	body := []byte(strings.Repeat("a", 500))
	preview := bodyPreview(body)
	if len(preview) > bodyPreviewLen {
		t.Errorf("preview length %d exceeds limit %d", len(preview), bodyPreviewLen)
	}
	if len(preview) != bodyPreviewLen {
		t.Errorf("preview should be exactly bodyPreviewLen (%d) for 500-byte input, got %d", bodyPreviewLen, len(preview))
	}
}

func TestBodyPreview_LeavesSmallBodyIntact(t *testing.T) {
	body := []byte("short response")
	if got := bodyPreview(body); got != "short response" {
		t.Errorf("bodyPreview = %q, want %q", got, "short response")
	}
}

func TestBodyPreview_EmptyReturnsEmpty(t *testing.T) {
	if got := bodyPreview(nil); got != "" {
		t.Errorf("bodyPreview(nil) = %q, want empty", got)
	}
	if got := bodyPreview([]byte{}); got != "" {
		t.Errorf("bodyPreview(empty) = %q, want empty", got)
	}
}

func TestBodyPreview_ExactlyAtBoundary(t *testing.T) {
	body := []byte(strings.Repeat("x", bodyPreviewLen))
	if got := bodyPreview(body); len(got) != bodyPreviewLen {
		t.Errorf("at boundary: len(preview) = %d, want %d", len(got), bodyPreviewLen)
	}
}

func TestDefaultHTTPTimeout_IsReasonable(t *testing.T) {
	// Sanity bounds — too short = false negatives on long prompts;
	// too long = goroutine leak risk on hung connections.
	if defaultHTTPTimeout < 10*time.Second {
		t.Errorf("defaultHTTPTimeout (%v) too short — risks false negatives", defaultHTTPTimeout)
	}
	if defaultHTTPTimeout > 5*time.Minute {
		t.Errorf("defaultHTTPTimeout (%v) too long — goroutine leak risk", defaultHTTPTimeout)
	}
}
