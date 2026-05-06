// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package telegram

import (
	"encoding/json"
	"testing"
)

// TestUpdate_PhotoMessage verifies that an incoming Telegram update
// carrying message.photo (PhotoSize array) round-trips into Update.
// The test pins the highest-resolution photo's file_id, file_size,
// width, and height — the bot's webhook handler picks the last
// PhotoSize for OCR, so all four fields must survive unmarshalling.
func TestUpdate_PhotoMessage(t *testing.T) {
	t.Parallel()

	raw := []byte(`{
		"update_id": 100,
		"message": {
			"message_id": 7,
			"from": {"id": 1, "is_bot": false, "first_name": "Daniil"},
			"chat": {"id": -42, "type": "private"},
			"photo": [
				{"file_id": "small-id", "width": 90, "height": 60, "file_size": 1500},
				{"file_id": "large-id", "width": 1280, "height": 720, "file_size": 250000}
			]
		}
	}`)

	var update Update
	if err := json.Unmarshal(raw, &update); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if update.Message == nil {
		t.Fatal("Message: nil")
	}
	if got := len(update.Message.Photo); got != 2 {
		t.Fatalf("Photo length: got %d, want 2", got)
	}

	largest := update.Message.Photo[len(update.Message.Photo)-1]
	if largest.FileID != "large-id" {
		t.Errorf("largest.FileID: got %q, want %q", largest.FileID, "large-id")
	}
	if largest.Width != 1280 || largest.Height != 720 {
		t.Errorf("largest dims: got %dx%d, want 1280x720", largest.Width, largest.Height)
	}
	if largest.FileSize != 250000 {
		t.Errorf("largest.FileSize: got %d, want 250000", largest.FileSize)
	}
}

// TestUpdate_VoiceMessage verifies that message.voice round-trips
// into Update. The bot reads file_id (to download) and mime_type (to
// pass to Whisper for filename inference).
func TestUpdate_VoiceMessage(t *testing.T) {
	t.Parallel()

	raw := []byte(`{
		"update_id": 101,
		"message": {
			"message_id": 8,
			"from": {"id": 1, "is_bot": false, "first_name": "Daniil"},
			"chat": {"id": -42, "type": "private"},
			"voice": {
				"file_id": "voice-id",
				"mime_type": "audio/ogg",
				"duration": 5,
				"file_size": 12345
			}
		}
	}`)

	var update Update
	if err := json.Unmarshal(raw, &update); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if update.Message == nil {
		t.Fatal("Message: nil")
	}
	v := update.Message.Voice
	if v == nil {
		t.Fatal("Voice: nil")
	}
	if v.FileID != "voice-id" {
		t.Errorf("FileID: got %q, want %q", v.FileID, "voice-id")
	}
	if v.MimeType != "audio/ogg" {
		t.Errorf("MimeType: got %q, want %q", v.MimeType, "audio/ogg")
	}
	if v.Duration != 5 {
		t.Errorf("Duration: got %d, want 5", v.Duration)
	}
	if v.FileSize != 12345 {
		t.Errorf("FileSize: got %d, want 12345", v.FileSize)
	}
}
