import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, expect, it, vi } from "vitest";
import { PinnedFunnelCard } from "@/components/PinnedFunnelCard";
import { api } from "@/lib/api";

vi.mock("@/lib/api", () => ({
  api: {
    analytics: {
      pinnedFunnel: vi.fn().mockResolvedValue({ name: "Signup journey", steps: ["landing", "signup"] }),
      funnel: vi.fn().mockImplementation((_tenant, _steps, from) =>
        Promise.resolve([{ sessions: 20 }, { sessions: from === 101 ? 10 : 5 }])),
    },
  },
}));
afterEach(cleanup);

it("compares the same pinned sequence in both periods and links to its editor", async () => {
  render(<PinnedFunnelCard tenantId="shop" from={101} to={200} />);
  expect(await screen.findByText("+25")).toBeInTheDocument();
  expect(api.analytics.funnel).toHaveBeenCalledWith("shop", ["landing", "signup"], 101, 200);
  expect(api.analytics.funnel).toHaveBeenCalledWith("shop", ["landing", "signup"], 1, 100);
  expect(screen.getByRole("link", { name: "Open pinned funnel" })).toHaveAttribute("href", "/analytics/funnel?pinned=1");
  expect(screen.getByText("Largest drop-off")).toBeInTheDocument();
});
