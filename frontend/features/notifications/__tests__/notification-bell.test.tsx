// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { NotificationBell } from "../components/notification-bell";

function makeWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  const wrapper = ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
  return wrapper;
}

describe("NotificationBell", () => {
  it("shows bell icon button", () => {
    render(<NotificationBell />, { wrapper: makeWrapper() });

    // The bell trigger is a <button> wrapping the Bell icon
    const buttons = screen.getAllByRole("button");
    expect(buttons.length).toBeGreaterThan(0);
  });

  it("renders bell trigger that can be hovered to show notification panel", async () => {
    const user = userEvent.setup();

    render(<NotificationBell />, { wrapper: makeWrapper() });

    // Find the bell button (wraps the SVG icon, has no text)
    const bellButton = screen.getAllByRole("button")[0];
    expect(bellButton).toBeDefined();

    // Hover to open the HoverCard
    await user.hover(bellButton);

    // After hover, HoverCardContent renders; title should be present
    expect(await screen.findByText("notifications.title")).toBeDefined();
  });

  it("renders unread count in notification panel header", async () => {
    const user = userEvent.setup();

    render(<NotificationBell />, { wrapper: makeWrapper() });

    const bellButton = screen.getAllByRole("button")[0];
    await user.hover(bellButton);

    // MSW returns count: 3 for unread-count endpoint
    // The count is displayed as a plain number next to the title
    expect(await screen.findByText("3")).toBeDefined();
  });

  it("renders notification message in the list", async () => {
    const user = userEvent.setup();

    render(<NotificationBell />, { wrapper: makeWrapper() });

    const bellButton = screen.getAllByRole("button")[0];
    await user.hover(bellButton);

    // formatMessage translates via eventMessage key with {name} substitution
    // Mock useTranslations returns the key with {name} replaced
    expect(await screen.findByText("notifications.eventMessage.member_added")).toBeDefined();
  });

  it("renders view all link in footer", async () => {
    const user = userEvent.setup();

    render(<NotificationBell />, { wrapper: makeWrapper() });

    const bellButton = screen.getAllByRole("button")[0];
    await user.hover(bellButton);

    expect(await screen.findByText("notifications.viewAll")).toBeDefined();
  });
});
