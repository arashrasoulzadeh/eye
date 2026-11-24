import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, expect, it, vi } from "vitest";
import FunnelPage from "@/app/(dashboard)/analytics/funnel/page";
import { api } from "@/lib/api";

const scope = vi.hoisted(() => ({ from: 100, to: 200, hours: "24" }));
const query = vi.hoisted(() => ({ pinned: false }));
vi.mock("next/navigation", () => ({
  useSearchParams: () => new URLSearchParams(query.pinned ? "pinned=1" : ""),
}));
vi.mock("@/components/TimeRangeContext", () => ({
  useTimeRange: () => scope,
  formatTimeRange: () => "Last 24 hours",
}));
vi.mock("@/components/TenantContext", () => ({
  useTenant: () => ({ tenantId: "shop", tenants: [], loading: false }),
}));
vi.mock("@/lib/api", () => ({
  api: {
    tenants: { list: vi.fn().mockResolvedValue([]) },
    eventOptions: vi.fn().mockResolvedValue([]),
    analytics: {
      pinnedFunnel: vi.fn().mockResolvedValue(null),
      pinFunnel: vi.fn(async (_tenant, definition) => definition),
      funnel: vi.fn().mockResolvedValue([]),
      suggestedFunnel: vi.fn().mockResolvedValue([
        { event_name: "view", title: "View" },
        { event_name: "purchase", title: "Purchase" },
      ]),
    },
  },
}));
afterEach(() => {
  cleanup();
  vi.clearAllMocks();
  query.pinned = false;
  scope.from = 100;
  scope.to = 200;
});

it("refreshes results after the clock ticks using the reordered stages", async () => {
  const { rerender } = render(<FunnelPage />);
  fireEvent.click(screen.getByRole("button", { name: "Use smart suggestion" }));
  const move = await screen.findByRole("button", { name: "Move stage earlier Purchase" });
  await waitFor(() => expect(move).toBeEnabled());
  fireEvent.click(move);
  const run = screen.getByRole("button", { name: "Run funnel" });
  await waitFor(() => expect(run).toBeEnabled());
  fireEvent.click(run);
  await waitFor(() => expect(api.analytics.funnel).toHaveBeenLastCalledWith("shop", ["purchase", "view"], 100, 200));
  scope.from = 160;
  scope.to = 260;
  rerender(<FunnelPage />);
  await waitFor(() => expect(api.analytics.funnel).toHaveBeenLastCalledWith("shop", ["purchase", "view"], 160, 260));
  expect(api.analytics.suggestedFunnel).toHaveBeenCalledTimes(1);
});

it("uses the first stage as the bar scale and highlights the largest relative loss", async () => {
  vi.mocked(api.analytics.suggestedFunnel).mockResolvedValueOnce([
    { event_name: "view", title: "View", position: 1, sessions: 100 },
    { event_name: "purchase", title: "Purchase", position: 2, sessions: 20 },
    { event_name: "confirm", title: "Confirm", position: 3, sessions: 1 },
  ]);
  vi.mocked(api.analytics.funnel).mockResolvedValueOnce([
    { event_name: "view", sessions: 100, conversion_rate: 1 },
    { event_name: "purchase", sessions: 20, conversion_rate: 0.2 },
    { event_name: "confirm", sessions: 1, conversion_rate: 0.05 },
  ]);
  render(<FunnelPage />);
  fireEvent.click(screen.getByRole("button", { name: "Use smart suggestion" }));

  const bars = await screen.findAllByRole("meter");
  expect(bars.map((bar) => bar.getAttribute("aria-valuenow"))).toEqual(["100", "20", "1"]);
  expect(bars[2].firstElementChild).toHaveStyle({ width: "1%" });

  const highlight = screen.getByText("Largest drop-off").closest("li");
  expect(highlight).toHaveTextContent("Purchase");
  expect(highlight).toHaveTextContent("95%");
  expect(highlight).toHaveTextContent("Dropped sessions 19");
});

it("pins the edited name and ordered steps to the selected tenant", async () => {
  render(<FunnelPage />);
  fireEvent.click(screen.getByRole("button", { name: "Use smart suggestion" }));
  fireEvent.click(await screen.findByRole("button", { name: "Move stage earlier Purchase" }));
  fireEvent.change(screen.getByLabelText("Funnel name"), { target: { value: "Purchase journey" } });
  fireEvent.click(screen.getByRole("button", { name: "Pin to dashboard" }));
  expect(await screen.findByRole("status")).toHaveTextContent("Funnel pinned to dashboard.");
  expect(api.analytics.pinFunnel).toHaveBeenCalledWith("shop", { name: "Purchase journey", steps: ["purchase", "view"] });
});

it("opens the saved definition from the dashboard link", async () => {
  query.pinned = true;
  vi.mocked(api.analytics.pinnedFunnel).mockResolvedValueOnce({ name: "Signup", steps: ["landing", "signup"] });
  render(<FunnelPage />);
  await waitFor(() => expect(api.analytics.funnel).toHaveBeenLastCalledWith("shop", ["landing", "signup"], 100, 200));
  expect(screen.getByLabelText("Funnel name")).toHaveValue("Signup");
});
