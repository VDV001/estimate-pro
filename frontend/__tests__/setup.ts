import "@testing-library/jest-dom/vitest";
import { afterAll, afterEach, beforeAll, vi } from "vitest";
import { cleanup } from "@testing-library/react";
import { createElement } from "react";
import { server } from "./mocks/server";

// MSW server lifecycle
beforeAll(() => server.listen({ onUnhandledRequest: "warn" }));
afterEach(() => {
  server.resetHandlers();
  cleanup();
});
afterAll(() => server.close());

// Mock next-intl
vi.mock("next-intl", () => ({
  useTranslations: (ns?: string) => {
    return (key: string, values?: Record<string, unknown>) => {
      const full = ns ? `${ns}.${key}` : key;
      if (values) {
        return Object.entries(values).reduce(
          (str, [k, v]) => str.replace(`{${k}}`, String(v)),
          full,
        );
      }
      return full;
    };
  },
  NextIntlClientProvider: ({ children }: { children: React.ReactNode }) => children,
}));

// Mock next-themes
vi.mock("next-themes", () => ({
  useTheme: () => ({
    theme: "light",
    resolvedTheme: "light",
    setTheme: vi.fn(),
  }),
  ThemeProvider: ({ children }: { children: React.ReactNode }) => children,
}));

// Mock i18n/navigation
vi.mock("@/i18n/navigation", () => ({
  Link: ({ children, href }: { children: React.ReactNode; href: string }) =>
    createElement("a", { href }, children),
  useRouter: () => ({
    push: vi.fn(),
    replace: vi.fn(),
    back: vi.fn(),
    prefetch: vi.fn(),
  }),
  usePathname: () => "/dashboard",
  redirect: vi.fn(),
}));

// Mock next/navigation
vi.mock("next/navigation", () => ({
  useRouter: () => ({
    push: vi.fn(),
    replace: vi.fn(),
    back: vi.fn(),
    prefetch: vi.fn(),
  }),
  usePathname: () => "/dashboard",
  useSearchParams: () => new URLSearchParams(),
  redirect: vi.fn(),
}));
