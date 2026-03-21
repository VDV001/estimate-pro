import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { Badge } from "../badge";

describe("Badge", () => {
  it("renders text", () => {
    render(<Badge>Status</Badge>);
    expect(screen.getByText("Status")).toBeInTheDocument();
  });

  it("applies default variant", () => {
    render(<Badge>Default</Badge>);
    expect(screen.getByText("Default").className).toContain("bg-primary");
  });

  it("applies secondary variant", () => {
    render(<Badge variant="secondary">Secondary</Badge>);
    expect(screen.getByText("Secondary").className).toContain("bg-secondary");
  });

  it("applies destructive variant", () => {
    render(<Badge variant="destructive">Error</Badge>);
    expect(screen.getByText("Error").className).toContain("bg-destructive");
  });
});
