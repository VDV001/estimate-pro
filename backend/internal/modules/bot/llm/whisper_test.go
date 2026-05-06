// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package llm_test

import (
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/VDV001/estimate-pro/backend/internal/modules/bot/llm"
	"github.com/VDV001/estimate-pro/backend/internal/modules/bot/usecase"
)

// fakeOgg is a tiny payload — the adapter never decodes it, only
// forwards as multipart bytes.
var fakeOgg = []byte("OggS\x00\x02\x00\x00\x00\x00\x00\x00")

func TestWhisperAdapter_ImplementsSpeechRecognizer(t *testing.T) {
	t.Parallel()
	var _ usecase.SpeechRecognizer = (*llm.WhisperAdapter)(nil)
}

// TestWhisperAdapter_RecognizeAudio_HappyPath asserts the wire
// contract: POST /v1/audio/transcriptions with Bearer auth and a
// multipart body carrying file+model fields, then unmarshal the
// {"text":"..."} response.
func TestWhisperAdapter_RecognizeAudio_HappyPath(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var (
		gotAuth     string
		gotPath     string
		gotModel    string
		gotFile     []byte
		gotFileName string
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")

		mediaType, params, err := mimeFromContentType(r.Header.Get("Content-Type"))
		if err != nil {
			http.Error(w, "bad content type", http.StatusBadRequest)
			return
		}
		if mediaType != "multipart/form-data" {
			http.Error(w, "expected multipart", http.StatusBadRequest)
			return
		}

		mr := multipart.NewReader(r.Body, params["boundary"])
		for {
			part, err := mr.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			data, _ := io.ReadAll(part)
			switch part.FormName() {
			case "file":
				gotFile = data
				gotFileName = part.FileName()
			case "model":
				gotModel = string(data)
			}
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"text":"привет это голосовое сообщение"}`))
	}))
	defer server.Close()

	adapter := llm.NewWhisperAdapterWithClient("test-key", "whisper-1", server.URL, server.Client())
	got, err := adapter.RecognizeAudio(ctx, fakeOgg, "audio/ogg")
	if err != nil {
		t.Fatalf("RecognizeAudio: unexpected error: %v", err)
	}
	if got != "привет это голосовое сообщение" {
		t.Fatalf("RecognizeAudio: got %q", got)
	}
	if gotPath != "/v1/audio/transcriptions" {
		t.Errorf("path: got %q, want /v1/audio/transcriptions", gotPath)
	}
	if gotAuth != "Bearer test-key" {
		t.Errorf("auth: got %q, want %q", gotAuth, "Bearer test-key")
	}
	if gotModel != "whisper-1" {
		t.Errorf("model: got %q, want whisper-1", gotModel)
	}
	if string(gotFile) != string(fakeOgg) {
		t.Errorf("file bytes mismatch: got %q", gotFile)
	}
	if !strings.HasSuffix(gotFileName, ".ogg") {
		t.Errorf("filename: got %q, want *.ogg (Whisper detects format from extension)", gotFileName)
	}
}

func TestWhisperAdapter_RecognizeAudio_HTTPError(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"upstream"}`))
	}))
	defer server.Close()

	adapter := llm.NewWhisperAdapterWithClient("test-key", "whisper-1", server.URL, server.Client())
	if _, err := adapter.RecognizeAudio(ctx, fakeOgg, "audio/ogg"); err == nil {
		t.Fatal("RecognizeAudio: expected error on 500, got nil")
	}
}

func TestWhisperAdapter_RecognizeAudio_FilenameByMime(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		mime        string
		wantFileExt string
	}{
		{"ogg", "audio/ogg", ".ogg"},
		{"mpeg", "audio/mpeg", ".mp3"},
		{"wav", "audio/wav", ".wav"},
		{"unknown defaults to ogg", "application/octet-stream", ".ogg"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			var gotFileName string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, params, _ := mimeFromContentType(r.Header.Get("Content-Type"))
				mr := multipart.NewReader(r.Body, params["boundary"])
				for {
					part, err := mr.NextPart()
					if err == io.EOF {
						break
					}
					if err != nil {
						http.Error(w, err.Error(), http.StatusBadRequest)
						return
					}
					_, _ = io.Copy(io.Discard, part)
					if part.FormName() == "file" {
						gotFileName = part.FileName()
					}
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"text":"ok"}`))
			}))
			defer server.Close()

			adapter := llm.NewWhisperAdapterWithClient("test-key", "whisper-1", server.URL, server.Client())
			if _, err := adapter.RecognizeAudio(ctx, fakeOgg, tc.mime); err != nil {
				t.Fatalf("RecognizeAudio: unexpected error: %v", err)
			}
			if !strings.HasSuffix(gotFileName, tc.wantFileExt) {
				t.Errorf("filename for mime %q: got %q, want suffix %q", tc.mime, gotFileName, tc.wantFileExt)
			}
		})
	}
}

// mimeFromContentType is a tiny shim so tests can pull the boundary
// param out of a Content-Type header without each case duplicating
// mime.ParseMediaType.
func mimeFromContentType(value string) (string, map[string]string, error) {
	return mimeParseMedia(value)
}
