import { describe, expect, it, vi } from "vitest";
import { EyeReactNativeClient, Storage } from "./index";

class MemoryStorage implements Storage { values = new Map<string, string>(); async getItem(key: string) { return this.values.get(key) ?? null; } async setItem(key: string, value: string) { this.values.set(key, value); } }

describe("EyeReactNativeClient", () => {
  it("keeps events in AsyncStorage-compatible storage across failed flushes", async () => {
    const storage = new MemoryStorage(); let ingestCalls = 0;
    const fetch = vi.fn(async (input: RequestInfo | URL) => { if (String(input).endsWith("/sessions")) return new Response('{"ID":"s1"}', { status: 201 }); ingestCalls++; return new Response("down", { status: ingestCalls === 1 ? 503 : 202 }); });
    const client = new EyeReactNativeClient({ endpoint: "https://eye.test", apiKey: "key", tenantId: "tenant", storage, fetch });
    await client.ensureSession(); await client.track("purchase"); await expect(client.flush()).rejects.toThrow("HTTP 503"); expect((await client.diagnostics()).queuedEvents).toBe(1); await expect(client.flush()).resolves.toBe(1); expect((await client.diagnostics()).queuedEvents).toBe(0);
  });
});
