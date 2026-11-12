import { NextResponse } from "next/server";
import { cookies } from "next/headers";

export async function POST() {
  const session = (await cookies()).get("admin_session")?.value;
  if (session) {
    try { await fetch(`${process.env.BACKEND_URL || "http://localhost:8080"}/admin-sessions`, { method: "DELETE", headers: { "x-admin-session": session } }); } catch { /* cookie expiry still logs the browser out */ }
  }
  const res = NextResponse.json({ ok: true });
  res.cookies.set("admin_key", "", { httpOnly: true, path: "/", maxAge: 0 });
  res.cookies.set("admin_session", "", { httpOnly: true, path: "/", maxAge: 0 });
  res.cookies.set("admin_email", "", { httpOnly: true, path: "/", maxAge: 0 });
  return res;
}
