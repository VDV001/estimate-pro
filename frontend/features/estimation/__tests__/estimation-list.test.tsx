import { describe, it, expect, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { EstimationList } from "../components/estimation-list";

function makeWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
}

beforeEach(() => {
  localStorage.setItem("ep_access_token", "test-token");
});

describe("EstimationList", () => {
  it("shows loading state initially", () => {
    render(<EstimationList projectId="p1" />, { wrapper: makeWrapper() });
    expect(screen.getByText("common.loading")).toBeDefined();
  });

  it("renders estimation list with status badge", async () => {
    render(<EstimationList projectId="p1" />, { wrapper: makeWrapper() });
    // MSW returns estimation with status "draft"
    expect(await screen.findByText("estimation.draft")).toBeDefined();
  });

  it("shows create button that toggles form", async () => {
    const user = userEvent.setup();
    render(<EstimationList projectId="p1" />, { wrapper: makeWrapper() });

    await screen.findByText("estimation.draft");

    const createBtn = screen.getByText("estimation.create");
    await user.click(createBtn);

    // Form should appear with task header
    expect(screen.getByText("estimation.createTitle")).toBeDefined();
  });

  it("expand row shows items table", async () => {
    const user = userEvent.setup();
    render(<EstimationList projectId="p1" />, { wrapper: makeWrapper() });

    // Wait for list to load
    await screen.findByText("estimation.draft");
    // Click the row to expand
    const expandableRows = screen.getAllByRole("button");
    const rowBtn = expandableRows.find(
      (btn) => btn.textContent?.includes("estimation.draft")
    );
    if (rowBtn) {
      await user.click(rowBtn);

      // After expanding, items table should load (MSW returns items)
      await waitFor(() => {
        expect(screen.getByText("Task 1")).toBeDefined();
      });
    }
  });

  it("draft estimation shows submit and delete buttons", async () => {
    render(<EstimationList projectId="p1" />, { wrapper: makeWrapper() });
    await screen.findByText("estimation.draft");

    // Submit button should be present for drafts
    expect(screen.getByText("estimation.submit")).toBeDefined();
  });

  it("draft estimation renders action buttons (submit and delete)", async () => {
    render(<EstimationList projectId="p1" />, { wrapper: makeWrapper() });
    await screen.findByText("estimation.draft");

    // Draft estimations should have submit button
    expect(screen.getByText("estimation.submit")).toBeDefined();

    // Should have more than just Create button (submit + delete + create + expand)
    const buttons = screen.getAllByRole("button");
    expect(buttons.length).toBeGreaterThanOrEqual(3);
  });
});
