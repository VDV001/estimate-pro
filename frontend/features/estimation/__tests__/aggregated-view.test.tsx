// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { http, HttpResponse } from "msw";
import { server } from "../../../__tests__/mocks/server";
import { AggregatedView } from "../components/aggregated-view";
import type { AggregatedResult } from "../api/index";

const API = "http://localhost:8080/api/v1";

function makeWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  const wrapper = ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
  return wrapper;
}

const aggregatedData: AggregatedResult = {
  total_hours: 25.0,
  items: [
    {
      task_name: "Backend API",
      avg_pert_hours: 10.0,
      min_of_mins: 6,
      max_of_maxes: 15,
      estimator_count: 3,
    },
    {
      task_name: "Frontend UI",
      avg_pert_hours: 15.0,
      min_of_mins: 10,
      max_of_maxes: 22,
      estimator_count: 2,
    },
  ],
};

describe("AggregatedView", () => {
  it("renders aggregated data table with task names", async () => {
    server.use(
      http.get(`${API}/projects/:id/estimations/aggregated`, () =>
        HttpResponse.json(aggregatedData)
      )
    );

    render(<AggregatedView projectId="p1" />, { wrapper: makeWrapper() });

    // Each task name appears twice: once in the table row and once in the PERT bar chart
    const backendItems = await screen.findAllByText("Backend API");
    expect(backendItems.length).toBeGreaterThanOrEqual(1);
    const frontendItems = await screen.findAllByText("Frontend UI");
    expect(frontendItems.length).toBeGreaterThanOrEqual(1);
  });

  it("renders column headers", async () => {
    server.use(
      http.get(`${API}/projects/:id/estimations/aggregated`, () =>
        HttpResponse.json(aggregatedData)
      )
    );

    render(<AggregatedView projectId="p1" />, { wrapper: makeWrapper() });

    expect(await screen.findByText("estimation.task")).toBeDefined();
    expect(await screen.findByText("estimation.avgPert")).toBeDefined();
    expect(await screen.findByText("estimation.estimators")).toBeDefined();
  });

  it("renders total hours row", async () => {
    server.use(
      http.get(`${API}/projects/:id/estimations/aggregated`, () =>
        HttpResponse.json(aggregatedData)
      )
    );

    render(<AggregatedView projectId="p1" />, { wrapper: makeWrapper() });

    expect(await screen.findByText("estimation.total")).toBeDefined();
    // total_hours: 25.0 → "25.0"
    expect(await screen.findByText("25.0")).toBeDefined();
  });

  it("renders avg_pert_hours values formatted to one decimal", async () => {
    server.use(
      http.get(`${API}/projects/:id/estimations/aggregated`, () =>
        HttpResponse.json(aggregatedData)
      )
    );

    render(<AggregatedView projectId="p1" />, { wrapper: makeWrapper() });

    expect(await screen.findByText("10.0")).toBeDefined();
    expect(await screen.findByText("15.0")).toBeDefined();
  });

  it("renders estimator counts", async () => {
    server.use(
      http.get(`${API}/projects/:id/estimations/aggregated`, () =>
        HttpResponse.json(aggregatedData)
      )
    );

    render(<AggregatedView projectId="p1" />, { wrapper: makeWrapper() });

    // Wait for data to load first
    await screen.findAllByText("Backend API");

    // estimator_count values: 3 and 2 appear in the table
    const allThrees = screen.getAllByText("3");
    expect(allThrees.length).toBeGreaterThan(0);
    const allTwos = screen.getAllByText("2");
    expect(allTwos.length).toBeGreaterThan(0);
  });

  it("shows empty state when items list is empty", async () => {
    server.use(
      http.get(`${API}/projects/:id/estimations/aggregated`, () =>
        HttpResponse.json({ total_hours: 0, items: [] })
      )
    );

    render(<AggregatedView projectId="p1" />, { wrapper: makeWrapper() });

    expect(await screen.findByText("estimation.noAggregated")).toBeDefined();
  });

  it("shows empty state when items is null", async () => {
    server.use(
      http.get(`${API}/projects/:id/estimations/aggregated`, () =>
        HttpResponse.json({ total_hours: 0, items: null })
      )
    );

    render(<AggregatedView projectId="p1" />, { wrapper: makeWrapper() });

    expect(await screen.findByText("estimation.noAggregated")).toBeDefined();
  });

  it("shows error state when request fails", async () => {
    server.use(
      http.get(`${API}/projects/:id/estimations/aggregated`, () =>
        HttpResponse.json({ message: "Internal error" }, { status: 500 })
      )
    );

    render(<AggregatedView projectId="p1" />, { wrapper: makeWrapper() });

    expect(await screen.findByText("common.error")).toBeDefined();
  });
});
