// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

const API_BASE = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

type RequestOptions = Omit<RequestInit, "body"> & {
  body?: unknown;
};

// ---------------------------------------------------------------------------
// Token helpers — used by auth store and api-client internals
// ---------------------------------------------------------------------------

const ACCESS_TOKEN_KEY = "ep_access_token";
const REFRESH_TOKEN_KEY = "ep_refresh_token";

export function getAccessToken(): string | null {
  if (typeof window === "undefined") return null;
  return localStorage.getItem(ACCESS_TOKEN_KEY);
}

export function getRefreshToken(): string | null {
  if (typeof window === "undefined") return null;
  return localStorage.getItem(REFRESH_TOKEN_KEY);
}

export function setTokens(accessToken: string, refreshToken: string): void {
  localStorage.setItem(ACCESS_TOKEN_KEY, accessToken);
  localStorage.setItem(REFRESH_TOKEN_KEY, refreshToken);
}

export function clearTokens(): void {
  localStorage.removeItem(ACCESS_TOKEN_KEY);
  localStorage.removeItem(REFRESH_TOKEN_KEY);
}

// ---------------------------------------------------------------------------
// Silent token refresh (singleton to avoid parallel refreshes)
// ---------------------------------------------------------------------------

let isRefreshing = false;
let refreshPromise: Promise<boolean> | null = null;

async function tryRefreshToken(): Promise<boolean> {
  const refreshToken = getRefreshToken();
  if (!refreshToken) return false;

  try {
    const res = await fetch(`${API_BASE}/api/v1/auth/refresh`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ refresh_token: refreshToken }),
    });

    if (!res.ok) return false;

    const data = await res.json();
    setTokens(data.access_token, data.refresh_token);
    return true;
  } catch {
    return false;
  }
}

async function refreshOnce(): Promise<boolean> {
  if (isRefreshing && refreshPromise) return refreshPromise;
  isRefreshing = true;
  refreshPromise = tryRefreshToken().finally(() => {
    isRefreshing = false;
    refreshPromise = null;
  });
  return refreshPromise;
}

// ---------------------------------------------------------------------------
// Generic API client
// ---------------------------------------------------------------------------

export async function apiClient<T>(
  path: string,
  options: RequestOptions = {}
): Promise<T> {
  const doFetch = async () => {
    const { body, headers: customHeaders, ...rest } = options;

    const headers: Record<string, string> = {
      "Content-Type": "application/json",
      ...Object.fromEntries(
        Object.entries(customHeaders ?? {}).filter(
          (entry): entry is [string, string] => typeof entry[1] === "string"
        )
      ),
    };

    const token = getAccessToken();
    if (token) {
      headers["Authorization"] = `Bearer ${token}`;
    }

    return fetch(`${API_BASE}${path}`, {
      ...rest,
      headers,
      body: body ? JSON.stringify(body) : undefined,
    });
  };

  let response = await doFetch();

  // Auto-refresh on 401
  if (response.status === 401 && typeof window !== "undefined") {
    const refreshed = await refreshOnce();
    if (refreshed) {
      response = await doFetch();
    } else {
      // Refresh failed — clear tokens, redirect to login
      clearTokens();
      window.location.href = "/login";
      throw new ApiError(401, "UNAUTHORIZED", "Session expired");
    }
  }

  if (!response.ok) {
    const error = await response.json().catch(() => ({
      error: { code: "UNKNOWN", message: response.statusText },
    }));
    throw new ApiError(response.status, error.error.code, error.error.message);
  }

  if (response.status === 204) {
    return undefined as T;
  }

  return response.json();
}

export class ApiError extends Error {
  constructor(
    public status: number,
    public code: string,
    message: string
  ) {
    super(message);
    this.name = "ApiError";
  }
}
