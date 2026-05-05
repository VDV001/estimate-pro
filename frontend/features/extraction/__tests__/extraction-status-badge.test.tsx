// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { ExtractionStatusBadge } from "../components/extraction-status-badge";
import type { ExtractionStatus } from "../api";

describe("ExtractionStatusBadge", () => {
  it.each<ExtractionStatus>([
    "pending",
    "processing",
    "completed",
    "failed",
    "cancelled",
  ])("renders translated label for status %s", (status) => {
    render(<ExtractionStatusBadge status={status} />);
    expect(screen.getByText(`extraction.status.${status}`)).toBeInTheDocument();
  });

  it.each<[ExtractionStatus, string]>([
    ["pending", "lucide-clock"],
    ["processing", "lucide-loader-circle"],
    ["completed", "lucide-circle-check"],
    ["failed", "lucide-circle-x"],
    ["cancelled", "lucide-ban"],
  ])("status %s shows icon %s", (status, iconClass) => {
    const { container } = render(<ExtractionStatusBadge status={status} />);
    expect(container.querySelector(`svg.${iconClass}`)).not.toBeNull();
  });

  it("processing icon is animated (spins)", () => {
    const { container } = render(<ExtractionStatusBadge status="processing" />);
    const icon = container.querySelector("svg.lucide-loader-circle");
    expect(icon?.getAttribute("class")).toMatch(/animate-spin/);
  });

  it("applies role=status for screen readers", () => {
    render(<ExtractionStatusBadge status="processing" />);
    expect(screen.getByRole("status")).toBeInTheDocument();
  });

  it("uses different color classes per status", () => {
    const { container: pendingBox } = render(
      <ExtractionStatusBadge status="pending" />,
    );
    const { container: failedBox } = render(
      <ExtractionStatusBadge status="failed" />,
    );

    const pendingClasses = pendingBox.firstElementChild?.className ?? "";
    const failedClasses = failedBox.firstElementChild?.className ?? "";
    expect(pendingClasses).not.toBe("");
    expect(failedClasses).not.toBe("");
    expect(pendingClasses).not.toBe(failedClasses);
  });
});
