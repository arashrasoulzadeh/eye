import { describe, expect, it } from "vitest";
import { formatTimeRange, parseTimeRange } from "./TimeRangeContext";

describe("global time range queries", () => {
  it("parses minute, hour, day, week, and month expressions", () => {
    expect(parseTimeRange("last 4 minutes")).toBeCloseTo(4 / 60);
    expect(parseTimeRange("last 2h")).toBe(2);
    expect(parseTimeRange("last 7 days")).toBe(168);
    expect(parseTimeRange("last week")).toBe(168);
    expect(parseTimeRange("last month")).toBe(720);
  });

  it("parses now-minus expressions and rejects invalid input", () => {
    expect(parseTimeRange("now-30m")).toBeCloseTo(0.5);
    expect(parseTimeRange("now - 3 hours")).toBe(3);
    expect(parseTimeRange("yesterday")).toBeNull();
    expect(parseTimeRange("last 0 bananas")).toBeNull();
  });

  it("formats custom durations for the global selector", () => {
    expect(formatTimeRange(String(4 / 60))).toBe("Last 4 minutes");
    expect(formatTimeRange("720")).toBe("Last 30 days");
  });
});
