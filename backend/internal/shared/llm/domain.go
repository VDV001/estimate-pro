// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package llm

import (
	"errors"
	"fmt"
	"time"
)

// LLMProviderType is the canonical provider identifier persisted in
// llm_configs.provider and used by the factory to dispatch to an adapter.
type LLMProviderType string

const (
	ProviderClaude LLMProviderType = "claude"
	ProviderOpenAI LLMProviderType = "openai"
	ProviderGrok   LLMProviderType = "grok"
	ProviderOllama LLMProviderType = "ollama"
)

// IsValid reports whether the provider value matches a known adapter.
func (p LLMProviderType) IsValid() bool {
	switch p {
	case ProviderClaude, ProviderOpenAI, ProviderGrok, ProviderOllama:
		return true
	}
	return false
}

// Sentinel errors. Callers must use errors.Is — direct string comparison
// is forbidden by the project's domain-error convention.
var (
	ErrInvalidProvider = errors.New("llm: invalid provider type")
	ErrEmptyModel      = errors.New("llm: model must not be empty")
	ErrEmptyAPIKey     = errors.New("llm: api_key must not be empty for non-local providers")
	ErrConfigNotFound  = errors.New("llm: config not found")
)

// LLMConfig is the persisted configuration for one LLM provider, scoped
// to a user (UserID != "") or system-wide (UserID == ""). Created via
// NewLLMConfig — direct struct literals outside this package are
// rejected at code review (DDD gate).
//
// APIKey is intentionally not tagged with json:"-" here; transport-layer
// DTOs are responsible for redaction. The field stays exported for the
// repository to read it.
type LLMConfig struct {
	ID        string
	UserID    string // "" means system-wide config (NULL in DB)
	Provider  LLMProviderType
	APIKey    string
	Model     string
	BaseURL   string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// NewLLMConfig validates invariants for a new LLMConfig and returns
// either a fully-formed entity or a sentinel error wrapped with context.
//
// Invariants:
//   - Provider must satisfy IsValid().
//   - Model must be non-empty.
//   - APIKey must be non-empty unless Provider == ProviderOllama (local
//     deployments don't require keys).
//
// userID == "" is treated as the system config (one row per type at the
// DB layer, enforced via UNIQUE(user_id) tolerating NULLs).
func NewLLMConfig(userID string, provider LLMProviderType, apiKey, model, baseURL string) (*LLMConfig, error) {
	if !provider.IsValid() {
		return nil, fmt.Errorf("%w: %q", ErrInvalidProvider, string(provider))
	}
	if model == "" {
		return nil, ErrEmptyModel
	}
	if apiKey == "" && provider != ProviderOllama {
		return nil, fmt.Errorf("%w: provider %s", ErrEmptyAPIKey, provider)
	}
	now := time.Now().UTC()
	return &LLMConfig{
		UserID:    userID,
		Provider:  provider,
		APIKey:    apiKey,
		Model:     model,
		BaseURL:   baseURL,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// IsSystem reports whether this config has no user owner (system-wide).
func (c *LLMConfig) IsSystem() bool { return c.UserID == "" }
