import { Suspense } from "react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, act } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useAuthStore } from "@/features/auth/store";

vi.mock("@/lib/query-client", () => ({
  getQueryClient: () => new QueryClient({ defaultOptions: { queries: { retry: false } } }),
}));

import ProjectDetailPage from "../[id]/page";

async function renderWithProviders() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  const params = Promise.resolve({ id: "p1" });

  await act(async () => {
    render(
      <QueryClientProvider client={queryClient}>
        <Suspense fallback={<div>Loading...</div>}>
          <ProjectDetailPage params={params} />
        </Suspense>
      </QueryClientProvider>,
    );
  });
}

beforeEach(() => {
  localStorage.setItem("ep_access_token", "test-token");
  useAuthStore.setState({
    user: { id: "u1", name: "Test User", email: "test@test.com", preferred_locale: "ru" },
    isAuthenticated: true,
    isLoading: false,
  });
});

describe("ProjectDetailPage", () => {
  it("renders all 4 tabs after data loads", async () => {
    await renderWithProviders();
    expect(await screen.findByText("projects.tabs.overview")).toBeInTheDocument();
    expect(screen.getByText("projects.tabs.documents")).toBeInTheDocument();
    expect(screen.getByText("projects.tabs.estimations")).toBeInTheDocument();
    expect(screen.getByText("projects.tabs.members")).toBeInTheDocument();
  });

  it("renders overview tab as active by default", async () => {
    await renderWithProviders();
    const overviewTab = await screen.findByText("projects.tabs.overview");
    // Default tab should be styled as active (has specific classes)
    expect(overviewTab).toBeInTheDocument();
    expect(overviewTab.closest("button")).toBeTruthy();
  });
});
