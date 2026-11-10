import { afterEach, describe, expect, it, vi } from "vitest";
import { EyeClient } from "./index";

afterEach(() => vi.restoreAllMocks());

describe("Web SDK ↔ ingestion API contract", () => {
  it("sends the backend-compatible session and ingest requests", async () => {
    const fetchMock = vi.spyOn(globalThis, "fetch")
      .mockResolvedValueOnce(new Response(JSON.stringify({ ID: "session-contract" }), { status: 201 }))
      .mockResolvedValueOnce(new Response(JSON.stringify({ accepted: 1, rejected: 0 }), { status: 200 }));
    const client = new EyeClient({ tenantId: "tenant-contract", endpoint: "https://collector.example", apiKey: "tenant-key", flushIntervalMs: 0 });

    await client.ensureSession("https://collector.example", "tenant-key");
    client.track("signup", { source: "landing" }, { plan: "pro" });
    await client.flush();

    expect(fetchMock.mock.calls[0][0]).toBe("https://collector.example/sessions");
    expect(fetchMock.mock.calls[0][1]).toMatchObject({
      method: "POST",
      headers: { "content-type": "application/json", "x-api-key": "tenant-key" },
    });
    expect(fetchMock.mock.calls[1][0]).toBe("https://collector.example/ingest");
    const ingestRequest = fetchMock.mock.calls[1][1] as RequestInit;
    expect(ingestRequest.headers).toEqual({ "content-type": "application/json", "x-api-key": "tenant-key" });
    expect(JSON.parse(ingestRequest.body as string)).toMatchObject({
      events: [{ session_id: "session-contract", event_name: "signup", payload: { plan: "pro" }, dims: { source: "landing" } }],
    });
  });
});
