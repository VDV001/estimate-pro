// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package llm

import "context"

// LLMConfigRepository is the persistence port for LLMConfig.
// Implementations live in module-specific packages — currently
// bot/repository/postgres.go owns the llm_configs table (migration
// 007_bot_module.sql). Table ownership migrates to a shared migration
// in a future PR when a second module needs to read configs.
//
// Method contracts:
//   - GetSystem returns the system-wide config (UserID == ""). Returns
//     ErrConfigNotFound if no system config row exists.
//   - GetByUserID returns the user-scoped config. Returns
//     ErrConfigNotFound if no row matches userID.
//   - Upsert inserts a new config or updates an existing one matched
//     on UserID. Empty UserID ("") is treated as the singleton
//     system-config row — implementations must enforce uniqueness
//     (e.g. via a partial UNIQUE index `WHERE user_id IS NULL` or by
//     converting "" to NULL at the SQL boundary). The repository
//     assigns ID if cfg.ID is empty.
//
// Implementations must propagate errors.Is(err, ErrConfigNotFound) for
// the not-found case so callers can fall back to env config or system
// config without parsing error strings.
type LLMConfigRepository interface {
	GetSystem(ctx context.Context) (*LLMConfig, error)
	GetByUserID(ctx context.Context, userID string) (*LLMConfig, error)
	Upsert(ctx context.Context, cfg *LLMConfig) error
}
