// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package telegram

import "encoding/json"

// Update represents a Telegram Bot API update.
type Update struct {
	UpdateID      int64          `json:"update_id"`
	Message       *Message       `json:"message,omitempty"`
	CallbackQuery *CallbackQuery `json:"callback_query,omitempty"`
}

// Message represents a Telegram message.
type Message struct {
	MessageID      int64       `json:"message_id"`
	From           *User       `json:"from,omitempty"`
	Chat           *Chat       `json:"chat"`
	Text           string      `json:"text,omitempty"`
	Document       *Document   `json:"document,omitempty"`
	Photo          []PhotoSize `json:"photo,omitempty"`
	Voice          *Voice      `json:"voice,omitempty"`
	ReplyToMessage *Message    `json:"reply_to_message,omitempty"`
}

// PhotoSize represents one of the resolutions Telegram offers for an
// uploaded photo. Telegram includes several entries in message.photo
// — small (thumb) through full size — and the bot picks the
// highest-resolution variant for OCR. file_unique_id is intentionally
// omitted: only file_id is needed to download.
type PhotoSize struct {
	FileID   string `json:"file_id"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	FileSize int64  `json:"file_size,omitempty"`
}

// Voice represents a Telegram voice message. Telegram always encodes
// voice as ogg/opus, regardless of the source platform — Whisper
// accepts that container directly, so the bot forwards the bytes
// verbatim into RecognizeAudio.
type Voice struct {
	FileID   string `json:"file_id"`
	MimeType string `json:"mime_type,omitempty"`
	Duration int    `json:"duration"`
	FileSize int64  `json:"file_size,omitempty"`
}

// User represents a Telegram user.
type User struct {
	ID        int64  `json:"id"`
	IsBot     bool   `json:"is_bot"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name,omitempty"`
	Username  string `json:"username,omitempty"`
}

// Chat represents a Telegram chat.
type Chat struct {
	ID    int64  `json:"id"`
	Type  string `json:"type"`
	Title string `json:"title,omitempty"`
}

// Document represents a Telegram document (file).
type Document struct {
	FileID   string `json:"file_id"`
	FileName string `json:"file_name,omitempty"`
	MimeType string `json:"mime_type,omitempty"`
	FileSize int64  `json:"file_size,omitempty"`
}

// CallbackQuery represents a Telegram callback query from an inline keyboard.
type CallbackQuery struct {
	ID      string   `json:"id"`
	From    *User    `json:"from"`
	Message *Message `json:"message,omitempty"`
	Data    string   `json:"data,omitempty"`
}

// InlineKeyboardMarkup represents an inline keyboard attached to a message.
type InlineKeyboardMarkup struct {
	InlineKeyboard [][]InlineKeyboardButton `json:"inline_keyboard"`
}

// InlineKeyboardButton represents a button in an inline keyboard.
type InlineKeyboardButton struct {
	Text         string `json:"text"`
	CallbackData string `json:"callback_data"`
}

// SendMessageRequest represents a request to send a message via the Telegram Bot API.
type SendMessageRequest struct {
	ChatID      string                `json:"chat_id"`
	Text        string                `json:"text"`
	ParseMode   string                `json:"parse_mode,omitempty"`
	ReplyMarkup *InlineKeyboardMarkup `json:"reply_markup,omitempty"`
}

// APIResponse represents a generic Telegram Bot API response.
type APIResponse struct {
	OK          bool            `json:"ok"`
	Description string          `json:"description,omitempty"`
	Result      json.RawMessage `json:"result,omitempty"`
}

// FileResponse represents a Telegram Bot API getFile response.
type FileResponse struct {
	OK     bool `json:"ok"`
	Result struct {
		FilePath string `json:"file_path"`
	} `json:"result"`
}
