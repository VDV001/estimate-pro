import { describe, it, expect, beforeEach } from "vitest";
import { server } from "@/__tests__/mocks/server";
import { http, HttpResponse } from "msw";
import {
  apiClient,
  getAccessToken,
  setTokens,
  clearTokens,
  ApiError,
} from "../api-client";

const ACCESS_KEY = "ep_access_token";
const REFRESH_KEY = "ep_refresh_token";

beforeEach(() => {
  localStorage.clear();
});

describe("token helpers", () => {
  it("getAccessToken returns null when no token", () => {
    expect(getAccessToken()).toBeNull();
  });

  it("setTokens stores both tokens", () => {
    setTokens("access-123", "refresh-456");
    expect(localStorage.getItem(ACCESS_KEY)).toBe("access-123");
    expect(localStorage.getItem(REFRESH_KEY)).toBe("refresh-456");
  });

  it("clearTokens removes both tokens", () => {
    setTokens("a", "r");
    clearTokens();
    expect(localStorage.getItem(ACCESS_KEY)).toBeNull();
    expect(localStorage.getItem(REFRESH_KEY)).toBeNull();
  });

  it("getAccessToken returns stored token", () => {
    setTokens("my-token", "ref");
    expect(getAccessToken()).toBe("my-token");
  });
});

describe("apiClient", () => {
  it("adds Authorization header when token exists", async () => {
    setTokens("bearer-token", "ref");

    server.use(
      http.get("http://localhost:8080/api/v1/test-auth", ({ request }) => {
        const auth = request.headers.get("Authorization");
        return HttpResponse.json({ auth });
      }),
    );

    const data = await apiClient<{ auth: string }>("/api/v1/test-auth");
    expect(data.auth).toBe("Bearer bearer-token");
  });

  it("parses JSON response", async () => {
    setTokens("t", "r");
    const data = await apiClient<{ count: number }>("/api/v1/notifications/unread-count");
    expect(data.count).toBe(3);
  });

  it("returns undefined for 204 responses", async () => {
    setTokens("t", "r");
    const result = await apiClient("/api/v1/notifications/read-all", {
      method: "PATCH",
    });
    expect(result).toBeUndefined();
  });

  it("throws ApiError on 4xx responses", async () => {
    setTokens("t", "r");

    server.use(
      http.get("http://localhost:8080/api/v1/test-error", () =>
        HttpResponse.json(
          { error: { code: "NOT_FOUND", message: "Not found" } },
          { status: 404 },
        ),
      ),
    );

    await expect(apiClient("/api/v1/test-error")).rejects.toThrow(ApiError);

    try {
      await apiClient("/api/v1/test-error");
    } catch (e) {
      const err = e as ApiError;
      expect(err.status).toBe(404);
      expect(err.code).toBe("NOT_FOUND");
    }
  });
});
