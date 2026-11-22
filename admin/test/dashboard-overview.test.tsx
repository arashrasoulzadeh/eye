import { cleanup, render, screen, within } from "@testing-library/react";
import { afterEach, expect, it, vi } from "vitest";
import { LanguageProvider } from "@/components/LanguageContext";
import { formatTimeRange } from "@/components/TimeRangeContext";
import { api } from "@/lib/api";
import Dashboard from "@/app/(dashboard)/dashboard/page";

vi.mock("@/components/TenantContext", () => ({
  useTenant: () => ({ tenantId: "shop-1", tenants: [{ ID: "shop-1", Name: "Storefront" }], loading: false }),
}));
vi.mock("@/components/TimeRangeContext", async (original) => ({
  ...await original<typeof import("@/components/TimeRangeContext")>(),
  useTimeRange: () => ({ from: 101, to: 200, hours: "24" }),
}));
vi.mock("@/components/Chart", () => ({ LineTrend: () => null, BarBreakdown: () => null }));
vi.mock("@/lib/api", () => ({
  api: {
    dashboardCache: vi.fn().mockResolvedValue({
      events: 125, sessions: 30, tenants: 1, events_by_hour: [], events_by_tenant: [], cached_at: 200,
    }),
    analytics: { pinnedFunnel: vi.fn().mockResolvedValue(null) },
    eventOptions: vi.fn().mockResolvedValue([]),
  },
}));
afterEach(() => {
  cleanup();
  window.localStorage.clear();
});

it("puts the selected scope and core counts up front and collapses technical metrics", async () => {
  render(<Dashboard />);
  const header = screen.getByRole("heading", { name: "Dashboard" }).closest("header")!;
  expect(within(header).getByText("Storefront")).toBeInTheDocument();
  expect(within(header).getByText("Last 24 hours")).toBeInTheDocument();
  const overview = screen.getByRole("region", { name: "Activity overview" });
  expect(await within(overview).findByText("125")).toBeInTheDocument();
  expect(within(overview).getByText("30")).toBeInTheDocument();
  const technical = screen.getByText("Technical metrics").closest("details");
  expect(technical).not.toHaveAttribute("open");
  expect(within(technical!).getByText("Cache age")).toBeInTheDocument();
  expect(within(technical!).getByText("Events / second")).toBeInTheDocument();
});

it("renders the persisted Persian scope and supports custom durations", async () => {
  window.localStorage.setItem("eye.admin.language", "fa");
  render(<LanguageProvider><Dashboard /></LanguageProvider>);
  expect(screen.getByRole("heading", { name: "داشبورد" })).toBeInTheDocument();
  expect(screen.getByText("۲۴ ساعت اخیر")).toBeInTheDocument();
  expect(formatTimeRange("48", "fa")).toBe("۲ روز اخیر");
  expect(await screen.findByText("۱۲۵")).toBeInTheDocument();
});

it("shows event and session changes against the adjacent period", async () => {
  vi.mocked(api.dashboardCache).mockResolvedValueOnce({
    events: 125, sessions: 30, tenants: 1, events_by_hour: [], events_by_tenant: [], cached_at: 200,
  }).mockResolvedValueOnce({
    events: 100, sessions: 40, tenants: 1, events_by_hour: [], events_by_tenant: [], cached_at: 100,
  });
  render(<Dashboard />);
  expect(await screen.findByText("Change vs previous period +25%")).toBeInTheDocument();
  expect(screen.getByText("Change vs previous period -25%")).toBeInTheDocument();
  expect(api.dashboardCache).toHaveBeenCalledWith({ tenant_id: "shop-1", from_ts: 1, to_ts: 100 });
});
