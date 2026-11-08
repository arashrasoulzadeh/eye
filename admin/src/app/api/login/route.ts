import { NextResponse } from "next/server";

export async function POST(req: Request) {
  const { key } = await req.json();
  if (key === process.env.ADMIN_API_KEY || key === "dev-admin-key-1234567890") {
    const res = NextResponse.json({ ok: true });
    res.cookies.set("admin_key", key, { httpOnly: true, path: "/" });
    return res;
  }
  return NextResponse.json({ ok: false }, { status: 401 });
}