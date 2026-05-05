// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

import { getAccessToken } from "@/lib/api-client";

const API_BASE = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

export type ReportFormat = "md" | "pdf" | "docx";

export const REPORT_FORMATS: readonly ReportFormat[] = ["md", "pdf", "docx"];

/**
 * downloadReport fetches the rendered report bytes from the backend
 * and triggers a browser download. The endpoint streams binary, so
 * we use raw fetch + blob() rather than apiClient (which expects
 * JSON).
 */
export async function downloadReport(
  _projectId: string,
  _format: ReportFormat,
): Promise<void> {
  throw new Error("downloadReport: not implemented");
}
