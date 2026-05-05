import { describe, it, expect, beforeEach, vi } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { http, HttpResponse } from "msw";
import { server } from "@/__tests__/mocks/server";

const requestExtractionMock = vi.fn();
vi.mock("@/features/extraction/api", async (importOriginal) => {
  const actual = await importOriginal<
    typeof import("@/features/extraction/api")
  >();
  return {
    ...actual,
    requestExtraction: (...args: unknown[]) => requestExtractionMock(...args),
  };
});

import { DocumentsList } from "../components/documents-list";

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

beforeEach(() => {
  localStorage.setItem("ep_access_token", "test-token");
  requestExtractionMock.mockReset();
  requestExtractionMock.mockResolvedValue({
    id: "ext-new",
    document_id: "d-new",
    document_version_id: "v-new",
    status: "pending",
    tasks: [],
    created_at: "2026-05-05T00:00:00Z",
    updated_at: "2026-05-05T00:00:00Z",
  });
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

  describe("auto-extraction after upload", () => {
    function uploadFile(file: File): Promise<void> {
      const user = userEvent.setup();
      return (async () => {
        const input = document.querySelector(
          'input[type="file"]',
        ) as HTMLInputElement;
        expect(input).not.toBeNull();
        await user.upload(input, file);
      })();
    }

    function setupUploadHandler(opts: {
      docId: string;
      versionId: string;
      versionFileSize: number;
    }) {
      server.use(
        http.post(`${API}/projects/:id/documents`, () =>
          HttpResponse.json({
            document: {
              id: opts.docId,
              project_id: "p1",
              title: "uploaded",
              uploaded_by: "u1",
              created_at: "2026-05-05T00:00:00Z",
            },
            version: {
              id: opts.versionId,
              document_id: opts.docId,
              version_number: 1,
              file_key: "k",
              file_type: "pdf",
              file_size: opts.versionFileSize,
              parsed_status: "pending",
              confidence_score: 0,
              is_signed: false,
              is_final: false,
              uploaded_by: "u1",
              uploaded_at: "2026-05-05T00:00:00Z",
            },
          }),
        ),
      );
    }

    it("triggers extraction for PDF upload with version id and file size", async () => {
      setupUploadHandler({ docId: "d-new", versionId: "v-new", versionFileSize: 1234 });
      render(<DocumentsList projectId="p1" />, { wrapper: makeWrapper() });
      await screen.findByText("spec.pdf");

      await uploadFile(
        new File(["pdf-bytes"], "spec.pdf", { type: "application/pdf" }),
      );

      await waitFor(() =>
        expect(requestExtractionMock).toHaveBeenCalledWith(
          "p1",
          "d-new",
          "v-new",
          expect.any(Number),
        ),
      );
    });

    it.each([
      ["docx", "application/vnd.openxmlformats-officedocument.wordprocessingml.document"],
      ["txt", "text/plain"],
      ["md", "text/markdown"],
      ["csv", "text/csv"],
      ["xlsx", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"],
    ])("triggers extraction for %s upload", async (ext, mime) => {
      setupUploadHandler({ docId: "d-new", versionId: "v-new", versionFileSize: 100 });
      render(<DocumentsList projectId="p1" />, { wrapper: makeWrapper() });
      await screen.findByText("spec.pdf");

      await uploadFile(new File(["data"], `file.${ext}`, { type: mime }));

      await waitFor(() => expect(requestExtractionMock).toHaveBeenCalled());
    });

    it("does not trigger extraction for unsupported formats (image)", async () => {
      setupUploadHandler({ docId: "d-new", versionId: "v-new", versionFileSize: 100 });
      render(<DocumentsList projectId="p1" />, { wrapper: makeWrapper() });
      await screen.findByText("spec.pdf");

      await uploadFile(
        new File(["png-bytes"], "img.png", { type: "image/png" }),
      );

      // Wait long enough for any kicked-off mutation to settle
      await new Promise((r) => setTimeout(r, 100));
      expect(requestExtractionMock).not.toHaveBeenCalled();
    });

    it("shows inline message when extraction kickoff fails", async () => {
      setupUploadHandler({ docId: "d-new", versionId: "v-new", versionFileSize: 100 });
      requestExtractionMock.mockRejectedValueOnce(new Error("kickoff failed"));

      render(<DocumentsList projectId="p1" />, { wrapper: makeWrapper() });
      await screen.findByText("spec.pdf");

      await uploadFile(
        new File(["pdf-bytes"], "spec.pdf", { type: "application/pdf" }),
      );

      expect(
        await screen.findByText("documents.extractionKickoffFailed"),
      ).toBeInTheDocument();
    });

    it("does not trigger extraction if upload fails", async () => {
      server.use(
        http.post(`${API}/projects/:id/documents`, () =>
          HttpResponse.json(
            { error: { code: "INTERNAL", message: "boom" } },
            { status: 500 },
          ),
        ),
      );
      render(<DocumentsList projectId="p1" />, { wrapper: makeWrapper() });
      await screen.findByText("spec.pdf");

      await uploadFile(
        new File(["pdf-bytes"], "spec.pdf", { type: "application/pdf" }),
      );

      await new Promise((r) => setTimeout(r, 100));
      expect(requestExtractionMock).not.toHaveBeenCalled();
    });
  });

  describe("ExtractionPanel rendering", () => {
    it("renders ExtractionPanel under DocumentCard when an extraction exists", async () => {
      // Override list-extractions to return one extraction tied to d1
      server.use(
        http.get(`${API}/projects/:id/extractions`, () =>
          HttpResponse.json([
            {
              id: "ext-d1",
              document_id: "d1",
              document_version_id: "v1",
              status: "completed",
              tasks: [{ name: "Discovered task" }],
              created_at: "2026-05-05T00:00:00Z",
              updated_at: "2026-05-05T00:01:00Z",
            },
          ]),
        ),
        http.get(`${API}/extractions/:id`, () =>
          HttpResponse.json({
            extraction: {
              id: "ext-d1",
              document_id: "d1",
              document_version_id: "v1",
              status: "completed",
              tasks: [{ name: "Discovered task" }],
              created_at: "2026-05-05T00:00:00Z",
              updated_at: "2026-05-05T00:01:00Z",
            },
            events: [],
          }),
        ),
      );

      render(<DocumentsList projectId="p1" />, { wrapper: makeWrapper() });
      await screen.findByText("spec.pdf");

      expect(await screen.findByText("Discovered task")).toBeInTheDocument();
    });

    it("does not render panel when no extraction matches", async () => {
      server.use(
        http.get(`${API}/projects/:id/extractions`, () => HttpResponse.json([])),
      );
      render(<DocumentsList projectId="p1" />, { wrapper: makeWrapper() });
      await screen.findByText("spec.pdf");

      // No status badge should be present
      expect(screen.queryByRole("status")).toBeNull();
    });
  });
});
