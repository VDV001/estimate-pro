import { describe, it, expect, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import RegisterPage from "../register/page";

function makeWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
}

beforeEach(() => {
  localStorage.clear();
});

describe("RegisterPage", () => {
  it("renders name, email and password inputs", () => {
    render(<RegisterPage />, { wrapper: makeWrapper() });
    expect(screen.getByLabelText("auth.name")).toBeDefined();
    expect(screen.getByLabelText("auth.email")).toBeDefined();
    expect(screen.getByLabelText("auth.password")).toBeDefined();
  });

  it("renders register button", () => {
    render(<RegisterPage />, { wrapper: makeWrapper() });
    expect(screen.getByText("auth.register")).toBeDefined();
  });

  it("renders login link for existing users", () => {
    render(<RegisterPage />, { wrapper: makeWrapper() });
    expect(screen.getByText("auth.login")).toBeDefined();
  });

  it("renders OAuth buttons", () => {
    render(<RegisterPage />, { wrapper: makeWrapper() });
    expect(screen.getByText("auth.google")).toBeDefined();
    expect(screen.getByText("auth.github")).toBeDefined();
  });
});
