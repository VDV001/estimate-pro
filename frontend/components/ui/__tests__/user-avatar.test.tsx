import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { UserAvatar } from "../user-avatar";

describe("UserAvatar", () => {
  it("renders initials when no avatar URL", () => {
    render(<UserAvatar name="Daniil Vdovin" />);
    expect(screen.getByText("DV")).toBeInTheDocument();
  });

  it("renders single initial for single name", () => {
    render(<UserAvatar name="Admin" />);
    expect(screen.getByText("A")).toBeInTheDocument();
  });

  it("renders ? for missing name", () => {
    render(<UserAvatar />);
    expect(screen.getByText("?")).toBeInTheDocument();
  });

  it("applies sm size class", () => {
    const { container } = render(<UserAvatar name="Test" size="sm" />);
    expect(container.firstElementChild?.className).toContain("h-8");
  });

  it("applies lg size class", () => {
    const { container } = render(<UserAvatar name="Test" size="lg" />);
    expect(container.firstElementChild?.className).toContain("h-24");
  });

  it("renders Image when external URL provided", () => {
    render(<UserAvatar name="Test" avatarUrl="https://example.com/avatar.jpg" />);
    const img = screen.getByAltText("Test");
    expect(img).toBeInTheDocument();
  });
});
