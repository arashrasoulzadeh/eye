import { NextResponse } from "next/server";

export async function POST(req: Request) {
  const { key } = await req.json();
  const configuredAdminKey = process.env.ADMIN_API_KEY;
  if (configuredAdminKey && key === configuredAdminKey) {
    const res = NextResponse.json({ ok: true });
    const backend = process.env.BACKEND_URL || "http://localhost:8080";
    try {
      const session = await fetch(`${backend}/admin-sessions`, { method: "POST", headers: { "x-admin-key": key, "content-type": "application/json" }, body: JSON.stringify({}) });
      if (session.ok) {
        const body = await session.json() as { token?: string; expires_at?: number };
        if (body.token) res.cookies.set("admin_session", body.token, { httpOnly: true, path: "/", sameSite: "strict", secure: process.env.NODE_ENV === "production", maxAge: Math.max(60, (body.expires_at || 0) - Math.floor(Date.now() / 1000)) });
      }
    } catch { /* legacy key cookie remains the migration fallback */ }
    res.cookies.set("admin_key", key, {
      httpOnly: true,
      path: "/",
      sameSite: "strict",
      secure: process.env.NODE_ENV === "production",
      maxAge: 8 * 60 * 60,
    });
    res.cookies.delete("admin_email");
    return res;
  }
  return NextResponse.json({ ok: false }, { status: 401 });
}
