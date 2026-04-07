// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package domain

import (
	"testing"
	"time"
)

func TestIntentType_IsValid(t *testing.T) {
	tests := []struct {
		name   string
		intent IntentType
		want   bool
	}{
		{"create_project", IntentCreateProject, true},
		{"update_project", IntentUpdateProject, true},
		{"list_projects", IntentListProjects, true},
		{"get_project_status", IntentGetProjectStatus, true},
		{"add_member", IntentAddMember, true},
		{"remove_member", IntentRemoveMember, true},
		{"list_members", IntentListMembers, true},
		{"request_estimation", IntentRequestEstimation, true},
		{"submit_estimation", IntentSubmitEstimation, true},
		{"get_aggregated", IntentGetAggregated, true},
		{"upload_document", IntentUploadDocument, true},
		{"help", IntentHelp, true},
		{"unknown", IntentUnknown, true},
		{"invalid", IntentType("invalid"), false},
		{"empty", IntentType(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.intent.IsValid(); got != tt.want {
				t.Errorf("IntentType(%q).IsValid() = %v, want %v", tt.intent, got, tt.want)
			}
		})
	}
}

func TestLLMProviderType_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		provider LLMProviderType
		want     bool
	}{
		{"claude", ProviderClaude, true},
		{"openai", ProviderOpenAI, true},
		{"grok", ProviderGrok, true},
		{"ollama", ProviderOllama, true},
		{"invalid", LLMProviderType("invalid"), false},
		{"empty", LLMProviderType(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.provider.IsValid(); got != tt.want {
				t.Errorf("LLMProviderType(%q).IsValid() = %v, want %v", tt.provider, got, tt.want)
			}
		})
	}
}

func TestBotSession_IsExpired(t *testing.T) {
	tests := []struct {
		name      string
		expiresAt time.Time
		want      bool
	}{
		{
			name:      "expired session (past)",
			expiresAt: time.Now().Add(-1 * time.Hour),
			want:      true,
		},
		{
			name:      "not expired session (future)",
			expiresAt: time.Now().Add(1 * time.Hour),
			want:      false,
		},
		{
			name:      "barely not expired (1ms ahead)",
			expiresAt: time.Now().Add(1 * time.Millisecond),
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &BotSession{ExpiresAt: tt.expiresAt}
			if got := s.IsExpired(); got != tt.want {
				t.Errorf("BotSession.IsExpired() = %v, want %v (expiresAt: %v, now: %v)",
					got, tt.want, tt.expiresAt, time.Now())
			}
		})
	}
}
