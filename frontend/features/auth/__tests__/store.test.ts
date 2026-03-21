import { describe, it, expect, beforeEach, vi } from "vitest";
import { useAuthStore } from "../store";

// Mock query-client to prevent import issues
vi.mock("@/lib/query-client", () => ({
  getQueryClient: () => ({ clear: vi.fn() }),
}));

beforeEach(() => {
  localStorage.clear();
  useAuthStore.setState({
    user: null,
    isLoading: true,
    isAuthenticated: false,
  });
});

describe("auth store", () => {
  it("has correct initial state", () => {
    const state = useAuthStore.getState();
    expect(state.user).toBeNull();
    expect(state.isAuthenticated).toBe(false);
  });

  it("setUser updates user", () => {
    useAuthStore.getState().setUser({
      id: "u1",
      name: "Test",
      email: "test@test.com",
      preferred_locale: "ru",
    });
    const state = useAuthStore.getState();
    expect(state.user?.name).toBe("Test");
  });

  it("logoutUser clears state and tokens", () => {
    localStorage.setItem("ep_access_token", "t");
    localStorage.setItem("ep_refresh_token", "r");

    useAuthStore.setState({
      user: { id: "u1", name: "Test", email: "test@test.com", preferred_locale: "ru" },
      isAuthenticated: true,
    });

    useAuthStore.getState().logoutUser();

    const state = useAuthStore.getState();
    expect(state.user).toBeNull();
    expect(state.isAuthenticated).toBe(false);
    expect(localStorage.getItem("ep_access_token")).toBeNull();
  });

  it("initialize sets isLoading false when no token", async () => {
    await useAuthStore.getState().initialize();
    const state = useAuthStore.getState();
    expect(state.isLoading).toBe(false);
    expect(state.isAuthenticated).toBe(false);
  });

  it("initialize fetches user when token exists", async () => {
    localStorage.setItem("ep_access_token", "valid-token");

    await useAuthStore.getState().initialize();
    const state = useAuthStore.getState();
    expect(state.isLoading).toBe(false);
    expect(state.isAuthenticated).toBe(true);
    expect(state.user?.name).toBe("Test User");
  });

  it("loginUser sets user and isAuthenticated", async () => {
    await useAuthStore.getState().loginUser({
      email: "test@test.com",
      password: "password",
    });

    const state = useAuthStore.getState();
    expect(state.isAuthenticated).toBe(true);
    expect(state.user?.email).toBe("test@test.com");
  });

  it("registerUser sets user and isAuthenticated", async () => {
    await useAuthStore.getState().registerUser({
      name: "New User",
      email: "new@test.com",
      password: "password",
    });

    const state = useAuthStore.getState();
    expect(state.isAuthenticated).toBe(true);
    expect(state.user).not.toBeNull();
  });
});
