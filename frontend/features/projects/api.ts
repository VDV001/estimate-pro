// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

import { apiClient } from "@/lib/api-client";

export interface Project {
  id: string;
  workspace_id: string;
  name: string;
  description: string;
  status: string;
  created_by: string;
  created_at: string;
  updated_at: string;
}

export interface Workspace {
  id: string;
  name: string;
  owner_id: string;
  created_at: string;
}

export interface ProjectListResponse {
  projects: Project[] | null;
  meta: { total: number; page: number; limit: number };
}

export async function listWorkspaces() {
  return apiClient<Workspace[]>("/api/v1/workspaces");
}

export async function createProject(data: {
  workspace_id: string;
  name: string;
  description: string;
}) {
  return apiClient<Project>("/api/v1/projects", {
    method: "POST",
    body: data,
  });
}

export async function listProjects(
  workspaceId?: string,
  page = 1,
  limit = 20
) {
  const params = new URLSearchParams({ page: String(page), limit: String(limit) });
  if (workspaceId) {
    params.set("workspace_id", workspaceId);
  }
  return apiClient<ProjectListResponse>(
    `/api/v1/projects?${params.toString()}`
  );
}

export async function getProject(id: string) {
  return apiClient<Project>(`/api/v1/projects/${id}`);
}

export interface MemberWithUser {
  project_id: string;
  user_id: string;
  role: string;
  user_name: string;
  user_email: string;
}

export async function listMembers(projectId: string) {
  return apiClient<MemberWithUser[]>(`/api/v1/projects/${projectId}/members`);
}

export async function addMember(
  projectId: string,
  data: { email: string; role: string }
) {
  return apiClient<{ status: string }>(`/api/v1/projects/${projectId}/members`, {
    method: "POST",
    body: data,
  });
}

export async function removeMember(projectId: string, userId: string) {
  return apiClient<void>(`/api/v1/projects/${projectId}/members/${userId}`, {
    method: "DELETE",
  });
}

export async function updateProject(
  id: string,
  data: { name?: string; description?: string }
) {
  return apiClient<Project>(`/api/v1/projects/${id}`, {
    method: "PATCH",
    body: data,
  });
}

export interface UserSearchResult {
  id: string;
  email: string;
  name: string;
  avatar_url?: string;
}

export async function searchUsers(query: string) {
  return apiClient<UserSearchResult[]>(`/api/v1/auth/users/search?q=${encodeURIComponent(query)}`);
}

export async function listColleagues() {
  return apiClient<UserSearchResult[]>("/api/v1/auth/users/colleagues");
}
export async function updateWorkspace(
  id: string,
  data: { name: string }
) {
  return apiClient<Workspace>(`/api/v1/workspaces/${id}`, {
    method: "PATCH",
    body: data,
  });
}

export async function listRecentlyAdded() {
  return apiClient<UserSearchResult[]>("/api/v1/auth/users/recent");
}

export async function archiveProject(id: string) {
  return apiClient<Project>(`/api/v1/projects/${id}`, {
    method: "DELETE",
  });
}

export async function restoreProject(id: string) {
  return apiClient<Project>(`/api/v1/projects/${id}/restore`, {
    method: "POST",
  });
}
