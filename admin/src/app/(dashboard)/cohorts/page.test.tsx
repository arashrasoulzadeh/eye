import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import CohortsPage from "./page";
import { ToastProvider } from "@/components/Toast";

vi.mock("@/lib/api", async () => {
  const actual = await vi.importActual<typeof import("@/lib/api")>("@/lib/api");
  return {
    ...actual,
    api: {
      tenants: { list: vi.fn() },
      events: { list: vi.fn() },
      cohorts: { list: vi.fn(), listPage: vi.fn(), create: vi.fn(), materialize: vi.fn() },
    },
  };
});

import { api } from "@/lib/api";

function renderPage() {
  return render(
    <ToastProvider>
      <CohortsPage />
    </ToastProvider>
  );
}

describe("CohortsPage", () => {
  beforeEach(() => {
    vi.mocked(api.tenants.list).mockResolvedValue([{ ID: "t1", Name: "acme", APIKey: "", CreatedAt: "" }]);
    vi.mocked(api.events.list).mockResolvedValue([]);
    vi.mocked(api.cohorts.list).mockResolvedValue([
      { ID: "c1", TenantID: "t1", Name: "power_users", EventName: "purchase", MinCount: 3, CreatedAt: 0, UpdatedAt: 0 },
    ]);
    vi.mocked(api.cohorts.listPage).mockResolvedValue({ items: [{ ID: "c1", TenantID: "t1", Name: "power_users", EventName: "purchase", MinCount: 3, CreatedAt: 0, UpdatedAt: 0 }], total: 1, limit: 10, offset: 0 });
  });

  it("lists existing cohorts for the selected tenant", async () => {
    renderPage();
    await waitFor(() => expect(screen.getByText("power_users")).toBeInTheDocument());
    expect(screen.getByText(/purchase/)).toBeInTheDocument();
    expect(screen.getByText("not materialized")).toBeInTheDocument();
  });

  it("materializes a cohort and shows the resulting member count", async () => {
    vi.mocked(api.cohorts.materialize).mockResolvedValue({ members: 7 });
    renderPage();
    await waitFor(() => expect(screen.getByText("power_users")).toBeInTheDocument());
    fireEvent.click(screen.getByText("Materialize"));
    await waitFor(() => expect(api.cohorts.materialize).toHaveBeenCalledWith("c1"));
    await waitFor(() => expect(screen.getByText("7")).toBeInTheDocument());
  });

  it("creates a new cohort from the form", async () => {
    vi.mocked(api.cohorts.create).mockResolvedValue({ ID: "c2", TenantID: "t1", Name: "new", EventName: "signup", MinCount: 1, CreatedAt: 0, UpdatedAt: 0 });
    renderPage();
    await waitFor(() => expect(screen.getByText("power_users")).toBeInTheDocument());

    fireEvent.change(screen.getByPlaceholderText("power_users"), { target: { value: "new" } });
    fireEvent.change(screen.getByPlaceholderText("purchase"), { target: { value: "signup" } });
    fireEvent.click(screen.getByText("Create cohort"));

    await waitFor(() => expect(api.cohorts.create).toHaveBeenCalledWith("t1", "new", "signup", 1));
  });
});
