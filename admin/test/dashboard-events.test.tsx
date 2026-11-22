import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, expect, it, vi } from "vitest";
import { DashboardEvents } from "@/components/DashboardEvents";
import { api } from "@/lib/api";

vi.mock("@/lib/api", () => ({ api: { eventOptions: vi.fn() } }));
afterEach(cleanup);

it("loads aggregated counts for the selected tenant and time range", async () => {
  vi.mocked(api.eventOptions).mockResolvedValue([{ value: "purchase", count: 27, last_ts: 200 }]);
  render(<DashboardEvents tenantId="shop" from={100} to={200} />);
  expect(await screen.findByText("purchase")).toBeInTheDocument();
  expect(screen.getByText("27")).toBeInTheDocument();
  expect(api.eventOptions).toHaveBeenCalledWith("shop", "", false, { from_ts: 100, to_ts: 200 });
});

it("distinguishes an unavailable breakdown from an empty range", async () => {
  vi.mocked(api.eventOptions).mockRejectedValueOnce(new Error("offline"));
  const { unmount } = render(<DashboardEvents tenantId="shop" from={100} to={200} />);
  expect(await screen.findByRole("alert")).toHaveTextContent("Could not load event counts.");
  unmount();
  vi.mocked(api.eventOptions).mockResolvedValueOnce([]);
  render(<DashboardEvents tenantId="shop" from={100} to={200} />);
  expect(await screen.findByText("No events match the current filters.")).toBeInTheDocument();
});

it("links to the event explorer with the exact dashboard scope", async () => {
  vi.mocked(api.eventOptions).mockResolvedValueOnce([{ value: "buy & confirm", count: 27, last_ts: 200 }]);
  render(<DashboardEvents tenantId="shop/1" from={100} to={200} />);
  const link = await screen.findByRole("link", { name: "buy & confirm" });
  const url = new URL(link.getAttribute("href")!, "http://localhost");
  expect(url.pathname).toBe("/events");
  expect(Object.fromEntries(url.searchParams)).toEqual({
    event_name: "buy & confirm", tenant_id: "shop/1", from_ts: "100", to_ts: "200",
  });
});
