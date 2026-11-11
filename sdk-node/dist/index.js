import { mkdir, readFile, rename, writeFile } from "node:fs/promises";
import { dirname } from "node:path";
import { randomUUID } from "node:crypto";
const emptyState = () => ({ sessionId: null, events: [], droppedEvents: 0, lastFlushAt: null, lastError: null });
export class EyeNodeClient {
    opts;
    state = emptyState();
    loaded = false;
    lock = Promise.resolve();
    constructor(options) {
        this.opts = { endpoint: options.endpoint.replace(/\/$/, ""), apiKey: options.apiKey, tenantId: options.tenantId, queueFile: options.queueFile ?? ".eye-queue.json", maxQueueSize: options.maxQueueSize ?? 10000, fetch: options.fetch ?? globalThis.fetch };
    }
    async withLock(work) {
        const previous = this.lock;
        let release;
        this.lock = new Promise((resolve) => { release = resolve; });
        await previous;
        try {
            return await work();
        }
        finally {
            release();
        }
    }
    async load() {
        if (this.loaded)
            return;
        try {
            this.state = JSON.parse(await readFile(this.opts.queueFile, "utf8"));
        }
        catch (error) {
            if (error.code !== "ENOENT")
                throw error;
        }
        this.loaded = true;
    }
    async persist() {
        await mkdir(dirname(this.opts.queueFile), { recursive: true });
        const temp = `${this.opts.queueFile}.${process.pid}.tmp`;
        await writeFile(temp, JSON.stringify(this.state), "utf8");
        await rename(temp, this.opts.queueFile);
    }
    async ensureSession() {
        return this.withLock(async () => {
            await this.load();
            if (this.state.sessionId)
                return this.state.sessionId;
            const response = await this.opts.fetch(`${this.opts.endpoint}/sessions`, { method: "POST", headers: { "content-type": "application/json", "x-api-key": this.opts.apiKey }, body: JSON.stringify({ tenant_id: this.opts.tenantId, device: { runtime: "node" } }) });
            if (!response.ok)
                throw new Error(`session creation failed: HTTP ${response.status}`);
            const body = await response.json();
            const id = body.session_id ?? body.ID ?? body.id;
            if (!id)
                throw new Error("session creation returned no id");
            this.state.sessionId = id;
            await this.persist();
            return id;
        });
    }
    async track(eventName, payload = {}, dims = {}) {
        if (!eventName.trim())
            throw new Error("event name is required");
        await this.withLock(async () => { await this.load(); if (!this.state.sessionId)
            throw new Error("call ensureSession first"); this.state.events.push({ event_id: randomUUID().replaceAll("-", ""), session_id: this.state.sessionId, event_name: eventName, client_ts: Date.now(), payload, dims }); if (this.state.events.length > this.opts.maxQueueSize) {
            this.state.events.splice(0, this.state.events.length - this.opts.maxQueueSize);
            this.state.droppedEvents++;
        } await this.persist(); });
    }
    async flush() {
        return this.withLock(async () => { await this.load(); if (!this.state.events.length)
            return 0; const events = [...this.state.events]; try {
            const response = await this.opts.fetch(`${this.opts.endpoint}/ingest`, { method: "POST", headers: { "content-type": "application/json", "x-api-key": this.opts.apiKey }, body: JSON.stringify({ events }) });
            if (!response.ok)
                throw new Error(`ingest failed: HTTP ${response.status}`);
            this.state.events.splice(0, events.length);
            this.state.lastFlushAt = Date.now();
            this.state.lastError = null;
            await this.persist();
            return events.length;
        }
        catch (error) {
            this.state.lastError = error instanceof Error ? error.message : String(error);
            await this.persist();
            throw error;
        } });
    }
    async diagnostics() { return this.withLock(async () => { await this.load(); return { tenantId: this.opts.tenantId, sessionId: this.state.sessionId, queuedEvents: this.state.events.length, droppedEvents: this.state.droppedEvents, lastFlushAt: this.state.lastFlushAt, lastError: this.state.lastError }; }); }
}
export default EyeNodeClient;
