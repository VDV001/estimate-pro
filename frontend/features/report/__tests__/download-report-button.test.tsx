// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

const downloadMock = vi.fn();
vi.mock("../api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../api")>();
  return {
    ...actual,
    downloadReport: (...args: unknown[]) => downloadMock(...args),
  };
});

import { DownloadReportButton } from "../components/download-report-button";

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

beforeEach(() => {
  downloadMock.mockReset();
  downloadMock.mockResolvedValue(undefined);
});

describe("DownloadReportButton", () => {
  it("renders download button with translated label", () => {
    render(<DownloadReportButton projectId="p1" />, { wrapper: makeWrapper() });
    expect(screen.getByText("report.download")).toBeInTheDocument();
  });

  it("clicking the button opens a format dropdown with three formats", async () => {
    const user = userEvent.setup();
    render(<DownloadReportButton projectId="p1" />, { wrapper: makeWrapper() });

    await user.click(screen.getByText("report.download"));

    expect(screen.getByText("md")).toBeInTheDocument();
    expect(screen.getByText("pdf")).toBeInTheDocument();
    expect(screen.getByText("docx")).toBeInTheDocument();
  });

  it("picking a format calls downloadReport with the right args", async () => {
    const user = userEvent.setup();
    render(<DownloadReportButton projectId="p1" />, { wrapper: makeWrapper() });

    await user.click(screen.getByText("report.download"));
    await user.click(screen.getByText("pdf"));

    await waitFor(() =>
      expect(downloadMock).toHaveBeenCalledWith("p1", "pdf"),
    );
  });

  it("shows error message when download fails", async () => {
    downloadMock.mockRejectedValueOnce(new Error("boom"));
    const user = userEvent.setup();
    render(<DownloadReportButton projectId="p1" />, { wrapper: makeWrapper() });

    await user.click(screen.getByText("report.download"));
    await user.click(screen.getByText("pdf"));

    expect(await screen.findByText("report.error")).toBeInTheDocument();
  });
});
