import { describe, it, expect, beforeEach } from "vitest";
import { listDocuments, deleteDocument, updateVersionFlags } from "../api";

beforeEach(() => {
  localStorage.setItem("access_token", "test-token");
});

describe("documents API", () => {
  it("listDocuments returns documents array", async () => {
    const docs = await listDocuments("p1");
    expect(docs).toHaveLength(1);
    expect(docs[0].title).toBe("spec.pdf");
  });

  it("deleteDocument sends DELETE", async () => {
    await expect(deleteDocument("p1", "d1")).resolves.toBeUndefined();
  });

  it("updateVersionFlags sends PATCH with flags", async () => {
    await expect(
      updateVersionFlags("p1", "d1", "v1", { is_signed: true, is_final: false }),
    ).resolves.toBeUndefined();
  });
});
