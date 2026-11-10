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

function browserStorage(): Storage | undefined {
  if (
    typeof localStorage === "undefined" ||
    typeof localStorage.getItem !== "function" ||
    typeof localStorage.setItem !== "function" ||
    typeof localStorage.removeItem !== "function"
  ) {
    return undefined;
  }
  return localStorage;
}

/**
 * Gathers whatever client-side device/environment info the server genuinely
 * can't derive from the request itself (User-Agent and IP are captured
 * server-side instead — see SessionsHandler). Every field is read behind a
 * typeof guard since this also runs under Vitest/Node, where `navigator`/
 * `screen`/`Intl` may be absent or incomplete; a field that can't be read is
 * simply omitted rather than throwing.
 */
function collectDeviceInfo(): Record<string, any> {
  const info: Record<string, any> = {};
  try {
    if (typeof navigator !== "undefined") {
      if (navigator.platform) info.platform = navigator.platform;
      if (navigator.language) info.language = navigator.language;
      if ((navigator as any).languages) info.languages = (navigator as any).languages;
      if ((navigator as any).userAgentData?.platform) info.platform = (navigator as any).userAgentData.platform;
    }
    if (typeof screen !== "undefined") {
      if (screen.width) info.screenWidth = screen.width;
      if (screen.height) info.screenHeight = screen.height;
    }
    if (typeof window !== "undefined") {
      if (window.innerWidth) info.viewportWidth = window.innerWidth;
      if (window.innerHeight) info.viewportHeight = window.innerHeight;
      if ((window as any).devicePixelRatio) info.devicePixelRatio = (window as any).devicePixelRatio;
    }
    if (typeof Intl !== "undefined") {
      const tz = Intl.DateTimeFormat().resolvedOptions().timeZone;
      if (tz) info.timezone = tz;
    }
  } catch {
    // best-effort only; a partial or empty object is fine
  }
  return info;
}

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
    this.sessionId = browserStorage()?.getItem(this.storageKey) ?? null;
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

  /** The current session id, or null before ensureSession() has resolved. */
  getSessionId(): string | null {
    return this.sessionId;
  }

  /**
   * Creates (or reuses) a session. Pass `{ forceNew: true }` to discard any
   * cached session id — including the one persisted in localStorage — and
   * always create a fresh one (e.g. a "start a new session" test control).
   */
  async ensureSession(endpoint: string, apiKey: string, opts?: { forceNew?: boolean }) {
    if (opts?.forceNew) {
      this.sessionId = null;
      browserStorage()?.removeItem(this.storageKey);
    }
    if (this.sessionId) return this.sessionId;
    const res = await fetch(endpoint + "/sessions", {
      method: "POST",
      headers: { "content-type": "application/json", "x-api-key": apiKey },
      body: JSON.stringify({ tenant_id: this.tenantId, device: collectDeviceInfo() }),
    });
    const data = await res.json();
    // The real backend's SessionsHandler returns the raw model.Session
    // (capitalized `ID`, no json tags, since the admin UI also depends on
    // that exact shape). Accept both so this also keeps working against
    // any future/mocked `{ session_id }` response shape.
    this.sessionId = data.session_id ?? data.ID ?? data.id;
    browserStorage()?.setItem(this.storageKey, this.sessionId!);
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
