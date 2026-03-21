import { describe, it, expect } from "vitest";
import { cn } from "../utils";

describe("cn", () => {
  it("merges classes", () => {
    expect(cn("foo", "bar")).toBe("foo bar");
  });

  it("handles conflicting Tailwind classes", () => {
    expect(cn("px-2", "px-4")).toBe("px-4");
  });

  it("handles conditional classes", () => {
    expect(cn("base", false && "hidden", "visible")).toBe("base visible");
  });

  it("handles undefined and null", () => {
    expect(cn("base", undefined, null)).toBe("base");
  });
});
