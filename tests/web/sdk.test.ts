import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { EyeClient } from "../../web-sdk/src/index";

function makeStorage() {
  const store = new Map<string, string>();
  return {
    getItem: (k: string) => store.get(k) ?? null,
    setItem: (k: string, v: string) => void store.set(k, v),
    removeItem: (k: string) => void store.delete(k),
    clear: () => store.clear(),
  } as unknown as Storage;
}

describe("EyeClient", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    (globalThis as any).localStorage = makeStorage();
  });

  afterEach(() => {
    vi.useRealTimers();
    delete (globalThis as any).localStorage;
    vi.restoreAllMocks();
  });

  it("buffers events", () => {
    const c = new EyeClient({ tenantId: "t1", flushIntervalMs: 0 });
    (c as any).sessionId = "s1";
    c.track("page_view", { page: "/home" });
    expect((c as any).buffer.length).toBe(1);
  });

  it("drops the oldest event once the ring buffer exceeds its cap", () => {
    const c = new EyeClient({ tenantId: "t1", maxBufferSize: 3, batchSize: 1000, flushIntervalMs: 0 });
    (c as any).sessionId = "s1";
    c.track("e1");
    c.track("e2");
    c.track("e3");
    c.track("e4");
    const buffer = (c as any).buffer as { eventName: string }[];
    expect(buffer.length).toBe(3);
    expect(buffer.map((e) => e.eventName)).toEqual(["e2", "e3", "e4"]);
  });

  it("reuses the persisted session id from storage instead of creating a new one", async () => {
    (globalThis as any).localStorage.setItem("eye_session_id", "existing-session");
    const fetchMock = vi.fn();
    (globalThis as any).fetch = fetchMock;

    const c = new EyeClient({ tenantId: "t1", flushIntervalMs: 0 });
    const id = await c.ensureSession("http://api", "key");

    expect(id).toBe("existing-session");
    expect(fetchMock).not.toHaveBeenCalled();
  });

  it("creates and persists a new session id when none is stored", async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      json: async () => ({ session_id: "new-session" }),
    });
    (globalThis as any).fetch = fetchMock;

    const c = new EyeClient({ tenantId: "t1", flushIntervalMs: 0 });
    const id = await c.ensureSession("http://api", "key");

    expect(id).toBe("new-session");
    expect(fetchMock).toHaveBeenCalledWith(
      "http://api/sessions",
      expect.objectContaining({ method: "POST" })
    );
    expect((globalThis as any).localStorage.getItem("eye_session_id")).toBe("new-session");
  });

  it("accepts the real backend's response shape ({ ID }, not { session_id })", async () => {
    // SessionsHandler returns the raw model.Session (capitalized ID, no
    // json tags) — this was a real contract mismatch the CDN build
    // surfaced: ensureSession() must handle this shape, not just a
    // hypothetical { session_id } one.
    const fetchMock = vi.fn().mockResolvedValue({
      json: async () => ({ ID: "real-backend-session", TenantID: "t1" }),
    });
    (globalThis as any).fetch = fetchMock;

    const c = new EyeClient({ tenantId: "t1", flushIntervalMs: 0 });
    const id = await c.ensureSession("http://api", "key");

    expect(id).toBe("real-backend-session");
    expect((globalThis as any).localStorage.getItem("eye_session_id")).toBe("real-backend-session");
  });

  it("flush sends buffered events and empties the buffer", async () => {
    const fetchMock = vi.fn().mockResolvedValue({ json: async () => ({}) });
    (globalThis as any).fetch = fetchMock;

    const c = new EyeClient({ tenantId: "t1", endpoint: "http://api", apiKey: "key", flushIntervalMs: 0 });
    (c as any).sessionId = "s1";
    c.track("e1");
    c.track("e2");

    await c.flush();

    expect(fetchMock).toHaveBeenCalledTimes(1);
    const [url, init] = fetchMock.mock.calls[0];
    expect(url).toBe("http://api/ingest");
    const body = JSON.parse(init.body);
    expect(body.events.length).toBe(2);
    expect((c as any).buffer.length).toBe(0);
  });

  it("auto-flushes once the buffer reaches the batch size threshold", () => {
    const fetchMock = vi.fn().mockResolvedValue({ json: async () => ({}) });
    (globalThis as any).fetch = fetchMock;

    const c = new EyeClient({ tenantId: "t1", endpoint: "http://api", apiKey: "key", batchSize: 3, flushIntervalMs: 0 });
    (c as any).sessionId = "s1";
    c.track("e1");
    c.track("e2");
    expect(fetchMock).not.toHaveBeenCalled();
    c.track("e3");
    expect(fetchMock).toHaveBeenCalledTimes(1);
  });

  it("auto-flushes on a time interval even under the batch size", () => {
    const fetchMock = vi.fn().mockResolvedValue({ json: async () => ({}) });
    (globalThis as any).fetch = fetchMock;

    const c = new EyeClient({ tenantId: "t1", endpoint: "http://api", apiKey: "key", batchSize: 1000, flushIntervalMs: 5000 });
    (c as any).sessionId = "s1";
    c.track("e1");
    expect(fetchMock).not.toHaveBeenCalled();

    vi.advanceTimersByTime(5000);

    expect(fetchMock).toHaveBeenCalledTimes(1);
    c.destroy();
  });

  describe("debug mode", () => {
    it("logs verbose info for track and flush when debug is true", async () => {
      const fetchMock = vi.fn().mockResolvedValue({ json: async () => ({}) });
      (globalThis as any).fetch = fetchMock;
      const infoSpy = vi.spyOn(console, "info").mockImplementation(() => {});

      const c = new EyeClient({ tenantId: "t1", endpoint: "http://api", apiKey: "key", flushIntervalMs: 0, debug: true });
      (c as any).sessionId = "s1";
      c.track("page_view", { page: "/home" });
      await c.flush();

      expect(infoSpy).toHaveBeenCalledWith("[eye] track", expect.objectContaining({ name: "page_view" }));
      expect(infoSpy).toHaveBeenCalledWith("[eye] flushing", 1, "events");
    });

    it("does not log when debug is false (default)", async () => {
      const fetchMock = vi.fn().mockResolvedValue({ json: async () => ({}) });
      (globalThis as any).fetch = fetchMock;
      const infoSpy = vi.spyOn(console, "info").mockImplementation(() => {});

      const c = new EyeClient({ tenantId: "t1", endpoint: "http://api", apiKey: "key", flushIntervalMs: 0 });
      (c as any).sessionId = "s1";
      c.track("page_view");
      await c.flush();

      expect(infoSpy).not.toHaveBeenCalled();
    });

    it("throws on a missing event name when debug is true", () => {
      const c = new EyeClient({ tenantId: "t1", flushIntervalMs: 0, debug: true });
      (c as any).sessionId = "s1";
      expect(() => c.track("")).toThrow(/invalid event/);
    });

    it("throws on a non-serializable payload when debug is true", () => {
      const c = new EyeClient({ tenantId: "t1", flushIntervalMs: 0, debug: true });
      (c as any).sessionId = "s1";
      const circular: Record<string, any> = {};
      circular.self = circular;
      expect(() => c.track("click", undefined, circular)).toThrow(/invalid event/);
    });

    it("does not validate when debug is false, even for a bad event name", () => {
      const fetchMock = vi.fn().mockResolvedValue({ json: async () => ({}) });
      (globalThis as any).fetch = fetchMock;
      const c = new EyeClient({ tenantId: "t1", flushIntervalMs: 0 });
      (c as any).sessionId = "s1";
      expect(() => c.track("")).not.toThrow();
    });
  });
});
