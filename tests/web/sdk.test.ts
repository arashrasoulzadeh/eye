import { describe, it, expect } from "vitest";
import { EyeClient } from "../../web-sdk/src/index";

describe("EyeClient", () => {
  it("buffers events", () => {
    const c = new EyeClient("t1", {} as any);
    (c as any).sessionId = "s1";
    c.track("page_view", { page: "/home" });
    expect((c as any).buffer.length).toBe(1);
  });
});
