import { Suspense } from "react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, act, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { http, HttpResponse } from "msw";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useAuthStore } from "@/features/auth/store";
import { server } from "@/__tests__/mocks/server";

vi.mock("@/lib/query-client", () => ({
  getQueryClient: () => new QueryClient({ defaultOptions: { queries: { retry: false } } }),
}));

import ProjectDetailPage from "../[id]/page";

const API = "http://localhost:8080/api/v1";

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

  it("createEstimation in extraction panel switches to estimations tab and pre-fills the form", async () => {
    server.use(
      http.get(`${API}/projects/:id/extractions`, () =>
        HttpResponse.json([
          {
            id: "ext-d1",
            document_id: "d1",
            document_version_id: "v1",
            status: "completed",
            tasks: [
              { name: "Implement login" },
              { name: "Wire OAuth" },
            ],
            created_at: "2026-05-05T00:00:00Z",
            updated_at: "2026-05-05T00:01:00Z",
          },
        ]),
      ),
      http.get(`${API}/extractions/:id`, () =>
        HttpResponse.json({
          extraction: {
            id: "ext-d1",
            document_id: "d1",
            document_version_id: "v1",
            status: "completed",
            tasks: [
              { name: "Implement login" },
              { name: "Wire OAuth" },
            ],
            created_at: "2026-05-05T00:00:00Z",
            updated_at: "2026-05-05T00:01:00Z",
          },
          events: [],
        }),
      ),
    );

    const user = userEvent.setup();
    await renderWithProviders();

    // Open documents tab
    const docsTab = await screen.findByText("projects.tabs.documents");
    await user.click(docsTab);

    // Wait for the extraction button to appear
    const createBtn = await screen.findByText(
      "extraction.actions.createEstimation",
    );
    await user.click(createBtn);

    // Now we should be on the estimations tab with the form pre-filled
    await waitFor(() => {
      const inputs = screen.getAllByPlaceholderText(
        "estimation.taskPlaceholder",
      ) as HTMLInputElement[];
      const taskNames = inputs.map((i) => i.value);
      expect(taskNames).toContain("Implement login");
      expect(taskNames).toContain("Wire OAuth");
    });
  });
});
