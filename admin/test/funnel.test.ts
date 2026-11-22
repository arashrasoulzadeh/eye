import { describe, expect, it } from "vitest";
import { funnelTransition } from "@/lib/funnel";

describe("funnelTransition", () => {
  it("measures drop-off and conversion against the preceding stage", () => {
    expect(funnelTransition(40, 10)).toEqual({ dropped: 30, rate: 0.25 });
  });

  it("distinguishes zero conversion from an empty preceding stage", () => {
    expect(funnelTransition(40, 0)).toEqual({ dropped: 40, rate: 0 });
    expect(funnelTransition(0, 0)).toEqual({ dropped: 0, rate: null });
  });
});
