// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

import { describe, it, expect, beforeEach } from "vitest";
import { http, HttpResponse } from "msw";
import { server } from "@/__tests__/mocks/server";
import {
  requestExtraction,
  getExtraction,
  cancelExtraction,
  retryExtraction,
  listExtractions,
} from "../api";

const API = "http://localhost:8080/api/v1";

beforeEach(() => {
  localStorage.setItem("ep_access_token", "test-token");
});

describe("extraction API", () => {
  describe("requestExtraction", () => {
    it("POSTs to extraction endpoint with file_size body and returns 201 extraction", async () => {
      const ext = await requestExtraction("p1", "d1", "v1", 4096);
      expect(ext.id).toBe("ext-new");
      expect(ext.document_id).toBe("d1");
      expect(ext.document_version_id).toBe("v1");
      expect(ext.status).toBe("pending");
    });

    it("sends file_size in JSON body", async () => {
      let received: { file_size?: number } | null = null;
      server.use(
        http.post(
          `${API}/projects/:projectId/documents/:docId/versions/:versionId/extractions`,
          async ({ request }) => {
            received = (await request.json()) as { file_size?: number };
            return HttpResponse.json(
              {
                id: "ext-new",
                document_id: "d1",
                document_version_id: "v1",
                status: "pending",
                tasks: [],
                created_at: "2026-05-05T00:00:00Z",
                updated_at: "2026-05-05T00:00:00Z",
              },
              { status: 201 },
            );
          },
        ),
      );

      await requestExtraction("p1", "d1", "v1", 8192);
      expect(received).not.toBeNull();
      expect(received!.file_size).toBe(8192);
    });

    it("sends Bearer token in Authorization header", async () => {
      let authHeader: string | null = null;
      server.use(
        http.post(
          `${API}/projects/:projectId/documents/:docId/versions/:versionId/extractions`,
          ({ request }) => {
            authHeader = request.headers.get("Authorization");
            return HttpResponse.json(
              {
                id: "ext-new",
                document_id: "d1",
                document_version_id: "v1",
                status: "pending",
                tasks: [],
                created_at: "2026-05-05T00:00:00Z",
                updated_at: "2026-05-05T00:00:00Z",
              },
              { status: 201 },
            );
          },
        ),
      );

      await requestExtraction("p1", "d1", "v1", 1024);
      expect(authHeader).toBe("Bearer test-token");
    });
  });

  describe("getExtraction", () => {
    it("GETs envelope with extraction + events", async () => {
      const env = await getExtraction("ext-abc");
      expect(env.extraction.id).toBe("ext-abc");
      expect(env.extraction.status).toBe("completed");
      expect(env.extraction.tasks).toHaveLength(2);
      expect(env.extraction.tasks[0].name).toBe("Implement login");
      expect(env.extraction.tasks[0].estimate_hint).toBe("small");
      expect(env.events).toHaveLength(1);
      expect(env.events[0].from_status).toBe("pending");
    });
  });

  describe("cancelExtraction", () => {
    it("POSTs to cancel and resolves on 204", async () => {
      await expect(cancelExtraction("ext-abc")).resolves.toBeUndefined();
    });

    it("uses POST method", async () => {
      let method: string | null = null;
      server.use(
        http.post(`${API}/extractions/:extractionId/cancel`, ({ request }) => {
          method = request.method;
          return new HttpResponse(null, { status: 204 });
        }),
      );
      await cancelExtraction("ext-abc");
      expect(method).toBe("POST");
    });
  });

  describe("retryExtraction", () => {
    it("POSTs to retry and resolves on 204", async () => {
      await expect(retryExtraction("ext-abc")).resolves.toBeUndefined();
    });

    it("uses POST method", async () => {
      let method: string | null = null;
      server.use(
        http.post(`${API}/extractions/:extractionId/retry`, ({ request }) => {
          method = request.method;
          return new HttpResponse(null, { status: 204 });
        }),
      );
      await retryExtraction("ext-abc");
      expect(method).toBe("POST");
    });
  });

  describe("listExtractions", () => {
    it("GETs project extractions array", async () => {
      const list = await listExtractions("p1");
      expect(list).toHaveLength(1);
      expect(list[0].id).toBe("ext1");
      expect(list[0].status).toBe("completed");
    });
  });

  describe("error handling", () => {
    it("requestExtraction throws on 4xx", async () => {
      server.use(
        http.post(
          `${API}/projects/:projectId/documents/:docId/versions/:versionId/extractions`,
          () =>
            HttpResponse.json(
              { error: { code: "PAYLOAD_TOO_LARGE", message: "file too big" } },
              { status: 413 },
            ),
        ),
      );
      await expect(requestExtraction("p1", "d1", "v1", 99999999)).rejects.toThrow();
    });

    it("getExtraction throws on 404", async () => {
      server.use(
        http.get(`${API}/extractions/:extractionId`, () =>
          HttpResponse.json(
            { error: { code: "NOT_FOUND", message: "not found" } },
            { status: 404 },
          ),
        ),
      );
      await expect(getExtraction("missing")).rejects.toThrow();
    });
  });
});
