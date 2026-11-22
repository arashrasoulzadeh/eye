import { expect, it } from "vitest";
import { previousPeriod, funnelSummary } from "@/lib/funnel";

it("uses an equal-length immediately preceding window with inclusive API bounds", () => {
  expect(previousPeriod(101, 200)).toEqual({ from_ts: 1, to_ts: 100 });
});

it("distinguishes no entrants from zero completion", () => {
  expect(funnelSummary([{ sessions: 0 }, { sessions: 0 }]).rate).toBeNull();
  expect(funnelSummary([{ sessions: 10 }, { sessions: 0 }]).rate).toBe(0);
});

it("identifies the greatest proportional loss", () => {
  expect(funnelSummary([{ sessions: 100 }, { sessions: 20 }, { sessions: 1 }]))
    .toMatchObject({ entered: 100, rate: 0.01, biggestDrop: { index: 1, dropped: 19, rate: 0.05 } });
});
