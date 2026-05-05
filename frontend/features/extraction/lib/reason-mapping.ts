// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

// ---------------------------------------------------------------------------
// Reason → i18n key mapping (RED stub — GREEN fills in the table)
//
// Mirrors backend/internal/modules/bot/handler/messages/messages.go
// ExtractionFailed — every known sentinel from extractor/worker
// reasonForPipelineError gets a dedicated i18n key. Unknown reasons
// fall through to the generic key so a future sentinel addition
// surfaces a translated message rather than the raw English value.
// ---------------------------------------------------------------------------

export type ExtractionReasonKey =
  | "encrypted"
  | "llmService"
  | "promptInjection"
  | "schemaInvalid"
  | "pipelineError"
  | "cancelled"
  | "unknown";

const REASON_TO_KEY: Record<string, ExtractionReasonKey> = {
  "encrypted file (password protected)": "encrypted",
  "LLM service error": "llmService",
  "prompt injection detected": "promptInjection",
  "LLM response failed schema validation": "schemaInvalid",
  "pipeline error": "pipelineError",
  cancelled: "cancelled",
};

export function mapExtractionReason(
  reason: string | undefined | null,
): ExtractionReasonKey {
  if (!reason) return "unknown";
  return REASON_TO_KEY[reason] ?? "unknown";
}
