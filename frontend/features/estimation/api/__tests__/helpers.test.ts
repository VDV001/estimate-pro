import { describe, it, expect } from "vitest";
import { pertHours } from "../index";

describe("pertHours", () => {
  it("calculates PERT for (1, 2, 3) → 2", () => {
    expect(pertHours(1, 2, 3)).toBe(2);
  });

  it("returns 0 for (0, 0, 0)", () => {
    expect(pertHours(0, 0, 0)).toBe(0);
  });

  it("calculates PERT for (10, 20, 60) → 25", () => {
    // (10 + 4*20 + 60) / 6 = (10 + 80 + 60) / 6 = 150 / 6 = 25
    expect(pertHours(10, 20, 60)).toBe(25);
  });

  it("returns same value when all inputs are equal", () => {
    expect(pertHours(5, 5, 5)).toBe(5);
  });
});
