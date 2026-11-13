import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { api, ApiError } from "./api";

function jsonResponse(body: unknown, status = 200) {
  return {
    ok: status >= 200 && status < 300,
    status,
    text: async () => JSON.stringify(body),
  } as Response;
}

describe("api client", () => {
  let fetchMock: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    api.tenants.clearCache();
    fetchMock = vi.fn();
    Object.defineProperty(globalThis, "fetch", { value: fetchMock, configurable: true, writable: true });
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("calls the proxy route with the given path", async () => {
    fetchMock.mockResolvedValueOnce(jsonResponse([{ ID: "t1", Name: "acme" }]));
    const tenants = await api.tenants.list();
    expect(tenants).toEqual([{ ID: "t1", Name: "acme" }]);
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/proxy/tenants",
      expect.objectContaining({ cache: "no-store" })
    );
  });

  it("throws ApiError with the response status on a non-401 failure", async () => {
    fetchMock.mockResolvedValueOnce({ ok: false, status: 500, text: async () => "boom" } as Response);
    await expect(api.tenants.list()).rejects.toMatchObject({ status: 500, message: "boom" });
  });

  it("normalizes a non-array tenant response before sorting or caching", async () => {
    fetchMock.mockResolvedValueOnce(jsonResponse({ items: [] }));
    await expect(api.tenants.list()).resolves.toEqual([]);
  });

  it("on 401, logs out and redirects to /login instead of returning data", async () => {
    // First call: the failing proxied request. Second call: the logout
    // route handleUnauthorized() fires internally.
    fetchMock
      .mockResolvedValueOnce({ ok: false, status: 401, text: async () => "unauthorized" } as Response)
      .mockResolvedValueOnce(jsonResponse({ ok: true }));

    // jsdom doesn't implement real navigation; stub the setter so the
    // assignment in handleUnauthorized doesn't throw.
    const locationSpy = vi.fn();
    Object.defineProperty(window, "location", {
      value: { ...window.location },
      writable: true,
    });
    Object.defineProperty(window.location, "href", {
      set: locationSpy,
      configurable: true,
    });

    await expect(api.tenants.list()).rejects.toBeInstanceOf(ApiError);

    // handleUnauthorized's fetch("/api/logout") + redirect happen in a
    // detached promise chain; flush microtasks before asserting.
    await new Promise((r) => setTimeout(r, 0));

    expect(fetchMock).toHaveBeenCalledWith("/api/logout", expect.objectContaining({ method: "POST" }));
    expect(locationSpy).toHaveBeenCalledWith("/login");
  });

  it("events.list hits /query (not /events) and forwards from_ts/to_ts", async () => {
    fetchMock.mockResolvedValueOnce(jsonResponse([]));
    await api.events.list({ tenant_id: "t1", from_ts: 1000, to_ts: 2000 });
    const [url] = fetchMock.mock.calls[0];
    expect(url).toContain("/api/proxy/query?");
    expect(url).toContain("tenant_id=t1");
    expect(url).toContain("from_ts=1000");
    expect(url).toContain("to_ts=2000");
  });

  it("normalizes null collection responses so pages can safely iterate them", async () => {
    fetchMock.mockResolvedValueOnce(jsonResponse(null));
    await expect(api.events.list({ tenant_id: "t1" })).resolves.toEqual([]);
    fetchMock.mockResolvedValueOnce(jsonResponse(null));
    await expect(api.analytics.retention("t1")).resolves.toEqual([]);
  });

  it("returns null for an empty (e.g. 204) response body", async () => {
    fetchMock.mockResolvedValueOnce({ ok: true, status: 204, text: async () => "" } as Response);
    const result = await api.tenants.remove("t1");
    expect(result).toBeNull();
  });
});
