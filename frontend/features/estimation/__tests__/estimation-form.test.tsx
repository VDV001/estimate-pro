import { describe, it, expect, beforeEach, vi } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { EstimationForm } from "../components/estimation-form";

function makeWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  const Wrapper = ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
  Wrapper.displayName = "TestWrapper";
  return Wrapper;
}

beforeEach(() => {
  localStorage.setItem("ep_access_token", "test-token");
});

describe("EstimationForm", () => {
  it("renders form with task input and PERT headers", () => {
    render(<EstimationForm projectId="p1" />, { wrapper: makeWrapper() });
    expect(screen.getByText("estimation.task")).toBeDefined();
    expect(screen.getByText("estimation.minHours")).toBeDefined();
    expect(screen.getByText("estimation.likelyHours")).toBeDefined();
    expect(screen.getByText("estimation.maxHours")).toBeDefined();
    expect(screen.getByText("estimation.pert")).toBeDefined();
  });

  it("add row button creates new row", async () => {
    const user = userEvent.setup();
    render(<EstimationForm projectId="p1" />, { wrapper: makeWrapper() });

    const addBtn = screen.getByText("estimation.addTask");
    await user.click(addBtn);

    // Should have 2 task name inputs now
    await waitFor(() => {
      const inputs = screen.getAllByPlaceholderText("estimation.taskPlaceholder");
      expect(inputs.length).toBe(2);
    });
  });

  it("remove row button is disabled when only one row", () => {
    render(<EstimationForm projectId="p1" />, { wrapper: makeWrapper() });

    // Find trash button (should be disabled)
    const buttons = screen.getAllByRole("button");
    const trashBtn = buttons.find(
      (btn) => btn.querySelector("svg.lucide-trash2, svg.lucide-trash-2") !== null
    );
    if (trashBtn) {
      expect(trashBtn).toHaveProperty("disabled", true);
    }
  });

  it("PERT calculation displayed when values entered", async () => {
    const user = userEvent.setup();
    render(<EstimationForm projectId="p1" />, { wrapper: makeWrapper() });

    const inputs = screen.getAllByRole("spinbutton");
    // min, likely, max inputs
    await user.type(inputs[0], "2");
    await user.type(inputs[1], "4");
    await user.type(inputs[2], "6");

    // PERT = (2 + 4*4 + 6) / 6 = 24/6 = 4.0
    // Both the row PERT and total show 4.0, use getAllByText
    await waitFor(() => {
      const pertTexts = screen.getAllByText("4.0");
      expect(pertTexts.length).toBeGreaterThanOrEqual(1);
    });
  });

  it("submit button disabled when all task names empty", () => {
    render(<EstimationForm projectId="p1" />, { wrapper: makeWrapper() });

    const submitBtn = screen.getByText("estimation.create");
    expect((submitBtn as HTMLButtonElement).disabled).toBe(true);
  });

  it("submit calls onCreated callback", async () => {
    const user = userEvent.setup();
    const onCreated = vi.fn();

    render(<EstimationForm projectId="p1" onCreated={onCreated} />, {
      wrapper: makeWrapper(),
    });

    // Fill task name
    const taskInput = screen.getByPlaceholderText("estimation.taskPlaceholder");
    await user.type(taskInput, "My Task");

    // Fill hours
    const inputs = screen.getAllByRole("spinbutton");
    await user.type(inputs[0], "1");
    await user.type(inputs[1], "2");
    await user.type(inputs[2], "3");

    // Submit
    const submitBtn = screen.getByText("estimation.create");
    await user.click(submitBtn);

    await waitFor(() => {
      expect(onCreated).toHaveBeenCalled();
    });
  });

  describe("initialTasks pre-fill", () => {
    it("pre-fills one row per task name when initialTasks given", () => {
      render(
        <EstimationForm
          projectId="p1"
          initialTasks={["Implement login", "Wire OAuth", "Add tests"]}
        />,
        { wrapper: makeWrapper() },
      );

      const inputs = screen.getAllByPlaceholderText(
        "estimation.taskPlaceholder",
      ) as HTMLInputElement[];
      expect(inputs).toHaveLength(3);
      expect(inputs[0].value).toBe("Implement login");
      expect(inputs[1].value).toBe("Wire OAuth");
      expect(inputs[2].value).toBe("Add tests");
    });

    it("pre-filled rows have empty hour fields (user enters PERT)", () => {
      render(
        <EstimationForm projectId="p1" initialTasks={["Task A"]} />,
        { wrapper: makeWrapper() },
      );

      const hourInputs = screen.getAllByRole("spinbutton") as HTMLInputElement[];
      // 3 hour inputs per row × 1 row = 3
      expect(hourInputs).toHaveLength(3);
      hourInputs.forEach((input) => expect(input.value).toBe(""));
    });

    it("falls back to one empty row when initialTasks is empty array", () => {
      render(
        <EstimationForm projectId="p1" initialTasks={[]} />,
        { wrapper: makeWrapper() },
      );
      const inputs = screen.getAllByPlaceholderText(
        "estimation.taskPlaceholder",
      ) as HTMLInputElement[];
      expect(inputs).toHaveLength(1);
      expect(inputs[0].value).toBe("");
    });

    it("falls back to one empty row when initialTasks is undefined", () => {
      render(<EstimationForm projectId="p1" />, { wrapper: makeWrapper() });
      const inputs = screen.getAllByPlaceholderText(
        "estimation.taskPlaceholder",
      ) as HTMLInputElement[];
      expect(inputs).toHaveLength(1);
    });
  });
});
