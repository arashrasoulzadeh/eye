import { cookies } from "next/headers";
import { NextRequest, NextResponse } from "next/server";

const BACKEND = process.env.BACKEND_URL || "http://localhost:8080";

export async function GET(req: NextRequest) {
  const jar = await cookies();
  const key = jar.get("admin_key")?.value;
  const session = jar.get("admin_session")?.value;
  if (!key && !session) {
    return NextResponse.json({ error: "not authenticated" }, { status: 401 });
  }
  const res = await fetch(`${BACKEND}/live-events${req.nextUrl.search}`, {
    headers: session ? { "x-admin-session": session } : { "x-admin-key": key! },
    signal: req.signal,
  });

  return new NextResponse(res.body, {
    status: res.status,
    headers: {
      "content-type": "text/event-stream",
      "cache-control": "no-cache",
      connection: "keep-alive",
    },
  });
}