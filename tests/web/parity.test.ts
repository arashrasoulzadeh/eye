import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { readFileSync } from "fs";
import { join } from "path";
import { EyeClient } from "../../web-sdk/src/index";

// Shared fixture also driven through sdk-go's EyeClient port in
// sdk-go/parity_test.go, asserting both SDKs evict/retain ring-buffer events
// identically for the same inputs.
type FixtureCase = {
  name: string;
  maxBufferSize: number;
  batchSize: number;
  trackNames: string[];
  expectedRemaining: string[];
};

function loadFixture(): { cases: FixtureCase[] } {
  const path = join(__dirname, "..", "fixtures", "sdk_ring_buffer_parity.json");
  return JSON.parse(readFileSync(path, "utf-8"));
}

function makeStorage() {
  const store = new Map<string, string>();
  return {
    getItem: (k: string) => store.get(k) ?? null,
    setItem: (k: string, v: string) => void store.set(k, v),
    removeItem: (k: string) => void store.delete(k),
    clear: () => store.clear(),
  } as unknown as Storage;
}

describe("EyeClient ring buffer parity with sdk-go", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    (globalThis as any).localStorage = makeStorage();
  });

  afterEach(() => {
    vi.useRealTimers();
    delete (globalThis as any).localStorage;
    vi.restoreAllMocks();
  });

  const fixture = loadFixture();
  expect(fixture.cases.length).toBeGreaterThan(0);

  for (const tc of fixture.cases) {
    it(tc.name, () => {
      const c = new EyeClient({
        tenantId: "t1",
        maxBufferSize: tc.maxBufferSize,
        batchSize: tc.batchSize,
        flushIntervalMs: 0,
      });
      (c as any).sessionId = "sess-1";

      for (const name of tc.trackNames) {
        c.track(name);
      }

      const remaining = ((c as any).buffer as { eventName: string }[]).map((e) => e.eventName);
      expect(remaining).toEqual(tc.expectedRemaining);
    });
  }
});
