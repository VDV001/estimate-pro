import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useAuthStore } from "@/features/auth/store";

vi.mock("@/lib/query-client", () => ({
  getQueryClient: () => new QueryClient({ defaultOptions: { queries: { retry: false } } }),
}));

import SettingsPage from "../settings/page";

function renderWithProviders() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <SettingsPage />
    </QueryClientProvider>,
  );
}

beforeEach(() => {
  localStorage.setItem("ep_access_token", "test-token");
  useAuthStore.setState({
    user: { id: "u1", name: "Test User", email: "test@test.com", preferred_locale: "ru" },
    isAuthenticated: true,
    isLoading: false,
  });
});

describe("SettingsPage", () => {
  it("renders profile section", () => {
    renderWithProviders();
    expect(screen.getByText("settings.profile")).toBeInTheDocument();
  });

  it("renders appearance section", () => {
    renderWithProviders();
    expect(screen.getByText("settings.appearance")).toBeInTheDocument();
  });

  it("renders user name in input", () => {
    renderWithProviders();
    const input = screen.getByPlaceholderText("settings.namePlaceholder");
    expect(input).toHaveValue("Test User");
  });
});
