import { describe, it, expect, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import LoginPage from "../login/page";

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

describe("LoginPage", () => {
  it("renders email and password inputs", () => {
    render(<LoginPage />, { wrapper: makeWrapper() });
    expect(screen.getByLabelText("auth.email")).toBeDefined();
    expect(screen.getByLabelText("auth.password")).toBeDefined();
  });

  it("renders login button", () => {
    render(<LoginPage />, { wrapper: makeWrapper() });
    expect(screen.getByText("auth.login")).toBeDefined();
  });

  it("renders forgot password link", () => {
    render(<LoginPage />, { wrapper: makeWrapper() });
    expect(screen.getByText("auth.forgotPassword")).toBeDefined();
  });

  it("renders register link", () => {
    render(<LoginPage />, { wrapper: makeWrapper() });
    expect(screen.getByText("auth.register")).toBeDefined();
  });

  it("renders OAuth buttons", () => {
    render(<LoginPage />, { wrapper: makeWrapper() });
    expect(screen.getByText("auth.google")).toBeDefined();
    expect(screen.getByText("auth.github")).toBeDefined();
  });

  it("renders EP logo", () => {
    render(<LoginPage />, { wrapper: makeWrapper() });
    expect(screen.getByText("EP")).toBeDefined();
    expect(screen.getByText("EstimatePro")).toBeDefined();
  });
});
