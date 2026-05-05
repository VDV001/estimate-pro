// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

import { useQuery } from "@tanstack/react-query";
import { getExtraction } from "../api";
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
// Hook
// ---------------------------------------------------------------------------

export interface UseExtractionStatusResult {
  data: ExtractionEnvelope | undefined;
  isLoading: boolean;
  isError: boolean;
}

export function useExtractionStatus(
  extractionId: string | undefined,
): UseExtractionStatusResult {
  const query = useQuery({
    queryKey: ["extraction", extractionId],
    queryFn: () => getExtraction(extractionId!),
    enabled: Boolean(extractionId),
    refetchInterval: (q) => {
      const status = q.state.data?.extraction.status;
      if (!status || isTerminalStatus(status)) return false;
      return POLL_INTERVAL_MS;
    },
  });

  return {
    data: query.data,
    isLoading: query.isLoading && query.fetchStatus !== "idle",
    isError: query.isError,
  };
}
