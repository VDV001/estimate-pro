// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

import { apiClient } from "@/lib/api-client";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export type EstimationStatus = "draft" | "submitted";

export interface Estimation {
  id: string;
  project_id: string;
  document_version_id?: string;
  submitted_by: string;
  status: EstimationStatus;
  submitted_at?: string;
  created_at: string;
}

export interface EstimationItem {
  id: string;
  estimation_id: string;
  task_name: string;
  min_hours: number;
  likely_hours: number;
  max_hours: number;
  sort_order: number;
  note?: string;
}

export interface EstimationWithItems {
  estimation: Estimation;
  items: EstimationItem[];
}

export interface AggregatedItem {
  task_name: string;
  avg_pert_hours: number;
  min_of_mins: number;
  max_of_maxes: number;
  estimator_count: number;
}

export interface AggregatedResult {
  items: AggregatedItem[] | null;
  total_hours: number;
}

export interface CreateEstimationItemInput {
  task_name: string;
  min_hours: number;
  likely_hours: number;
  max_hours: number;
  sort_order: number;
  note?: string;
}

export interface CreateEstimationInput {
  document_version_id?: string;
  items: CreateEstimationItemInput[];
}

// ---------------------------------------------------------------------------
// API functions
// ---------------------------------------------------------------------------

export async function listEstimations(
  projectId: string,
  mine?: boolean
): Promise<Estimation[]> {
  const params = mine ? "?mine=true" : "";
  return apiClient<Estimation[]>(
    `/api/v1/projects/${projectId}/estimations${params}`
  );
}

export async function getEstimation(
  projectId: string,
  estId: string
): Promise<EstimationWithItems> {
  return apiClient<EstimationWithItems>(
    `/api/v1/projects/${projectId}/estimations/${estId}`
  );
}

export async function createEstimation(
  projectId: string,
  input: CreateEstimationInput
): Promise<EstimationWithItems> {
  return apiClient<EstimationWithItems>(
    `/api/v1/projects/${projectId}/estimations`,
    {
      method: "POST",
      body: input,
    }
  );
}

export async function submitEstimation(
  projectId: string,
  estId: string
): Promise<void> {
  return apiClient<void>(
    `/api/v1/projects/${projectId}/estimations/${estId}/submit`,
    { method: "PUT" }
  );
}

export async function deleteEstimation(
  projectId: string,
  estId: string
): Promise<void> {
  return apiClient<void>(
    `/api/v1/projects/${projectId}/estimations/${estId}`,
    { method: "DELETE" }
  );
}

export async function getAggregated(
  projectId: string
): Promise<AggregatedResult> {
  return apiClient<AggregatedResult>(
    `/api/v1/projects/${projectId}/estimations/aggregated`
  );
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

export function pertHours(min: number, likely: number, max: number): number {
  return (min + 4 * likely + max) / 6;
}
