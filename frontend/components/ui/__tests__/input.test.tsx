import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Input } from "../input";

describe("Input", () => {
  it("renders with placeholder", () => {
    render(<Input placeholder="Enter text" />);
    expect(screen.getByPlaceholderText("Enter text")).toBeInTheDocument();
  });

  it("handles user input", async () => {
    render(<Input placeholder="Type here" />);
    const input = screen.getByPlaceholderText("Type here");
    await userEvent.type(input, "hello");
    expect(input).toHaveValue("hello");
  });

  it("respects disabled state", () => {
    render(<Input disabled placeholder="Disabled" />);
    expect(screen.getByPlaceholderText("Disabled")).toBeDisabled();
  });

  it("renders with type", () => {
    render(<Input type="email" placeholder="Email" />);
    expect(screen.getByPlaceholderText("Email")).toHaveAttribute("type", "email");
  });
});
