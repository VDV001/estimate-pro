import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { InlineEdit } from "../inline-edit";

describe("InlineEdit", () => {
  it("renders text value", () => {
    render(<InlineEdit value="Hello" onSave={vi.fn()} />);
    expect(screen.getByText("Hello")).toBeDefined();
  });

  it("shows edit button on hover", () => {
    render(<InlineEdit value="Hello" onSave={vi.fn()} />);
    // Pencil button exists but hidden via opacity
    const btn = screen.getByRole("button");
    expect(btn).toBeDefined();
  });

  it("click edit button shows input", async () => {
    const user = userEvent.setup();
    render(<InlineEdit value="Hello" onSave={vi.fn()} />);

    await user.click(screen.getByRole("button"));
    expect(screen.getByRole("textbox")).toBeDefined();
  });

  it("Enter key saves trimmed value", async () => {
    const user = userEvent.setup();
    const onSave = vi.fn();
    render(<InlineEdit value="Hello" onSave={onSave} />);

    await user.click(screen.getByRole("button"));
    const input = screen.getByRole("textbox");
    await user.clear(input);
    await user.type(input, "World{Enter}");

    expect(onSave).toHaveBeenCalledWith("World");
  });

  it("Escape key reverts without saving", async () => {
    const user = userEvent.setup();
    const onSave = vi.fn();
    render(<InlineEdit value="Hello" onSave={onSave} />);

    await user.click(screen.getByRole("button"));
    const input = screen.getByRole("textbox");
    await user.clear(input);
    await user.type(input, "Changed{Escape}");

    expect(onSave).not.toHaveBeenCalled();
    // Should show original text again
    expect(screen.getByText("Hello")).toBeDefined();
  });

  it("empty value does not trigger save", async () => {
    const user = userEvent.setup();
    const onSave = vi.fn();
    render(<InlineEdit value="Hello" onSave={onSave} />);

    await user.click(screen.getByRole("button"));
    const input = screen.getByRole("textbox");
    await user.clear(input);
    await user.type(input, "{Enter}");

    expect(onSave).not.toHaveBeenCalled();
  });

  it("unchanged value does not trigger save", async () => {
    const user = userEvent.setup();
    const onSave = vi.fn();
    render(<InlineEdit value="Hello" onSave={onSave} />);

    await user.click(screen.getByRole("button"));
    // Just press Enter without changing
    await user.type(screen.getByRole("textbox"), "{Enter}");

    expect(onSave).not.toHaveBeenCalled();
  });

  it("disabled hides edit button", () => {
    render(<InlineEdit value="Hello" onSave={vi.fn()} disabled />);
    expect(screen.queryByRole("button")).toBeNull();
  });
});
