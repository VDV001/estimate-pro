// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

import { apiClient } from "@/lib/api-client";

export interface Notification {
  id: string;
  user_id: string;
  event_type: string;
  title: string;
  message: string;
  project_id?: string;
  read: boolean;
  created_at: string;
}

export interface NotificationListResponse {
  notifications: Notification[] | null;
  meta: { total: number; page: number; limit: number };
}

export interface NotificationPreference {
  channel: "in_app" | "email" | "telegram";
  enabled: boolean;
}

export async function listNotifications(
  page = 1,
  limit = 20
): Promise<NotificationListResponse> {
  return apiClient<NotificationListResponse>(
    `/api/v1/notifications?page=${page}&limit=${limit}`
  );
}

export async function getUnreadCount(): Promise<{ count: number }> {
  return apiClient<{ count: number }>("/api/v1/notifications/unread-count");
}

export async function markRead(id: string): Promise<void> {
  return apiClient<void>(`/api/v1/notifications/${id}/read`, {
    method: "PATCH",
  });
}

export async function markAllRead(): Promise<void> {
  return apiClient<void>("/api/v1/notifications/read-all", {
    method: "PATCH",
  });
}

export async function getPreferences(): Promise<{
  preferences: NotificationPreference[];
}> {
  return apiClient<{ preferences: NotificationPreference[] }>(
    "/api/v1/notifications/preferences"
  );
}

export async function setPreference(
  channel: string,
  enabled: boolean
): Promise<void> {
  return apiClient<void>("/api/v1/notifications/preferences", {
    method: "PUT",
    body: { channel, enabled },
  });
}
