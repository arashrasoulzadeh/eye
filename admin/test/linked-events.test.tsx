import { cleanup, render, waitFor } from "@testing-library/react";
import { afterEach, expect, it, vi } from "vitest";
import EventsPage from "@/app/(dashboard)/events/page";
import { ToastProvider } from "@/components/Toast";
import { api } from "@/lib/api";

vi.mock("next/navigation", () => ({
  useSearchParams: () => new URLSearchParams("event_name=checkout&tenant_id=&from_ts=100&to_ts=200"),
}));
vi.mock("@/components/TenantContext", () => ({
  useTenant: () => ({ tenantId: "other-shop", loading: false }),
}));
vi.mock("@/components/TimeRangeContext", () => ({
  useTimeRange: () => ({ from: 300, to: 400, hours: "24" }),
  formatTimeRange: () => "Last 24 hours",
}));
vi.mock("@/components/DataTable", () => ({ DataTable: () => null }));
vi.mock("@/lib/api", () => ({
  api: {
    tenants: { list: vi.fn().mockResolvedValue([]) },
    sessions: { list: vi.fn().mockResolvedValue({ items: [] }) },
    events: { listPage: vi.fn().mockResolvedValue({ items: [], total: 0 }) },
    eventOptions: vi.fn().mockResolvedValue([]),
  },
}));
afterEach(cleanup);

it("queries the exact linked event and range even with another global selection", async () => {
  render(<ToastProvider><EventsPage /></ToastProvider>);
  await waitFor(() => expect(api.events.listPage).toHaveBeenCalledWith(expect.objectContaining({
    event_name: "checkout", tenant_id: "", from_ts: 100, to_ts: 200,
  })));
});
