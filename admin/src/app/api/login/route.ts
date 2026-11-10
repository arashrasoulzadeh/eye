import { NextResponse } from "next/server";

export async function POST(req: Request) {
  const { key } = await req.json();
  const configuredAdminKey = process.env.ADMIN_API_KEY;
  if (configuredAdminKey && key === configuredAdminKey) {
    const res = NextResponse.json({ ok: true });
    res.cookies.set("admin_key", key, { httpOnly: true, path: "/" });
    return res;
  }
  return NextResponse.json({ ok: false }, { status: 401 });
}
