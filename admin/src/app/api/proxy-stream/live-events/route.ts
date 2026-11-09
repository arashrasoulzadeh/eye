import { cookies } from "next/headers";
import { NextRequest, NextResponse } from "next/server";

const BACKEND = process.env.BACKEND_URL || "http://localhost:8080";

export async function GET(req: NextRequest) {
  const key = (await cookies()).get("admin_key")?.value;
  if (!key) {
    return NextResponse.json({ error: "not authenticated" }, { status: 401 });
  }

  const sessionId = req.nextUrl.searchParams.get("session_id") || "";
  const res = await fetch(`${BACKEND}/live-events?session_id=${encodeURIComponent(sessionId)}`, {
    headers: { "x-admin-key": key },
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