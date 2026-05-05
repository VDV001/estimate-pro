// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

import { describe, it, expect, beforeEach, vi } from "vitest";
import { http, HttpResponse } from "msw";
import { server } from "@/__tests__/mocks/server";
import { downloadReport, REPORT_FORMATS } from "../api";

const API = "http://localhost:8080/api/v1";

beforeEach(() => {
  localStorage.setItem("ep_access_token", "test-token");
});

describe("REPORT_FORMATS", () => {
  it("contains exactly md/pdf/docx", () => {
    expect([...REPORT_FORMATS].sort()).toEqual(["docx", "md", "pdf"]);
  });
});

describe("downloadReport", () => {
  it("fetches /projects/{id}/report?format=pdf with Authorization", async () => {
    let authHeader: string | null = null;
    let url: string | null = null;
    server.use(
      http.get(`${API}/projects/:id/report`, ({ request }) => {
        url = request.url;
        authHeader = request.headers.get("Authorization");
        return new HttpResponse(new Blob(["%PDF-1.4 ..."]), {
          status: 200,
          headers: {
            "Content-Type": "application/pdf",
            "Content-Disposition": 'attachment; filename="report-p1.pdf"',
          },
        });
      }),
    );

    // Mock browser DOM helpers used to trigger the download.
    const createObjectURL = vi.fn().mockReturnValue("blob:fake");
    const revokeObjectURL = vi.fn();
    Object.defineProperty(URL, "createObjectURL", { value: createObjectURL, configurable: true });
    Object.defineProperty(URL, "revokeObjectURL", { value: revokeObjectURL, configurable: true });

    await downloadReport("p1", "pdf");

    expect(url).not.toBeNull();
    expect(url!).toContain("/projects/p1/report?format=pdf");
    expect(authHeader).toBe("Bearer test-token");
    expect(createObjectURL).toHaveBeenCalled();
  });

  it("uses filename from Content-Disposition", async () => {
    server.use(
      http.get(`${API}/projects/:id/report`, () =>
        new HttpResponse(new Blob(["%PDF-1.4"]), {
          status: 200,
          headers: {
            "Content-Type": "application/pdf",
            "Content-Disposition": 'attachment; filename="report-p1.pdf"',
          },
        }),
      ),
    );

    const createdAnchors: HTMLAnchorElement[] = [];
    const originalCreate = document.createElement.bind(document);
    vi.spyOn(document, "createElement").mockImplementation((tagName: string) => {
      const el = originalCreate(tagName);
      if (tagName === "a") {
        createdAnchors.push(el as HTMLAnchorElement);
      }
      return el;
    });
    Object.defineProperty(URL, "createObjectURL", { value: vi.fn().mockReturnValue("blob:x"), configurable: true });
    Object.defineProperty(URL, "revokeObjectURL", { value: vi.fn(), configurable: true });

    await downloadReport("p1", "pdf");

    const downloaded = createdAnchors.find((a) => a.download !== "");
    expect(downloaded?.download).toBe("report-p1.pdf");
    vi.restoreAllMocks();
  });

  it("throws on 4xx", async () => {
    server.use(
      http.get(`${API}/projects/:id/report`, () =>
        HttpResponse.json(
          { error: { code: "CONFLICT", message: "no submitted estimations" } },
          { status: 409 },
        ),
      ),
    );

    await expect(downloadReport("p1", "pdf")).rejects.toThrow();
  });
});
