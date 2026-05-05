// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

import { describe, it, expect } from "vitest";
import {
  mapExtractionReason,
  type ExtractionReasonKey,
} from "../lib/reason-mapping";

describe("mapExtractionReason", () => {
  it.each<[string, ExtractionReasonKey]>([
    ["encrypted file (password protected)", "encrypted"],
    ["LLM service error", "llmService"],
    ["prompt injection detected", "promptInjection"],
    ["LLM response failed schema validation", "schemaInvalid"],
    ["pipeline error", "pipelineError"],
    ["cancelled", "cancelled"],
  ])("known reason %s -> key %s", (reason, expected) => {
    expect(mapExtractionReason(reason)).toBe(expected);
  });

  it.each<[string | null | undefined, ExtractionReasonKey]>([
    [undefined, "unknown"],
    [null, "unknown"],
    ["", "unknown"],
    ["some new untranslated reason", "unknown"],
    ["random English string", "unknown"],
  ])("falsy or unmapped reason %j -> unknown", (reason, expected) => {
    expect(mapExtractionReason(reason)).toBe(expected);
  });

  it("does not concatenate raw reason into return value (no leak)", () => {
    const result = mapExtractionReason("a brand new untranslated sentinel");
    expect(result).toBe("unknown");
    expect(String(result)).not.toContain("brand new");
  });
});
