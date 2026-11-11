import { mkdtemp, readFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { describe, expect, it, vi } from "vitest";
import { EyeNodeClient } from "./index.js";
describe("EyeNodeClient", () => {
    it("persists queued events when ingestion fails and flushes them later", async () => {
        const dir = await mkdtemp(join(tmpdir(), "eye-node-"));
        let ingestAttempts = 0;
        const fetch = vi.fn(async (input) => {
            const url = String(input);
            if (url.endsWith("/sessions"))
                return new Response(JSON.stringify({ ID: "session-1" }), { status: 201 });
            ingestAttempts++;
            return new Response("unavailable", { status: ingestAttempts === 1 ? 503 : 202 });
        });
        const client = new EyeNodeClient({ endpoint: "https://eye.test", apiKey: "key", tenantId: "tenant", queueFile: join(dir, "queue.json"), fetch });
        await client.ensureSession();
        await client.track("purchase", { amount: 10 });
        await expect(client.flush()).rejects.toThrow("HTTP 503");
        expect((await client.diagnostics()).queuedEvents).toBe(1);
        await expect(client.flush()).resolves.toBe(1);
        expect((await client.diagnostics()).queuedEvents).toBe(0);
        expect(JSON.parse(await readFile(join(dir, "queue.json"), "utf8")).sessionId).toBe("session-1");
    });
    it("bounds the durable queue and reports dropped events", async () => {
        const dir = await mkdtemp(join(tmpdir(), "eye-node-"));
        const fetch = vi.fn(async (_input) => new Response(JSON.stringify({ ID: "s" }), { status: 201 }));
        const client = new EyeNodeClient({ endpoint: "https://eye.test", apiKey: "key", tenantId: "tenant", queueFile: join(dir, "queue.json"), maxQueueSize: 2, fetch });
        await client.ensureSession();
        await client.track("a");
        await client.track("b");
        await client.track("c");
        await expect(client.diagnostics()).resolves.toMatchObject({ queuedEvents: 2, droppedEvents: 1 });
    });
});
