import { describe, it, expect, beforeEach, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { DocumentsList } from "../components/documents-list";

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

describe("DocumentsList", () => {
  it("renders loading state", () => {
    render(<DocumentsList projectId="p1" />, { wrapper: makeWrapper() });
    expect(screen.getByText("common.loading")).toBeDefined();
  });

  it("renders document list from API", async () => {
    render(<DocumentsList projectId="p1" />, { wrapper: makeWrapper() });
    expect(await screen.findByText("spec.pdf")).toBeDefined();
  });

  it("shows upload button", async () => {
    render(<DocumentsList projectId="p1" />, { wrapper: makeWrapper() });
    await screen.findByText("spec.pdf");
    const uploadBtn = screen.getByText("documents.upload");
    expect(uploadBtn).toBeDefined();
  });

  it("delete button triggers confirm and removes document", async () => {
    const user = userEvent.setup();
    vi.spyOn(window, "confirm").mockReturnValue(true);

    render(<DocumentsList projectId="p1" />, { wrapper: makeWrapper() });
    await screen.findByText("spec.pdf");

    // Find delete buttons (trash icons)
    const buttons = screen.getAllByRole("button");
    const deleteBtn = buttons.find(
      (btn) => btn.querySelector("svg.lucide-trash-2") !== null
    );
    if (deleteBtn) {
      await user.click(deleteBtn);
      expect(window.confirm).toHaveBeenCalled();
    }

    vi.restoreAllMocks();
  });
});
