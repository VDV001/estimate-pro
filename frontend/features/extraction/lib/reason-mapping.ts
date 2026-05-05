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

export function mapExtractionReason(
  _reason: string | undefined | null,
): ExtractionReasonKey {
  throw new Error("mapExtractionReason: not implemented");
}
