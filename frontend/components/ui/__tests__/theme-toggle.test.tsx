import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

// Override the global mock for this specific test
const mockSetTheme = vi.fn();
vi.mock("next-themes", () => ({
  useTheme: () => ({
    theme: "light",
    resolvedTheme: "light",
    setTheme: mockSetTheme,
  }),
}));

import { ThemeToggle } from "../theme-toggle";

describe("ThemeToggle", () => {
  it("renders without crash", () => {
    render(<ThemeToggle />);
    expect(screen.getByRole("button")).toBeInTheDocument();
  });

  it("calls setTheme on click", async () => {
    render(<ThemeToggle />);
    await userEvent.click(screen.getByRole("button"));
    expect(mockSetTheme).toHaveBeenCalledWith("dark");
  });

  it("applies custom className", () => {
    render(<ThemeToggle className="custom-class" />);
    const btn = screen.getByRole("button");
    expect(btn.className).toContain("custom-class");
  });
});
