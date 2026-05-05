// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

import { describe, it, expect, beforeEach } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { server } from "@/__tests__/mocks/server";
import {
  useExtractionStatus,
  isTerminalStatus,
  POLL_INTERVAL_MS,
} from "../hooks/use-extraction-status";
import type { ExtractionStatus } from "../api";

const API = "http://localhost:8080/api/v1";

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

function envelopeWithStatus(id: string, status: ExtractionStatus) {
  return {
    extraction: {
      id,
      document_id: "d1",
      document_version_id: "v1",
      status,
      tasks: [],
      created_at: "2026-05-05T00:00:00Z",
      updated_at: "2026-05-05T00:00:00Z",
    },
    events: [],
  };
}

beforeEach(() => {
  localStorage.setItem("ep_access_token", "test-token");
});

describe("isTerminalStatus", () => {
  it.each<[ExtractionStatus, boolean]>([
    ["pending", false],
    ["processing", false],
    ["completed", true],
    ["failed", true],
    ["cancelled", true],
  ])("status %s -> terminal=%s", (status, expected) => {
    expect(isTerminalStatus(status)).toBe(expected);
  });
});

describe("useExtractionStatus", () => {
  it("POLL_INTERVAL_MS is 5 seconds", () => {
    expect(POLL_INTERVAL_MS).toBe(5000);
  });

  it("returns extraction envelope from API", async () => {
    server.use(
      http.get(`${API}/extractions/:id`, () =>
        HttpResponse.json(envelopeWithStatus("ext1", "pending")),
      ),
    );

    const { result } = renderHook(() => useExtractionStatus("ext1"), {
      wrapper: makeWrapper(),
    });

    await waitFor(() => expect(result.current.data).toBeDefined());
    expect(result.current.data!.extraction.id).toBe("ext1");
    expect(result.current.data!.extraction.status).toBe("pending");
    expect(result.current.isError).toBe(false);
  });

  it("does not query when extractionId is undefined", () => {
    const { result } = renderHook(() => useExtractionStatus(undefined), {
      wrapper: makeWrapper(),
    });

    expect(result.current.data).toBeUndefined();
    expect(result.current.isLoading).toBe(false);
  });

  it("isLoading=true while fetching", async () => {
    const { result } = renderHook(() => useExtractionStatus("ext1"), {
      wrapper: makeWrapper(),
    });

    expect(result.current.isLoading).toBe(true);
    await waitFor(() => expect(result.current.isLoading).toBe(false));
  });

  it.each<ExtractionStatus>(["completed", "failed", "cancelled"])(
    "stops polling once status is terminal (%s)",
    async (status) => {
      let callCount = 0;
      server.use(
        http.get(`${API}/extractions/:id`, () => {
          callCount++;
          return HttpResponse.json(envelopeWithStatus("ext1", status));
        }),
      );

      const { result, unmount } = renderHook(
        () => useExtractionStatus("ext1"),
        { wrapper: makeWrapper() },
      );

      await waitFor(() => expect(result.current.data).toBeDefined());
      expect(result.current.data!.extraction.status).toBe(status);
      const baseline = callCount;

      // Wait long enough for at least one poll interval — terminal status
      // must keep callCount stable.
      await new Promise((resolve) =>
        setTimeout(resolve, POLL_INTERVAL_MS + 200),
      );
      expect(callCount).toBe(baseline);

      unmount();
    },
    POLL_INTERVAL_MS + 2000,
  );

  it("isError=true on 4xx", async () => {
    server.use(
      http.get(`${API}/extractions/:id`, () =>
        HttpResponse.json(
          { error: { code: "NOT_FOUND", message: "missing" } },
          { status: 404 },
        ),
      ),
    );

    const { result } = renderHook(() => useExtractionStatus("missing"), {
      wrapper: makeWrapper(),
    });

    await waitFor(() => expect(result.current.isError).toBe(true));
  });
});
