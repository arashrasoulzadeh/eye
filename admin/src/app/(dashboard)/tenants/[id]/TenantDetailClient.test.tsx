import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import TenantDetailPage from "./TenantDetailClient";
import { ToastProvider } from "@/components/Toast";

vi.mock("next/navigation", () => ({
  useParams: () => ({ id: "t1" }),
}));

vi.mock("@/lib/api", async () => {
  const actual = await vi.importActual<typeof import("@/lib/api")>("@/lib/api");
  return {
    ...actual,
    api: {
      tenants: { list: vi.fn(), usage: vi.fn(), rotateKey: vi.fn() },
      sessions: { list: vi.fn() },
      events: { list: vi.fn() },
    },
  };
});

import { api } from "@/lib/api";

describe("TenantDetailPage", () => {
  beforeEach(() => {
    vi.mocked(api.tenants.usage).mockRejectedValue(new Error("Usage unavailable"));
    vi.mocked(api.tenants.list).mockResolvedValue([{ ID: "t1", Name: "acme", APIKey: "", CreatedAt: "2024-01-01T00:00:00Z" }]);
    vi.mocked(api.sessions.list).mockResolvedValue({ items: [{ ID: "s1", TenantID: "t1", StartedAt: "", LastSeen: "", EventCount: 2 }], total: 1, limit: 10, offset: 0 });
    vi.mocked(api.events.list).mockResolvedValue([
      { EventID: "e1", SessionID: "s1", TenantID: "t1", EventName: "click", ClientTS: Date.now(), IngestTS: "", Payload: "{}", Dims: "{}", Source: "client", IsAggregated: false, AggregationID: "" },
    ]);
  });

  it("shows the tenant name, session count scoped to this tenant, and a not-editable retention note", async () => {
    render(
      <ToastProvider>
        <TenantDetailPage />
      </ToastProvider>
    );
    await waitFor(() => expect(screen.getByText("acme")).toBeInTheDocument());
    expect(await screen.findByText("Tenant usage is temporarily unavailable.")).toBeInTheDocument();
    expect(api.tenants.list).toHaveBeenCalledWith({ filter: "t1" });
    expect(screen.getByText("Sessions").nextElementSibling).toHaveTextContent("1");
    expect(screen.getByText(/no per-tenant retention setting/i)).toBeInTheDocument();

    // The sessions table lists only this tenant's session (s1, not s2), as a
    // real link to its detail page — not just a count.
    expect(screen.getByText("Sessions (1)")).toBeInTheDocument();
    const sessionLink = screen.getByRole("link", { name: /s1/ });
    expect(sessionLink).toHaveAttribute("href", "/sessions/s1");
    expect(screen.queryByRole("link", { name: /s2/ })).not.toBeInTheDocument();
  });

  it("shows a not-found state for an unknown tenant id", async () => {
    vi.mocked(api.tenants.list).mockResolvedValue([]);
    render(
      <ToastProvider>
        <TenantDetailPage />
      </ToastProvider>
    );
    await waitFor(() => expect(screen.getByText("Tenant not found")).toBeInTheDocument());
  });
});
