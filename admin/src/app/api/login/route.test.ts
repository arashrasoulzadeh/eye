import { afterEach, describe, expect, it } from "vitest";
import { POST } from "./route";

const originalAdminKey = process.env.ADMIN_API_KEY;

afterEach(() => {
  if (originalAdminKey === undefined) delete process.env.ADMIN_API_KEY;
  else process.env.ADMIN_API_KEY = originalAdminKey;
});

describe("POST /api/login", () => {
  it("rejects login when ADMIN_API_KEY is not configured", async () => {
    delete process.env.ADMIN_API_KEY;

    const response = await POST(
      new Request("http://localhost/api/login", {
        method: "POST",
        body: JSON.stringify({ key: "dev-admin-key-1234567890" }),
      }),
    );

    expect(response.status).toBe(401);
  });

  it("accepts only the configured ADMIN_API_KEY", async () => {
    process.env.ADMIN_API_KEY = "configured-admin-key";

    const response = await POST(
      new Request("http://localhost/api/login", {
        method: "POST",
        body: JSON.stringify({ key: "configured-admin-key" }),
      }),
    );

    expect(response.status).toBe(200);
    expect((await response.json()).ok).toBe(true);
  });

  it("sets a strict, secure session cookie in production", async () => {
    process.env.ADMIN_API_KEY = "configured-admin-key";
    const originalNodeEnv = process.env.NODE_ENV;
    process.env.NODE_ENV = "production";

    const response = await POST(
      new Request("http://localhost/api/login", {
        method: "POST",
        body: JSON.stringify({ key: "configured-admin-key" }),
      }),
    );

    expect(response.headers.get("set-cookie")).toContain("HttpOnly");
    expect(response.headers.get("set-cookie")?.toLowerCase()).toContain("samesite=strict");
    expect(response.headers.get("set-cookie")).toContain("Secure");
    process.env.NODE_ENV = originalNodeEnv;
  });
});
