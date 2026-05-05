// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package generator

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"
)

// GotenbergConverter posts a document to a Gotenberg sidecar's
// /forms/libreoffice/convert endpoint and returns the rendered
// PDF bytes. Single dependency: the sidecar URL + a per-call
// timeout. The HTTP client is configured at construction so
// goroutines can reuse the same converter without contention.
//
// Resilience contract: 5xx and network-level failures surface as
// ErrGotenbergUnavailable (caller may retry); 4xx surface as
// ErrInvalidConversionInput (caller fails fast). 2xx success
// streams the response body verbatim — Gotenberg already sets
// Content-Type: application/pdf, so we trust the body.
type GotenbergConverter struct {
	baseURL string
	client  *http.Client
}

// NewGotenbergConverter pins the sidecar base URL and the
// per-request timeout. The base URL is everything before
// /forms/libreoffice/convert (e.g. "http://gotenberg:3000"); the
// converter appends the path internally. Timeout covers the full
// round-trip (dial + send body + receive PDF) — a 5-minute upper
// bound matches the river job timeout in PR-B3 ADR-016.
func NewGotenbergConverter(baseURL string, timeout time.Duration) *GotenbergConverter {
	return &GotenbergConverter{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{Timeout: timeout},
	}
}

// Convert builds a multipart/form-data body with the input bytes
// (named "files" per Gotenberg's convention) and posts it to the
// libreoffice convert endpoint. Empty input short-circuits with
// ErrEmptyTemplate so the wire is never touched.
func (c *GotenbergConverter) Convert(ctx context.Context, input []byte, filename string) ([]byte, error) {
	if len(input) == 0 {
		return nil, fmt.Errorf("gotenberg: %w", ErrEmptyTemplate)
	}

	body, contentType, err := buildMultipartBody(input, filename)
	if err != nil {
		return nil, fmt.Errorf("gotenberg: build multipart: %w", err)
	}

	endpoint := c.baseURL + "/forms/libreoffice/convert"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, body)
	if err != nil {
		return nil, fmt.Errorf("gotenberg: build request: %w", err)
	}
	req.Header.Set("Content-Type", contentType)

	resp, err := c.client.Do(req)
	if err != nil {
		// Network-level failure (dial refused, TLS, DNS, timeout).
		// All collapse onto ErrGotenbergUnavailable — the caller
		// branches on retry-vs-fail by sentinel, not by sub-class.
		return nil, fmt.Errorf("gotenberg: %w: %v", ErrGotenbergUnavailable, err)
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode >= 500:
		return nil, fmt.Errorf("gotenberg: %w: status %d", ErrGotenbergUnavailable, resp.StatusCode)
	case resp.StatusCode >= 400:
		return nil, fmt.Errorf("gotenberg: %w: status %d", ErrInvalidConversionInput, resp.StatusCode)
	}

	pdf, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("gotenberg: read body: %w", err)
	}
	return pdf, nil
}

// buildMultipartBody assembles a "files" form part carrying the
// document bytes under the supplied filename, then closes the
// writer so the caller can post the buffer directly. Gotenberg
// recognises the part name "files" specifically — any other name
// is rejected upstream.
func buildMultipartBody(input []byte, filename string) (*bytes.Buffer, string, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, err := w.CreateFormFile("files", filename)
	if err != nil {
		return nil, "", fmt.Errorf("create form file: %w", err)
	}
	if _, err := fw.Write(input); err != nil {
		return nil, "", fmt.Errorf("write form body: %w", err)
	}
	if err := w.Close(); err != nil {
		return nil, "", fmt.Errorf("close multipart writer: %w", err)
	}
	return &buf, w.FormDataContentType(), nil
}

// Compile-time assertion.
var _ Converter = (*GotenbergConverter)(nil)
