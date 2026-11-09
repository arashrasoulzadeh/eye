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
  /**
   * When true: logs verbose console info for track/flush/session lifecycle,
   * and validates each tracked event's shape locally (missing event name,
   * non-JSON-serializable payload) instead of silently sending it.
   */
  debug?: boolean;
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
  private debug: boolean;

  constructor(opts: EyeClientOptions) {
    this.tenantId = opts.tenantId;
    this.storageKey = opts.storageKey ?? DEFAULTS.storageKey;
    this.maxBufferSize = opts.maxBufferSize ?? DEFAULTS.maxBufferSize;
    this.batchSize = opts.batchSize ?? DEFAULTS.batchSize;
    this.flushIntervalMs = opts.flushIntervalMs ?? DEFAULTS.flushIntervalMs;
    this.endpoint = opts.endpoint;
    this.apiKey = opts.apiKey;
    this.debug = opts.debug ?? false;
    this.sessionId = typeof localStorage !== "undefined" ? localStorage.getItem(this.storageKey) : null;
    if (this.debug) console.info("[eye] client created", { tenantId: this.tenantId, sessionId: this.sessionId });

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
    // The real backend's SessionsHandler returns the raw model.Session
    // (capitalized `ID`, no json tags, since the admin UI also depends on
    // that exact shape). Accept both so this also keeps working against
    // any future/mocked `{ session_id }` response shape.
    this.sessionId = data.session_id ?? data.ID ?? data.id;
    if (typeof localStorage !== "undefined") localStorage.setItem(this.storageKey, this.sessionId!);
    if (this.debug) console.info("[eye] session established", this.sessionId);
    return this.sessionId!;
  }

  /**
   * Validates an event's shape before it's queued. Returns a list of
   * problems (empty means valid) rather than throwing, so callers in debug
   * mode can decide whether a warning or a hard failure is appropriate.
   */
  private validateEvent(name: string, payload?: Record<string, any>): string[] {
    const problems: string[] = [];
    if (!name || typeof name !== "string") problems.push("event name is missing or not a string");
    if (payload !== undefined) {
      try {
        JSON.stringify(payload);
      } catch {
        problems.push("payload is not JSON-serializable");
      }
    }
    return problems;
  }

  track(name: string, props?: Record<string, any>, payload?: Record<string, any>) {
    if (!this.sessionId) throw new Error("call ensureSession first");
    if (this.debug) {
      const problems = this.validateEvent(name, payload);
      if (problems.length) {
        console.warn("[eye] invalid event", { name, problems });
        throw new Error("[eye] invalid event: " + problems.join("; "));
      }
      console.info("[eye] track", { name, props, payload });
    }
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
    if (!endpoint || !this.buffer.length) {
      if (this.debug) console.info("[eye] flush skipped", { endpoint, bufferLength: this.buffer.length });
      return;
    }
    const events = this.buffer.splice(0);
    // The wire format the real /ingest endpoint expects is snake_case
    // (session_id, event_id, event_name, client_ts, payload, dims) — a
    // real, previously-unexercised mismatch with this class's internal
    // camelCase Event shape, caught by actually running the CDN build
    // against the live backend rather than against mocked fetch() tests.
    const body = {
      events: events.map((e) => ({
        session_id: e.sessionId,
        event_id: e.eventId,
        event_name: e.eventName,
        client_ts: e.clientTs,
        payload: e.payload,
        dims: e.dims,
      })),
    };
    if (this.debug) console.info("[eye] flushing", events.length, "events");
    await fetch(endpoint + "/ingest", {
      method: "POST",
      headers: { "content-type": "application/json", "x-api-key": apiKey ?? "" },
      body: JSON.stringify(body),
    });
    if (this.debug) console.info("[eye] flush complete");
  }
}
