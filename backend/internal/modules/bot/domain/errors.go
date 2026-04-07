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
)
