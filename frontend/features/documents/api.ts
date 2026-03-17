import { apiClient, getAccessToken } from "@/lib/api-client";

const API_BASE = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export interface Document {
  id: string;
  project_id: string;
  title: string;
  uploaded_by: string;
  created_at: string;
}

export interface DocumentVersion {
  id: string;
  document_id: string;
  version_number: number;
  file_key: string;
  file_type: string;
  file_size: number;
  parsed_status: string;
  confidence_score: number;
  uploaded_by: string;
  uploaded_at: string;
}

export interface DocumentWithVersion {
  document: Document;
  version: DocumentVersion;
}

// ---------------------------------------------------------------------------
// API functions
// ---------------------------------------------------------------------------

export async function listDocuments(projectId: string): Promise<Document[]> {
  return apiClient<Document[]>(`/api/v1/projects/${projectId}/documents`);
}

export async function getDocument(
  projectId: string,
  docId: string
): Promise<DocumentWithVersion> {
  return apiClient<DocumentWithVersion>(
    `/api/v1/projects/${projectId}/documents/${docId}`
  );
}

/**
 * Upload a document via multipart/form-data.
 * We use raw fetch (not apiClient) because apiClient sets Content-Type: application/json.
 */
export async function uploadDocument(
  projectId: string,
  file: File,
  title?: string
): Promise<DocumentWithVersion> {
  const formData = new FormData();
  formData.append("file", file);
  if (title) {
    formData.append("title", title);
  }

  const headers: Record<string, string> = {};
  const token = getAccessToken();
  if (token) {
    headers["Authorization"] = `Bearer ${token}`;
  }

  const response = await fetch(
    `${API_BASE}/api/v1/projects/${projectId}/documents`,
    {
      method: "POST",
      headers,
      body: formData,
    }
  );

  if (!response.ok) {
    const error = await response.json().catch(() => ({
      error: { code: "UNKNOWN", message: response.statusText },
    }));
    throw new Error(error.error?.message ?? "Upload failed");
  }

  return response.json() as Promise<DocumentWithVersion>;
}

/**
 * Download a document file — triggers a browser download.
 */
export async function downloadDocument(
  projectId: string,
  docId: string
): Promise<void> {
  const headers: Record<string, string> = {};
  const token = getAccessToken();
  if (token) {
    headers["Authorization"] = `Bearer ${token}`;
  }

  const response = await fetch(
    `${API_BASE}/api/v1/projects/${projectId}/documents/${docId}/download`,
    { headers }
  );

  if (!response.ok) {
    throw new Error("Download failed");
  }

  const blob = await response.blob();

  // Extract filename from Content-Disposition header if available
  const disposition = response.headers.get("Content-Disposition");
  let filename = "document";
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

export async function deleteDocument(
  projectId: string,
  docId: string
): Promise<void> {
  return apiClient<void>(
    `/api/v1/projects/${projectId}/documents/${docId}`,
    { method: "DELETE" }
  );
}
