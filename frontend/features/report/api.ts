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
  projectId: string,
  format: ReportFormat,
): Promise<void> {
  const headers: Record<string, string> = {};
  const token = getAccessToken();
  if (token) {
    headers["Authorization"] = `Bearer ${token}`;
  }

  const response = await fetch(
    `${API_BASE}/api/v1/projects/${projectId}/report?format=${format}`,
    { headers },
  );
  if (!response.ok) {
    const fallback = { error: { code: "UNKNOWN", message: response.statusText } };
    const body = await response.json().catch(() => fallback);
    throw new Error(body.error?.message ?? "Report download failed");
  }

  const blob = await response.blob();

  let filename = `report.${format}`;
  const disposition = response.headers.get("Content-Disposition");
  if (disposition) {
    const match = disposition.match(/filename="?([^";\n]+)"?/);
    if (match?.[1]) {
      filename = match[1];
    }
  }

  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);
  URL.revokeObjectURL(url);
}
