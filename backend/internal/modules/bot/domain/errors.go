// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package domain

import "errors"

var (
	// ErrUserNotLinked is returned when a Telegram user has no linked EstimatePro account.
	ErrUserNotLinked = errors.New("bot: telegram user is not linked to an account")

	// ErrSessionExpired is returned when a bot session has expired.
	ErrSessionExpired = errors.New("bot: session has expired")

	// ErrSessionNotFound is returned when a bot session is not found.
	ErrSessionNotFound = errors.New("bot: session not found")

	// ErrInvalidIntent is returned when a parsed intent is not valid.
	ErrInvalidIntent = errors.New("bot: invalid intent")

	// ErrUnsupportedProvider is returned when an LLM provider is not supported.
	ErrUnsupportedProvider = errors.New("bot: unsupported LLM provider")

	// ErrNoLLMConfig is returned when no LLM configuration is available.
	ErrNoLLMConfig = errors.New("bot: no LLM configuration found")

	// ErrBotNotMentioned is returned when the bot is not mentioned in a group message.
	ErrBotNotMentioned = errors.New("bot: bot was not mentioned in the message")

	// ErrMissingChat is returned when a session or memory entry has no chat id.
	ErrMissingChat = errors.New("bot: chat id is required")

	// ErrMissingUser is returned when a session or memory entry has no user id.
	ErrMissingUser = errors.New("bot: user id is required")

	// ErrInvalidTTL is returned when a session ttl is non-positive.
	ErrInvalidTTL = errors.New("bot: session ttl must be positive")

	// ErrInvalidRole is returned when a memory entry role is not "user" or "esti".
	ErrInvalidRole = errors.New("bot: memory role must be user or esti")

	// ErrEmptyContent is returned when a memory entry content is empty.
	ErrEmptyContent = errors.New("bot: memory content must be non-empty")

	// ErrUserNotFound is returned when no EstimatePro user matches the given Telegram ID.
	ErrUserNotFound = errors.New("bot: no user found for telegram id")
)
