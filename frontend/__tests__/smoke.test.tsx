import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";

describe("test infrastructure", () => {
  it("renders a simple element", () => {
    render(<div>hello</div>);
    expect(screen.getByText("hello")).toBeInTheDocument();
  });

  it("MSW intercepts API requests", async () => {
    const res = await fetch("http://localhost:8080/api/v1/notifications/unread-count");
    const data = await res.json();
    expect(data.count).toBe(3);
  });

  it("useTranslations mock returns key", async () => {
    const { useTranslations } = await import("next-intl");
    const t = useTranslations("test");
    expect(t("hello")).toBe("test.hello");
  });
});
