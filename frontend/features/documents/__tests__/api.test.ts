import { describe, it, expect, beforeEach } from "vitest";
import {
  listDocuments,
  getDocument,
  uploadDocument,
  deleteDocument,
  updateVersionFlags,
  setVersionTags,
} from "../api";

beforeEach(() => {
  localStorage.setItem("ep_access_token", "test-token");
});

describe("documents API", () => {
  it("listDocuments returns documents array", async () => {
    const docs = await listDocuments("p1");
    expect(docs).toHaveLength(1);
    expect(docs[0].title).toBe("spec.pdf");
  });

  it("getDocument returns doc with latest_version", async () => {
    const result = await getDocument("p1", "d1");
    expect(result).toHaveProperty("id", "d1");
    expect(result).toHaveProperty("latest_version");
    expect(result.latest_version).toHaveProperty("is_signed", false);
  });

  it("uploadDocument sends FormData", async () => {
    const file = new File(["content"], "test.pdf", { type: "application/pdf" });
    const result = await uploadDocument("p1", file);
    expect(result).toHaveProperty("id", "d-new");
  });

  it("uploadDocument with title", async () => {
    const file = new File(["content"], "test.pdf", { type: "application/pdf" });
    const result = await uploadDocument("p1", file, "Custom Title");
    expect(result).toHaveProperty("id", "d-new");
  });

  it("deleteDocument sends DELETE", async () => {
    await expect(deleteDocument("p1", "d1")).resolves.toBeUndefined();
  });

  it("updateVersionFlags sends PATCH with flags", async () => {
    await expect(
      updateVersionFlags("p1", "d1", "v1", { is_signed: true, is_final: false }),
    ).resolves.toBeUndefined();
  });

  it("setVersionTags sends PUT with tags", async () => {
    await expect(
      setVersionTags("p1", "d1", "v1", ["urgent", "draft"]),
    ).resolves.toBeUndefined();
  });
});
