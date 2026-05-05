// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ExtractionEnvelope, ExtractionStatus } from "../api";

// Hoisted mock state — read by both the vi.mock factory and the tests.
const hookState = vi.hoisted(() => ({
  data: undefined as ExtractionEnvelope | undefined,
  isLoading: false,
  isError: false,
}));

vi.mock("../hooks/use-extraction-status", () => ({
  useExtractionStatus: () => hookState,
  isTerminalStatus: (s: ExtractionStatus) =>
    s === "completed" || s === "failed" || s === "cancelled",
  POLL_INTERVAL_MS: 5000,
}));

const cancelMock = vi.fn();
const retryMock = vi.fn();
vi.mock("../api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../api")>();
  return {
    ...actual,
    cancelExtraction: (...args: unknown[]) => cancelMock(...args),
    retryExtraction: (...args: unknown[]) => retryMock(...args),
  };
});

import { ExtractionPanel } from "../components/extraction-panel";

function makeWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  const Wrapper = ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
  Wrapper.displayName = "TestWrapper";
  return Wrapper;
}

function envelope(
  status: ExtractionStatus,
  overrides: Partial<ExtractionEnvelope["extraction"]> = {},
): ExtractionEnvelope {
  return {
    extraction: {
      id: "ext1",
      document_id: "d1",
      document_version_id: "v1",
      status,
      tasks: [],
      created_at: "2026-05-05T00:00:00Z",
      updated_at: "2026-05-05T00:00:00Z",
      ...overrides,
    },
    events: [],
  };
}

beforeEach(() => {
  hookState.data = undefined;
  hookState.isLoading = false;
  hookState.isError = false;
  cancelMock.mockReset();
  retryMock.mockReset();
  cancelMock.mockResolvedValue(undefined);
  retryMock.mockResolvedValue(undefined);
});

describe("ExtractionPanel", () => {
  it("shows loading state while data is undefined and isLoading=true", () => {
    hookState.isLoading = true;
    render(<ExtractionPanel extractionId="ext1" />, { wrapper: makeWrapper() });
    expect(screen.getByText("common.loading")).toBeInTheDocument();
  });

  it("renders status badge for pending", () => {
    hookState.data = envelope("pending");
    render(<ExtractionPanel extractionId="ext1" />, { wrapper: makeWrapper() });
    expect(screen.getByText("extraction.status.pending")).toBeInTheDocument();
  });

  it("processing shows badge + cancel button (no retry, no tasks)", () => {
    hookState.data = envelope("processing");
    render(<ExtractionPanel extractionId="ext1" />, { wrapper: makeWrapper() });

    expect(screen.getByText("extraction.status.processing")).toBeInTheDocument();
    expect(screen.getByText("extraction.actions.cancel")).toBeInTheDocument();
    expect(screen.queryByText("extraction.actions.retry")).toBeNull();
  });

  it("completed shows tasks list with names + estimate hints", () => {
    hookState.data = envelope("completed", {
      tasks: [
        { name: "Implement login", estimate_hint: "small" },
        { name: "Wire OAuth" },
      ],
    });
    render(<ExtractionPanel extractionId="ext1" />, { wrapper: makeWrapper() });

    expect(screen.getByText("Implement login")).toBeInTheDocument();
    expect(screen.getByText("small")).toBeInTheDocument();
    expect(screen.getByText("Wire OAuth")).toBeInTheDocument();
    expect(
      screen.getByText("extraction.tasksFound"),
    ).toBeInTheDocument();
    // No retry / cancel on completed
    expect(screen.queryByText("extraction.actions.retry")).toBeNull();
    expect(screen.queryByText("extraction.actions.cancel")).toBeNull();
  });

  it("completed with empty tasks shows noTasks copy", () => {
    hookState.data = envelope("completed", { tasks: [] });
    render(<ExtractionPanel extractionId="ext1" />, { wrapper: makeWrapper() });
    expect(screen.getByText("extraction.noTasks")).toBeInTheDocument();
  });

  it("failed shows translated reason + retry button", () => {
    hookState.data = envelope("failed", {
      failure_reason: "encrypted file (password protected)",
    });
    render(<ExtractionPanel extractionId="ext1" />, { wrapper: makeWrapper() });

    expect(
      screen.getByText("extraction.reason.encrypted"),
    ).toBeInTheDocument();
    expect(screen.getByText("extraction.actions.retry")).toBeInTheDocument();
    expect(screen.queryByText("extraction.actions.cancel")).toBeNull();
  });

  it("failed with unmapped reason falls through to unknown copy (no leak)", () => {
    hookState.data = envelope("failed", {
      failure_reason: "some new English sentinel",
    });
    render(<ExtractionPanel extractionId="ext1" />, { wrapper: makeWrapper() });

    expect(screen.getByText("extraction.reason.unknown")).toBeInTheDocument();
    expect(
      screen.queryByText("some new English sentinel"),
    ).toBeNull();
  });

  it("cancelled shows reason + retry (no cancel)", () => {
    hookState.data = envelope("cancelled");
    render(<ExtractionPanel extractionId="ext1" />, { wrapper: makeWrapper() });

    expect(
      screen.getByText("extraction.reason.cancelled"),
    ).toBeInTheDocument();
    expect(screen.getByText("extraction.actions.retry")).toBeInTheDocument();
    expect(screen.queryByText("extraction.actions.cancel")).toBeNull();
  });

  it("cancel button calls cancelExtraction after window.confirm", async () => {
    const user = userEvent.setup();
    vi.spyOn(window, "confirm").mockReturnValue(true);
    hookState.data = envelope("processing");

    render(<ExtractionPanel extractionId="ext1" />, { wrapper: makeWrapper() });
    await user.click(screen.getByText("extraction.actions.cancel"));

    expect(window.confirm).toHaveBeenCalled();
    expect(cancelMock).toHaveBeenCalledWith("ext1");
    vi.restoreAllMocks();
  });

  it("cancel button does nothing when user declines confirm", async () => {
    const user = userEvent.setup();
    vi.spyOn(window, "confirm").mockReturnValue(false);
    hookState.data = envelope("processing");

    render(<ExtractionPanel extractionId="ext1" />, { wrapper: makeWrapper() });
    await user.click(screen.getByText("extraction.actions.cancel"));

    expect(cancelMock).not.toHaveBeenCalled();
    vi.restoreAllMocks();
  });

  it("retry button calls retryExtraction without confirm", async () => {
    const user = userEvent.setup();
    hookState.data = envelope("failed", { failure_reason: "pipeline error" });

    render(<ExtractionPanel extractionId="ext1" />, { wrapper: makeWrapper() });
    await user.click(screen.getByText("extraction.actions.retry"));

    expect(retryMock).toHaveBeenCalledWith("ext1");
  });

  it("renders nothing when isError=true and no data", () => {
    hookState.isError = true;
    const { container } = render(
      <ExtractionPanel extractionId="ext1" />,
      { wrapper: makeWrapper() },
    );
    expect(container.firstChild).toBeNull();
  });
});
