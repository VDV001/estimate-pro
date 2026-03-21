// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { http, HttpResponse } from "msw";
import { server } from "../../../__tests__/mocks/server";
import { MembersList } from "../components/members-list";

const API = "http://localhost:8080/api/v1";

function makeWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  const wrapper = ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
  return wrapper;
}

// The component expects flat MemberWithUser shape: user_name, user_email
const flatMembers = [
  { project_id: "p1", user_id: "u1", role: "admin", user_name: "Admin User", user_email: "admin@test.com" },
  { project_id: "p1", user_id: "u2", role: "developer", user_name: "Dev User", user_email: "dev@test.com" },
];

describe("MembersList", () => {
  it("renders member list with names", async () => {
    server.use(
      http.get(`${API}/projects/:id/members`, () =>
        HttpResponse.json(flatMembers)
      )
    );

    render(<MembersList projectId="p1" />, { wrapper: makeWrapper() });

    expect(await screen.findByText("Admin User")).toBeDefined();
    expect(await screen.findByText("Dev User")).toBeDefined();
  });

  it("renders member emails", async () => {
    server.use(
      http.get(`${API}/projects/:id/members`, () =>
        HttpResponse.json(flatMembers)
      )
    );

    render(<MembersList projectId="p1" />, { wrapper: makeWrapper() });

    expect(await screen.findByText("admin@test.com")).toBeDefined();
    expect(await screen.findByText("dev@test.com")).toBeDefined();
  });

  it("renders role badges for members", async () => {
    server.use(
      http.get(`${API}/projects/:id/members`, () =>
        HttpResponse.json(flatMembers)
      )
    );

    render(<MembersList projectId="p1" />, { wrapper: makeWrapper() });

    // Roles are translated via mock: roles.admin, roles.developer
    expect(await screen.findByText("roles.admin")).toBeDefined();
    expect(await screen.findByText("roles.developer")).toBeDefined();
  });

  it("shows add member button", async () => {
    server.use(
      http.get(`${API}/projects/:id/members`, () =>
        HttpResponse.json(flatMembers)
      )
    );

    render(<MembersList projectId="p1" />, { wrapper: makeWrapper() });

    // Button text is translated: projects.addMember
    expect(await screen.findByText("projects.addMember")).toBeDefined();
  });

  it("shows empty state when no members", async () => {
    server.use(
      http.get(`${API}/projects/:id/members`, () =>
        HttpResponse.json([])
      )
    );

    render(<MembersList projectId="p1" />, { wrapper: makeWrapper() });

    expect(await screen.findByText("projects.noMembers")).toBeDefined();
  });

  it("opens add member dialog when add button is clicked", async () => {
    const user = userEvent.setup();

    server.use(
      http.get(`${API}/projects/:id/members`, () =>
        HttpResponse.json(flatMembers)
      )
    );

    render(<MembersList projectId="p1" />, { wrapper: makeWrapper() });

    const addButton = await screen.findByText("projects.addMember");
    await user.click(addButton);

    // AddMemberDialog renders with title translation
    expect(screen.getByText("projects.inviteTitle")).toBeDefined();
  });
});
