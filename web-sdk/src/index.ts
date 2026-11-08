export type Event = {
  eventId: string;
  eventName: string;
  clientTs: number;
  payload: Record<string, any>;
  dims?: Record<string, string>;
  sessionId: string;
};

export class EyeClient {
  private buffer: Event[] = [];
  private readonly max = 1000;
  private tenantId: string;
  private sessionId: string | null = null;
  private storageKey: string;

  constructor(tenantId: string, storageKey = "eye_session_id") {
    this.tenantId = tenantId;
    this.storageKey = storageKey;
    this.sessionId = typeof localStorage !== "undefined" ? localStorage.getItem(storageKey) : null;
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
      dims: props ? Object.fromEntries(Object.entries(props).map(([k,v])=>[k,String(v)])) : {},
      sessionId: this.sessionId,
    };
    if (this.buffer.length >= this.max) this.buffer.shift();
    this.buffer.push(ev);
  }

  async flush(endpoint: string, apiKey: string) {
    const events = this.buffer.splice(0);
    if (!events.length) return;
    const body = { events };
    await fetch(endpoint + "/ingest", {
      method: "POST",
      headers: { "content-type": "application/json", "x-api-key": apiKey },
      body: JSON.stringify(body),
    });
  }
}