// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { http, HttpResponse } from "msw";
import { server } from "../../../__tests__/mocks/server";
import { CreateProjectDialog } from "../components/create-project-dialog";

function makeWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  const wrapper = ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
  return wrapper;
}

function renderDialog(props: { open: boolean; onOpenChange: (open: boolean) => void; workspaceId: string }) {
  const wrapper = makeWrapper();
  return render(<CreateProjectDialog {...props} />, { wrapper });
}

describe("CreateProjectDialog", () => {
  it("renders dialog content when open", () => {
    renderDialog({ open: true, onOpenChange: () => {}, workspaceId: "w1" });

    // Uses mock translations: ns.key
    expect(screen.getByText("projects.createTitle")).toBeDefined();
    expect(screen.getByText("projects.createDesc")).toBeDefined();
  });

  it("does not show dialog content when closed", () => {
    renderDialog({ open: false, onOpenChange: () => {}, workspaceId: "w1" });

    expect(screen.queryByText("projects.createTitle")).toBeNull();
  });

  it("renders name and description inputs when open", () => {
    renderDialog({ open: true, onOpenChange: () => {}, workspaceId: "w1" });

    expect(screen.getByLabelText("projects.name")).toBeDefined();
    expect(screen.getByLabelText("projects.description")).toBeDefined();
  });

  it("submits form with name and description", async () => {
    const user = userEvent.setup();

    let capturedBody: Record<string, unknown> | null = null;
    server.use(
      http.post("http://localhost:8080/api/v1/projects", async ({ request }) => {
        capturedBody = (await request.json()) as Record<string, unknown>;
        return HttpResponse.json({
          id: "p-new",
          name: capturedBody.name,
          description: capturedBody.description,
          status: "active",
          created_at: "2026-03-21T00:00:00Z",
        });
      })
    );

    const onOpenChange = () => {};
    renderDialog({ open: true, onOpenChange, workspaceId: "w1" });

    const nameInput = screen.getByLabelText("projects.name");
    const descInput = screen.getByLabelText("projects.description");

    await user.type(nameInput, "My New Project");
    await user.type(descInput, "A project description");

    const submitButton = screen.getByRole("button", { name: "projects.create" });
    await user.click(submitButton);

    // Wait for the mutation to complete (fields reset on success, dialog closes)
    await vi.waitFor(() => {
      expect(capturedBody).not.toBeNull();
    });

    expect(capturedBody!.name).toBe("My New Project");
    expect(capturedBody!.description).toBe("A project description");
    expect(capturedBody!.workspace_id).toBe("w1");
  });

  it("keeps submit button disabled when name is empty", async () => {
    renderDialog({ open: true, onOpenChange: () => {}, workspaceId: "w1" });

    const submitButton = screen.getByRole("button", { name: "projects.create" });
    expect((submitButton as HTMLButtonElement).disabled).toBe(true);
  });

  it("calls onOpenChange(false) when cancel is clicked", async () => {
    const user = userEvent.setup();
    let isOpen = true;
    const onOpenChange = (open: boolean) => { isOpen = open; };

    renderDialog({ open: true, onOpenChange, workspaceId: "w1" });

    const cancelButton = screen.getByRole("button", { name: /cancel/i });
    await user.click(cancelButton);

    expect(isOpen).toBe(false);
  });
});
