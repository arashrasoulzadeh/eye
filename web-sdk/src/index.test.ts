import { afterEach, describe, expect, it, vi } from "vitest";
import { EyeClient } from "./index";

afterEach(() => vi.restoreAllMocks());

describe("EyeClient", () => {
  it("creates and persists a session from the API response", async () => {
    const fetchMock = vi.spyOn(globalThis, "fetch").mockResolvedValue(
      new Response(JSON.stringify({ session_id: "session-123" }), { status: 201 }),
    );
    const client = new EyeClient({ tenantId: "tenant-1", flushIntervalMs: 0 });

    await expect(client.ensureSession("https://api.example", "api-key")).resolves.toBe("session-123");
    expect(fetchMock).toHaveBeenCalledWith(
      "https://api.example/sessions",
      expect.objectContaining({ method: "POST", headers: expect.objectContaining({ "x-api-key": "api-key" }) }),
    );
    expect(client.getSessionId()).toBe("session-123");
  });

  it("flushes tracked events using the backend wire format", async () => {
    const fetchMock = vi.spyOn(globalThis, "fetch")
      .mockResolvedValueOnce(new Response(JSON.stringify({ session_id: "session-123" }), { status: 201 }))
      .mockResolvedValueOnce(new Response(JSON.stringify({ accepted: 1 }), { status: 200 }));
    const client = new EyeClient({
      tenantId: "tenant-1",
      endpoint: "https://api.example",
      apiKey: "api-key",
      flushIntervalMs: 0,
    });

    await client.ensureSession("https://api.example", "api-key");
    client.track("purchase", { plan: "pro" }, { amount: 99 });
    await client.flush();

    const request = fetchMock.mock.calls[1][1] as RequestInit;
    const body = JSON.parse(request.body as string);
    expect(body.events).toHaveLength(1);
    expect(body.events[0]).toMatchObject({
      session_id: "session-123",
      event_name: "purchase",
      payload: { amount: 99 },
      dims: { plan: "pro" },
    });
  });
});
