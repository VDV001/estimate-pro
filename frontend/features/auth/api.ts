// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

import { apiClient, setTokens, clearTokens, getAccessToken } from "@/lib/api-client";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export interface LoginRequest {
  email: string;
  password: string;
}

export interface RegisterRequest {
  email: string;
  password: string;
  name: string;
}

export interface User {
  id: string;
  email: string;
  name: string;
  avatar_url?: string;
  preferred_locale: string;
  telegram_chat_id?: string;
  notification_email?: string;
}

export interface AuthResponse {
  user: User;
  access_token: string;
  refresh_token: string;
}

export interface TokenResponse {
  access_token: string;
  refresh_token: string;
}

// ---------------------------------------------------------------------------
// API functions
// ---------------------------------------------------------------------------

export async function login(data: LoginRequest): Promise<AuthResponse> {
  const res = await apiClient<AuthResponse>("/api/v1/auth/login", {
    method: "POST",
    body: data,
  });
  setTokens(res.access_token, res.refresh_token);
  return res;
}

export async function register(data: RegisterRequest): Promise<AuthResponse> {
  const res = await apiClient<AuthResponse>("/api/v1/auth/register", {
    method: "POST",
    body: data,
  });
  setTokens(res.access_token, res.refresh_token);
  return res;
}

export async function refreshTokens(
  refreshToken: string
): Promise<TokenResponse> {
  const res = await apiClient<TokenResponse>("/api/v1/auth/refresh", {
    method: "POST",
    body: { refresh_token: refreshToken },
  });
  setTokens(res.access_token, res.refresh_token);
  return res;
}

export async function getCurrentUser(): Promise<User> {
  return apiClient<User>("/api/v1/auth/me");
}

export async function updateProfile(data: { name?: string; telegram_chat_id?: string; notification_email?: string }): Promise<User> {
  return apiClient<User>("/api/v1/auth/profile", {
    method: "PATCH",
    body: data,
  });
}

export async function uploadAvatar(file: File): Promise<User> {
  const formData = new FormData();
  formData.append("avatar", file);

  const headers: Record<string, string> = {};
  const token = getAccessToken();
  if (token) {
    headers["Authorization"] = `Bearer ${token}`;
  }

  const API_BASE = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";
  const response = await fetch(`${API_BASE}/api/v1/auth/avatar`, {
    method: "POST",
    headers,
    body: formData,
  });

  if (!response.ok) {
    throw new Error("Avatar upload failed");
  }

  return response.json() as Promise<User>;
}

export async function resetPassword(token: string, newPassword: string): Promise<void> {
  await apiClient<{ message: string }>("/api/v1/auth/reset-password", {
    method: "POST",
    body: { token, new_password: newPassword },
  });
}

export async function forgotPassword(email: string): Promise<void> {
  await apiClient<{ message: string }>("/api/v1/auth/forgot-password", {
    method: "POST",
    body: { email },
  });
}

export function logout(): void {
  clearTokens();
}
