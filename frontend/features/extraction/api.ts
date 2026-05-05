// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

import { apiClient } from "@/lib/api-client";

// ---------------------------------------------------------------------------
// Types — mirrored from backend extractor handler DTOs
// ---------------------------------------------------------------------------

export type ExtractionStatus =
  | "pending"
  | "processing"
  | "completed"
  | "failed"
  | "cancelled";

export interface ExtractedTask {
  name: string;
  estimate_hint?: string;
}

export interface Extraction {
  id: string;
  document_id: string;
  document_version_id: string;
  status: ExtractionStatus;
  tasks: ExtractedTask[];
  failure_reason?: string;
  created_at: string;
  updated_at: string;
  started_at?: string;
  completed_at?: string;
}

export interface ExtractionEvent {
  id: string;
  extraction_id: string;
  from_status: string;
  to_status: string;
  error_message?: string;
  actor: string;
  created_at: string;
}

export interface ExtractionEnvelope {
  extraction: Extraction;
  events: ExtractionEvent[];
}

// ---------------------------------------------------------------------------
// API functions
// ---------------------------------------------------------------------------

export async function requestExtraction(
  projectId: string,
  docId: string,
  versionId: string,
  fileSize: number,
): Promise<Extraction> {
  return apiClient<Extraction>(
    `/api/v1/projects/${projectId}/documents/${docId}/versions/${versionId}/extractions`,
    { method: "POST", body: { file_size: fileSize } },
  );
}

export async function getExtraction(
  extractionId: string,
): Promise<ExtractionEnvelope> {
  return apiClient<ExtractionEnvelope>(
    `/api/v1/extractions/${extractionId}`,
  );
}

export async function cancelExtraction(extractionId: string): Promise<void> {
  return apiClient<void>(
    `/api/v1/extractions/${extractionId}/cancel`,
    { method: "POST" },
  );
}

export async function retryExtraction(extractionId: string): Promise<void> {
  return apiClient<void>(
    `/api/v1/extractions/${extractionId}/retry`,
    { method: "POST" },
  );
}

export async function listExtractions(
  projectId: string,
): Promise<Extraction[]> {
  return apiClient<Extraction[]>(
    `/api/v1/projects/${projectId}/extractions`,
  );
}
