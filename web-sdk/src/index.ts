export type Event = {
  eventId: string;
  eventName: string;
  clientTs: number;
  payload: Record<string, any>;
  dims?: Record<string, string>;
  sessionId: string;
};

export type EyeClientOptions = {
  tenantId: string;
  /** Base URL of the collector API, used by track()-triggered flushes. */
  endpoint?: string;
  apiKey?: string;
  storageKey?: string;
  /** Bounded ring buffer cap; oldest events are dropped once exceeded. */
  maxBufferSize?: number;
  /** Auto-flush once the buffer reaches this many events. 0 disables it. */
  batchSize?: number;
  /** Auto-flush on this interval (ms) regardless of buffer size. 0 disables it. */
  flushIntervalMs?: number;
};

const DEFAULTS = {
  storageKey: "eye_session_id",
  maxBufferSize: 1000,
  batchSize: 20,
  flushIntervalMs: 5000,
};

export class EyeClient {
  private buffer: Event[] = [];
  private readonly maxBufferSize: number;
  private readonly batchSize: number;
  private readonly flushIntervalMs: number;
  private tenantId: string;
  private sessionId: string | null = null;
  private storageKey: string;
  private endpoint?: string;
  private apiKey?: string;
  private timer?: ReturnType<typeof setInterval>;

  constructor(opts: EyeClientOptions) {
    this.tenantId = opts.tenantId;
    this.storageKey = opts.storageKey ?? DEFAULTS.storageKey;
    this.maxBufferSize = opts.maxBufferSize ?? DEFAULTS.maxBufferSize;
    this.batchSize = opts.batchSize ?? DEFAULTS.batchSize;
    this.flushIntervalMs = opts.flushIntervalMs ?? DEFAULTS.flushIntervalMs;
    this.endpoint = opts.endpoint;
    this.apiKey = opts.apiKey;
    this.sessionId = typeof localStorage !== "undefined" ? localStorage.getItem(this.storageKey) : null;

    if (this.flushIntervalMs > 0 && typeof setInterval !== "undefined") {
      this.timer = setInterval(() => {
        this.flush().catch(() => {
          // best-effort background flush; errors surface on the next manual flush
        });
      }, this.flushIntervalMs);
    }
  }

  /** Stops the time-based auto-flush timer. */
  destroy() {
    if (this.timer) clearInterval(this.timer);
  }

  async ensureSession(endpoint: string, apiKey: string) {
    if (this.sessionId) return this.sessionId;
    const res = await fetch(endpoint + "/sessions", {
      method: "POST",
      headers: { "content-type": "application/json", "x-api-key": apiKey },
      body: JSON.stringify({ tenant_id: this.tenantId }),
    });
    const data = await res.json();
    this.sessionId = data.session_id;
    if (typeof localStorage !== "undefined") localStorage.setItem(this.storageKey, this.sessionId!);
    return this.sessionId!;
  }

  track(name: string, props?: Record<string, any>, payload?: Record<string, any>) {
    if (!this.sessionId) throw new Error("call ensureSession first");
    const ev: Event = {
      eventId: crypto.randomUUID(),
      eventName: name,
      clientTs: Date.now(),
      payload: payload ?? {},
      dims: props ? Object.fromEntries(Object.entries(props).map(([k, v]) => [k, String(v)])) : {},
      sessionId: this.sessionId,
    };
    if (this.buffer.length >= this.maxBufferSize) this.buffer.shift();
    this.buffer.push(ev);

    if (this.batchSize > 0 && this.buffer.length >= this.batchSize) {
      this.flush().catch(() => {
        // best-effort background flush; buffered events remain for the next attempt
      });
    }
  }

  async flush(endpointOverride?: string, apiKeyOverride?: string) {
    const endpoint = endpointOverride ?? this.endpoint;
    const apiKey = apiKeyOverride ?? this.apiKey;
    if (!endpoint || !this.buffer.length) return;
    const events = this.buffer.splice(0);
    const body = { events };
    await fetch(endpoint + "/ingest", {
      method: "POST",
      headers: { "content-type": "application/json", "x-api-key": apiKey ?? "" },
      body: JSON.stringify(body),
    });
  }
}
