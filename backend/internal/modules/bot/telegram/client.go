// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/VDV001/estimate-pro/backend/internal/modules/bot/domain"
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
	req := SendMessageRequest{
		ChatID: chatID,
		Text:   text,
	}
	_, err := c.doRequest(ctx, "sendMessage", req)
	if err != nil {
		return fmt.Errorf("telegram.Client.SendMessage: %w", err)
	}
	return nil
}

// SendMarkdown sends a Markdown-formatted message to a chat.
func (c *Client) SendMarkdown(ctx context.Context, chatID, text string) error {
	req := SendMessageRequest{
		ChatID:    chatID,
		Text:      text,
		ParseMode: "Markdown",
	}
	_, err := c.doRequest(ctx, "sendMessage", req)
	if err != nil {
		return fmt.Errorf("telegram.Client.SendMarkdown: %w", err)
	}
	return nil
}

// SendInlineKeyboard sends a message with an inline keyboard to a chat.
func (c *Client) SendInlineKeyboard(ctx context.Context, chatID, text string, keyboard [][]domain.InlineKeyboardButton) error {
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
		return fmt.Errorf("telegram.Client.SendInlineKeyboard: %w", err)
	}
	return nil
}

// AnswerCallbackQuery answers an incoming callback query.
func (c *Client) AnswerCallbackQuery(ctx context.Context, callbackQueryID, text string) error {
	payload := struct {
		CallbackQueryID string `json:"callback_query_id"`
		Text            string `json:"text,omitempty"`
	}{
		CallbackQueryID: callbackQueryID,
		Text:            text,
	}
	_, err := c.doRequest(ctx, "answerCallbackQuery", payload)
	if err != nil {
		return fmt.Errorf("telegram.Client.AnswerCallbackQuery: %w", err)
	}
	return nil
}

// SetReaction sets an emoji reaction on a message.
func (c *Client) SetReaction(ctx context.Context, chatID string, messageID int64, emoji string) error {
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
		return fmt.Errorf("telegram.Client.SetReaction: %w", err)
	}
	return nil
}

// GetFileURL retrieves the download URL for a file by its file ID.
func (c *Client) GetFileURL(ctx context.Context, fileID string) (string, error) {
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
	return downloadURL, nil
}

// DownloadFile downloads a file from the given URL and returns its content.
func (c *Client) DownloadFile(ctx context.Context, url string) ([]byte, error) {
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
	return data, nil
}

// doRequest marshals the payload, sends a POST request to the Telegram Bot API,
// and returns the parsed API response.
func (c *Client) doRequest(ctx context.Context, method string, payload any) (*APIResponse, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/"+method, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var apiResp APIResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	if !apiResp.OK {
		return nil, fmt.Errorf("API error: %s", apiResp.Description)
	}

	return &apiResp, nil
}
