// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

import type { ExtractionEnvelope, ExtractionStatus } from "../api";

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

export const POLL_INTERVAL_MS = 5000;

const TERMINAL_STATUSES: readonly ExtractionStatus[] = [
  "completed",
  "failed",
  "cancelled",
];

export function isTerminalStatus(status: ExtractionStatus): boolean {
  return TERMINAL_STATUSES.includes(status);
}

// ---------------------------------------------------------------------------
// Hook (RED stub — GREEN replaces with real TanStack Query call)
// ---------------------------------------------------------------------------

export interface UseExtractionStatusResult {
  data: ExtractionEnvelope | undefined;
  isLoading: boolean;
  isError: boolean;
}

export function useExtractionStatus(
  _extractionId: string | undefined,
): UseExtractionStatusResult {
  throw new Error("useExtractionStatus: not implemented");
}
