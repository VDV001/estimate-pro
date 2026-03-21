// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { AuthGuard } from "../components/auth-guard";
import { useAuthStore } from "../store";

vi.mock("@/lib/query-client", () => ({
  getQueryClient: () => ({ clear: vi.fn() }),
}));

const mockPush = vi.fn();

vi.mock("@/i18n/navigation", async () => {
  const { createElement } = await import("react");
  return {
    useRouter: () => ({
      push: mockPush,
      replace: vi.fn(),
      back: vi.fn(),
      prefetch: vi.fn(),
    }),
    usePathname: () => "/dashboard",
    Link: ({ children, href }: { children: React.ReactNode; href: string }) =>
      createElement("a", { href }, children),
    redirect: vi.fn(),
  };
});

beforeEach(() => {
  mockPush.mockReset();
  // Reset store to a clean state before each test
  useAuthStore.setState({
    user: null,
    isLoading: true,
    isAuthenticated: false,
    initialize: vi.fn(),
    loginUser: vi.fn(),
    registerUser: vi.fn(),
    logoutUser: vi.fn(),
    setUser: vi.fn(),
  });
});

describe("AuthGuard", () => {
  it("renders children if authenticated", () => {
    useAuthStore.setState({
      isLoading: false,
      isAuthenticated: true,
      user: {
        id: "u1",
        name: "Test User",
        email: "test@test.com",
        preferred_locale: "en",
      },
      initialize: vi.fn(),
      loginUser: vi.fn(),
      registerUser: vi.fn(),
      logoutUser: vi.fn(),
      setUser: vi.fn(),
    });

    render(
      <AuthGuard>
        <div>Protected content</div>
      </AuthGuard>
    );

    expect(screen.getByText("Protected content")).toBeDefined();
    expect(mockPush).not.toHaveBeenCalled();
  });

  it("redirects to /login if not authenticated", () => {
    useAuthStore.setState({
      isLoading: false,
      isAuthenticated: false,
      user: null,
      initialize: vi.fn(),
      loginUser: vi.fn(),
      registerUser: vi.fn(),
      logoutUser: vi.fn(),
      setUser: vi.fn(),
    });

    render(
      <AuthGuard>
        <div>Protected content</div>
      </AuthGuard>
    );

    expect(mockPush).toHaveBeenCalledWith("/login");
  });

  it("shows children during initialization (isLoading=true) without redirect", () => {
    useAuthStore.setState({
      isLoading: true,
      isAuthenticated: false,
      user: null,
      initialize: vi.fn(),
      loginUser: vi.fn(),
      registerUser: vi.fn(),
      logoutUser: vi.fn(),
      setUser: vi.fn(),
    });

    render(
      <AuthGuard>
        <div>Loading content</div>
      </AuthGuard>
    );

    // Children always render (no flash guard)
    expect(screen.getByText("Loading content")).toBeDefined();
    // No redirect while still loading
    expect(mockPush).not.toHaveBeenCalled();
  });
});
