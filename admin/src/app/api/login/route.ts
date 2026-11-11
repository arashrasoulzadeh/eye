import { NextResponse } from "next/server";

export async function POST(req: Request) {
  const { key } = await req.json();
  const configuredAdminKey = process.env.ADMIN_API_KEY;
  if (configuredAdminKey && key === configuredAdminKey) {
    const res = NextResponse.json({ ok: true });
    res.cookies.set("admin_key", key, {
      httpOnly: true,
      path: "/",
      sameSite: "strict",
      secure: process.env.NODE_ENV === "production",
    });
    res.cookies.delete("admin_email");
    return res;
  }
  return NextResponse.json({ ok: false }, { status: 401 });
}
