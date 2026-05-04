// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package usecase

import "github.com/VDV001/estimate-pro/backend/internal/shared/security"

// isPromptInjection delegates to the shared injection detector. The bot
// caller (BotUsecase.ProcessMessage) wraps this with chat_id/user_id
// observability — the shared detector stays a pure function.
func isPromptInjection(text string) bool {
	return security.IsPromptInjection(text)
}

// deflectionResponse returns one playful deflection from the shared pool.
func deflectionResponse() string {
	return security.PromptInjectionDeflection()
}
