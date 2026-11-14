import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import SessionDetailClient from "./SessionDetailClient";
import { ToastProvider } from "@/components/Toast";

vi.mock("@/components/TenantContext", () => ({ useTenant: () => ({ tenantId: "t1", loading: false }) }));
vi.mock("@/components/TimeRangeContext", () => ({ useTimeRange: () => ({ from: 1000, to: 2000 }) }));

vi.mock("next/navigation", () => ({
  useParams: () => ({ id: "s1" }),
}));

vi.mock("@/lib/api", async () => {
  const actual = await vi.importActual<typeof import("@/lib/api")>("@/lib/api");
  return {
    ...actual,
    api: {
      tenants: { list: vi.fn() },
      sessions: { list: vi.fn(), get: vi.fn() },
      events: { list: vi.fn(), listPage: vi.fn() },
    },
  };
});

import { api } from "@/lib/api";

describe("SessionDetailClient", () => {
  beforeEach(() => {
    vi.mocked(api.tenants.list).mockResolvedValue([{ ID: "t1", Name: "acme", APIKey: "", CreatedAt: "" }]);
    vi.mocked(api.sessions.list).mockResolvedValue([
      {
        ID: "s1",
        TenantID: "t1",
        StartedAt: "2024-01-01T00:00:00Z",
        LastSeen: "2024-01-01T01:00:00Z",
        EventCount: 2,
        UserAgent: "Mozilla/5.0 (test-agent)",
        IP: "203.0.113.7",
        DeviceInfo: '{"platform":"MacIntel","timezone":"Asia/Tehran"}',
      },
      { ID: "s2", TenantID: "t1", StartedAt: "", LastSeen: "", EventCount: 0, UserAgent: "", IP: "", DeviceInfo: "" },
    ]);
    vi.mocked(api.sessions.get).mockResolvedValue({ intelligence_tags: ["gameplay"], ID: "s1", TenantID: "t1", StartedAt: "", LastSeen: "", EventCount: 2, UserAgent: "Mozilla/5.0 (test-agent)", IP: "203.0.113.7", DeviceInfo: '{"platform":"MacIntel","timezone":"Asia/Tehran"}' });
    vi.mocked(api.events.list).mockResolvedValue([
      { EventID: "e1", SessionID: "s1", TenantID: "t1", EventName: "click", ClientTS: Date.now(), IngestTS: "", Payload: "{}", Dims: "{}", Source: "client", IsAggregated: false, AggregationID: "" },
    ]);
    vi.mocked(api.events.listPage).mockResolvedValue({ items: [
      { EventID: "e1", SessionID: "s1", TenantID: "t1", EventName: "click", ClientTS: Date.now(), IngestTS: "", Payload: "{}", Dims: "{}", Source: "client", IsAggregated: false, AggregationID: "" },
    ], total: 1, limit: 10, offset: 0 });
  });

  it("shows the session's tenant (as a link) and its events, scoped to this session only", async () => {
    render(
      <ToastProvider>
        <SessionDetailClient />
      </ToastProvider>
    );
    await waitFor(() => expect(screen.getByText("Session")).toBeInTheDocument());

    const tenantLink = screen.getByRole("link", { name: "acme" });
    expect(tenantLink).toHaveAttribute("href", "/tenants/t1");

    expect(api.sessions.get).toHaveBeenCalledWith("s1", "t1");
    expect(api.events.list).toHaveBeenCalledWith({ session_id: "s1", tenant_id: "t1", from_ts: 1000, to_ts: 2000, limit: 100, offset: 0, sort: "asc" });
    expect(screen.getByText("Events in this session (1)")).toBeInTheDocument();
    expect(screen.getByText("gameplay")).toBeInTheDocument();
    expect(screen.getByText("click")).toBeInTheDocument();

    // Device info: server-captured UA/IP plus the client-supplied fields.
    expect(screen.getByText("203.0.113.7")).toBeInTheDocument();
    expect(screen.getByText("Mozilla/5.0 (test-agent)")).toBeInTheDocument();
    expect(screen.getByText("MacIntel")).toBeInTheDocument();
    expect(screen.getByText("Asia/Tehran")).toBeInTheDocument();
  });

  it("deduplicates repeated event ids before rendering keyed lists", async () => {
    vi.mocked(api.events.list).mockResolvedValue([
      { EventID: "duplicate", SessionID: "s1", TenantID: "t1", EventName: "click", ClientTS: 2, IngestTS: "", Payload: "{}", Dims: "{}", Source: "client", IsAggregated: false, AggregationID: "" },
      { EventID: "duplicate", SessionID: "s1", TenantID: "t1", EventName: "click", ClientTS: 1, IngestTS: "", Payload: "{}", Dims: "{}", Source: "client", IsAggregated: false, AggregationID: "" },
    ]);
    render(<ToastProvider><SessionDetailClient /></ToastProvider>);
    await waitFor(() => expect(screen.getByText("Events in this session (1)")).toBeInTheDocument());
  });

  it("shows a not-found state for an unknown session id", async () => {
    vi.mocked(api.sessions.list).mockResolvedValue([]);
    vi.mocked(api.sessions.get).mockResolvedValue(undefined as never);
    render(
      <ToastProvider>
        <SessionDetailClient />
      </ToastProvider>
    );
    await waitFor(() => expect(screen.getByText("Session not found")).toBeInTheDocument());
  });
});
