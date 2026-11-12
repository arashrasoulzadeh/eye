import { describe, expect, it } from "vitest";
import { formatEventLabel } from "@/lib/funnel";

describe("formatEventLabel", () => {
  it("formats event names and applies configured titles", () => {
    expect(formatEventLabel("page_view")).toBe("Page View");
    expect(formatEventLabel("page_view", { page_view: "Viewed page" })).toBe("Viewed page");
  });

  it("does not throw for malformed API values", () => {
    expect(formatEventLabel(undefined)).toBe("Unknown Event");
    expect(formatEventLabel(null)).toBe("Unknown Event");
    expect(formatEventLabel({})).toBe("Unknown Event");
  });
});
