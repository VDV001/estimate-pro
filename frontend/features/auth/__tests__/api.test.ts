import { describe, it, expect, beforeEach } from "vitest";
import {
  login,
  register,
  getCurrentUser,
  refreshTokens,
  updateProfile,
  forgotPassword,
  resetPassword,
  logout,
} from "../api";
import { getAccessToken } from "@/lib/api-client";

beforeEach(() => {
  localStorage.clear();
});

describe("auth API", () => {
  it("login returns user and sets tokens", async () => {
    const res = await login({ email: "test@test.com", password: "password123" });
    expect(res.user).toHaveProperty("id", "u1");
    expect(res.access_token).toBe("test-access-token");
    expect(getAccessToken()).toBe("test-access-token");
  });

  it("register returns user and sets tokens", async () => {
    const res = await register({ email: "test@test.com", password: "password123", name: "Test" });
    expect(res.user).toHaveProperty("id", "u1");
    expect(getAccessToken()).toBe("test-access-token");
  });

  it("getCurrentUser returns user", async () => {
    localStorage.setItem("ep_access_token", "test-token");
    const user = await getCurrentUser();
    expect(user).toHaveProperty("email", "test@test.com");
  });

  it("refreshTokens sets new tokens", async () => {
    const res = await refreshTokens("old-refresh-token");
    expect(res.access_token).toBe("new-access-token");
    expect(getAccessToken()).toBe("new-access-token");
  });

  it("updateProfile sends PATCH", async () => {
    localStorage.setItem("ep_access_token", "test-token");
    const user = await updateProfile({ name: "Updated Name" });
    expect(user).toHaveProperty("name", "Updated Name");
  });

  it("forgotPassword resolves without error", async () => {
    localStorage.setItem("ep_access_token", "test-token");
    await expect(forgotPassword("test@test.com")).resolves.toBeUndefined();
  });

  it("resetPassword resolves without error", async () => {
    localStorage.setItem("ep_access_token", "test-token");
    await expect(resetPassword("valid-token", "newPassword123")).resolves.toBeUndefined();
  });

  it("logout clears tokens", () => {
    localStorage.setItem("ep_access_token", "test-token");
    localStorage.setItem("ep_refresh_token", "test-refresh");
    logout();
    expect(getAccessToken()).toBeNull();
  });
});
