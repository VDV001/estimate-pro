// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/VDV001/estimate-pro/backend/internal/modules/bot/domain"
)

const (
	maxRetries   = 2 // up to 3 attempts total (initial + 2 retries)
	baseBackoff  = 500 * time.Millisecond
)

// Client is a Telegram Bot API client.
type Client struct {
	token      string
	httpClient *http.Client
	baseURL    string
}

// NewClient creates a new Telegram Bot API client.
func NewClient(token string) *Client {
	return &Client{
		token:      token,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    fmt.Sprintf("https://api.telegram.org/bot%s", token),
	}
}

// SendMessage sends a plain text message to a chat.
func (c *Client) SendMessage(ctx context.Context, chatID, text string) error {
	slog.DebugContext(ctx, "telegram.SendMessage", slog.String("chat_id", chatID), slog.Int("text_len", len(text)))
	req := SendMessageRequest{
		ChatID: chatID,
		Text:   text,
	}
	_, err := c.doRequest(ctx, "sendMessage", req)
	if err != nil {
		slog.ErrorContext(ctx, "telegram.SendMessage failed", slog.String("chat_id", chatID), slog.String("error", err.Error()))
		return fmt.Errorf("telegram.Client.SendMessage: %w", err)
	}
	return nil
}

// SendMarkdown sends a Markdown-formatted message to a chat.
func (c *Client) SendMarkdown(ctx context.Context, chatID, text string) error {
	slog.DebugContext(ctx, "telegram.SendMarkdown", slog.String("chat_id", chatID), slog.Int("text_len", len(text)))
	req := SendMessageRequest{
		ChatID:    chatID,
		Text:      text,
		ParseMode: "Markdown",
	}
	_, err := c.doRequest(ctx, "sendMessage", req)
	if err != nil {
		slog.ErrorContext(ctx, "telegram.SendMarkdown failed", slog.String("chat_id", chatID), slog.String("error", err.Error()))
		return fmt.Errorf("telegram.Client.SendMarkdown: %w", err)
	}
	return nil
}

// SendInlineKeyboard sends a message with an inline keyboard to a chat.
func (c *Client) SendInlineKeyboard(ctx context.Context, chatID, text string, keyboard [][]domain.InlineKeyboardButton) error {
	slog.DebugContext(ctx, "telegram.SendInlineKeyboard", slog.String("chat_id", chatID), slog.Int("rows", len(keyboard)))
	// Convert domain buttons to telegram wire format.
	tgKeyboard := make([][]InlineKeyboardButton, len(keyboard))
	for i, row := range keyboard {
		tgKeyboard[i] = make([]InlineKeyboardButton, len(row))
		for j, btn := range row {
			tgKeyboard[i][j] = InlineKeyboardButton{
				Text:         btn.Text,
				CallbackData: btn.CallbackData,
			}
		}
	}
	req := SendMessageRequest{
		ChatID: chatID,
		Text:   text,
		ReplyMarkup: &InlineKeyboardMarkup{
			InlineKeyboard: tgKeyboard,
		},
	}
	_, err := c.doRequest(ctx, "sendMessage", req)
	if err != nil {
		slog.ErrorContext(ctx, "telegram.SendInlineKeyboard failed", slog.String("chat_id", chatID), slog.String("error", err.Error()))
		return fmt.Errorf("telegram.Client.SendInlineKeyboard: %w", err)
	}
	return nil
}

// AnswerCallbackQuery answers an incoming callback query.
func (c *Client) AnswerCallbackQuery(ctx context.Context, callbackQueryID, text string) error {
	slog.DebugContext(ctx, "telegram.AnswerCallbackQuery", slog.String("callback_query_id", callbackQueryID))
	payload := struct {
		CallbackQueryID string `json:"callback_query_id"`
		Text            string `json:"text,omitempty"`
	}{
		CallbackQueryID: callbackQueryID,
		Text:            text,
	}
	_, err := c.doRequest(ctx, "answerCallbackQuery", payload)
	if err != nil {
		slog.ErrorContext(ctx, "telegram.AnswerCallbackQuery failed", slog.String("callback_query_id", callbackQueryID), slog.String("error", err.Error()))
		return fmt.Errorf("telegram.Client.AnswerCallbackQuery: %w", err)
	}
	return nil
}

// SetReaction sets an emoji reaction on a message.
func (c *Client) SetReaction(ctx context.Context, chatID string, messageID int64, emoji string) error {
	slog.DebugContext(ctx, "telegram.SetReaction", slog.String("chat_id", chatID), slog.Int64("message_id", messageID), slog.String("emoji", emoji))
	payload := struct {
		ChatID    string `json:"chat_id"`
		MessageID int64  `json:"message_id"`
		Reaction  []struct {
			Type  string `json:"type"`
			Emoji string `json:"emoji"`
		} `json:"reaction"`
	}{
		ChatID:    chatID,
		MessageID: messageID,
		Reaction: []struct {
			Type  string `json:"type"`
			Emoji string `json:"emoji"`
		}{{Type: "emoji", Emoji: emoji}},
	}
	_, err := c.doRequest(ctx, "setMessageReaction", payload)
	if err != nil {
		// REACTION_INVALID is expected in chats with restricted reactions — not an error.
		if strings.Contains(err.Error(), "REACTION_INVALID") {
			slog.DebugContext(ctx, "telegram.SetReaction: reaction not available in this chat", slog.String("chat_id", chatID), slog.String("emoji", emoji))
			return nil
		}
		slog.WarnContext(ctx, "telegram.SetReaction failed", slog.String("chat_id", chatID), slog.String("error", err.Error()))
		return fmt.Errorf("telegram.Client.SetReaction: %w", err)
	}
	return nil
}

// GetFileURL retrieves the download URL for a file by its file ID.
func (c *Client) GetFileURL(ctx context.Context, fileID string) (string, error) {
	slog.DebugContext(ctx, "telegram.GetFileURL", slog.String("file_id", fileID))
	payload := struct {
		FileID string `json:"file_id"`
	}{
		FileID: fileID,
	}
	resp, err := c.doRequest(ctx, "getFile", payload)
	if err != nil {
		return "", fmt.Errorf("telegram.Client.GetFileURL: %w", err)
	}

	var fileResp struct {
		FilePath string `json:"file_path"`
	}
	if err := json.Unmarshal(resp.Result, &fileResp); err != nil {
		return "", fmt.Errorf("telegram.Client.GetFileURL: unmarshal result: %w", err)
	}

	downloadURL := fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", c.token, fileResp.FilePath)
	slog.DebugContext(ctx, "telegram.GetFileURL resolved", slog.String("file_id", fileID), slog.String("file_path", fileResp.FilePath))
	return downloadURL, nil
}

// DownloadFile downloads a file from the given URL and returns its content.
func (c *Client) DownloadFile(ctx context.Context, url string) ([]byte, error) {
	slog.DebugContext(ctx, "telegram.DownloadFile", slog.String("url", url))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("telegram.Client.DownloadFile: create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("telegram.Client.DownloadFile: execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("telegram.Client.DownloadFile: unexpected status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("telegram.Client.DownloadFile: read body: %w", err)
	}
	slog.DebugContext(ctx, "telegram.DownloadFile complete", slog.Int("bytes", len(data)))
	return data, nil
}

// doRequest marshals the payload, sends a POST request to the Telegram Bot API,
// and returns the parsed API response. Retries on transient errors (5xx, 429, network).
func (c *Client) doRequest(ctx context.Context, method string, payload any) (*APIResponse, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}

	var lastErr error
	for attempt := range maxRetries + 1 {
		if attempt > 0 {
			slog.WarnContext(ctx, "telegram.doRequest: retrying",
				slog.String("method", method),
				slog.Int("attempt", attempt+1),
				slog.String("prev_error", lastErr.Error()),
			)
		}

		var apiResp *APIResponse
		var retryAfter time.Duration
		apiResp, retryAfter, lastErr = c.doRequestOnce(ctx, method, body)
		if lastErr == nil {
			return apiResp, nil
		}

		// Don't retry on non-transient errors (4xx except 429).
		if retryAfter < 0 {
			return nil, lastErr
		}

		if attempt < maxRetries {
			backoff := retryAfter
			if backoff == 0 {
				backoff = baseBackoff << attempt // 500ms, 1s
			}
			select {
			case <-ctx.Done():
				return nil, fmt.Errorf("context cancelled during retry: %w", ctx.Err())
			case <-time.After(backoff):
			}
		}
	}
	return nil, lastErr
}

// doRequestOnce executes a single HTTP request. Returns:
//   - response on success
//   - retryAfter > 0 for 429 with Retry-After header
//   - retryAfter == 0 for retryable errors (5xx, network)
//   - retryAfter < 0 for non-retryable errors (4xx)
func (c *Client) doRequestOnce(ctx context.Context, method string, body []byte) (*APIResponse, time.Duration, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/"+method, bytes.NewReader(body))
	if err != nil {
		return nil, -1, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		// Network errors (TLS timeout, connection refused) are retryable.
		return nil, 0, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	// HTTP 5xx — retryable.
	if resp.StatusCode >= 500 {
		respBody, _ := io.ReadAll(resp.Body)
		var apiResp APIResponse
		if json.Unmarshal(respBody, &apiResp) == nil && apiResp.Description != "" {
			slog.WarnContext(ctx, "telegram.doRequest: API error", slog.String("method", method), slog.String("description", apiResp.Description))
			return nil, 0, fmt.Errorf("API error: %s", apiResp.Description)
		}
		return nil, 0, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	// HTTP 429 — retryable with Retry-After.
	if resp.StatusCode == http.StatusTooManyRequests {
		retryAfter := parseRetryAfter(resp.Header.Get("Retry-After"))
		respBody, _ := io.ReadAll(resp.Body)
		var apiResp APIResponse
		if json.Unmarshal(respBody, &apiResp) == nil && apiResp.Description != "" {
			slog.WarnContext(ctx, "telegram.doRequest: rate limited", slog.String("method", method), slog.Duration("retry_after", retryAfter))
			return nil, retryAfter, fmt.Errorf("API error: %s", apiResp.Description)
		}
		return nil, retryAfter, fmt.Errorf("HTTP 429")
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, -1, fmt.Errorf("read response: %w", err)
	}

	var apiResp APIResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, -1, fmt.Errorf("unmarshal response: %w", err)
	}

	if !apiResp.OK {
		slog.WarnContext(ctx, "telegram.doRequest: API error", slog.String("method", method), slog.String("description", apiResp.Description))
		return nil, -1, fmt.Errorf("API error: %s", apiResp.Description)
	}

	return &apiResp, 0, nil
}

// parseRetryAfter parses the Retry-After header value as seconds.
func parseRetryAfter(s string) time.Duration {
	if s == "" {
		return baseBackoff
	}
	sec, err := strconv.Atoi(s)
	if err != nil {
		return baseBackoff
	}
	return time.Duration(sec) * time.Second
}
