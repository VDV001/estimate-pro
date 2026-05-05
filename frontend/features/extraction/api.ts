// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

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
// API functions — STUBS (RED phase, GREEN replaces with apiClient calls)
// ---------------------------------------------------------------------------

export async function requestExtraction(
  _projectId: string,
  _docId: string,
  _versionId: string,
  _fileSize: number,
): Promise<Extraction> {
  throw new Error("requestExtraction: not implemented");
}

export async function getExtraction(
  _extractionId: string,
): Promise<ExtractionEnvelope> {
  throw new Error("getExtraction: not implemented");
}

export async function cancelExtraction(_extractionId: string): Promise<void> {
  throw new Error("cancelExtraction: not implemented");
}

export async function retryExtraction(_extractionId: string): Promise<void> {
  throw new Error("retryExtraction: not implemented");
}

export async function listExtractions(
  _projectId: string,
): Promise<Extraction[]> {
  throw new Error("listExtractions: not implemented");
}
