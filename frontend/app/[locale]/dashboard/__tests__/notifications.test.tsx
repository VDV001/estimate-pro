import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

vi.mock("@/lib/query-client", () => ({
  getQueryClient: () => new QueryClient({ defaultOptions: { queries: { retry: false } } }),
}));

import NotificationsPage from "../notifications/page";

function renderWithProviders() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <NotificationsPage />
    </QueryClientProvider>,
  );
}

beforeEach(() => {
  localStorage.setItem("ep_access_token", "test-token");
});

describe("NotificationsPage", () => {
  it("renders page title", () => {
    renderWithProviders();
    expect(screen.getByText("notifications.title")).toBeInTheDocument();
  });

  it("renders mark all read button when there are unread", async () => {
    renderWithProviders();
    expect(await screen.findByText("notifications.markAllRead")).toBeInTheDocument();
  });
});
