import { cookies } from "next/headers";
import { NextRequest, NextResponse } from "next/server";

const BACKEND = process.env.BACKEND_URL || "http://localhost:8080";

async function forward(req: NextRequest, path: string[]) {
  const key = (await cookies()).get("admin_key")?.value;
  if (!key) {
    return NextResponse.json({ error: "not authenticated" }, { status: 401 });
  }

  const search = req.nextUrl.search;
  const url = `${BACKEND}/${path.join("/")}${search}`;
  const init: RequestInit = {
    method: req.method,
    headers: { "x-admin-key": key, "content-type": "application/json" },
  };
  if (req.method !== "GET" && req.method !== "DELETE" && req.method !== "HEAD") {
    init.body = await req.text();
  }

  const res = await fetch(url, init);
  const text = await res.text();
  return new NextResponse(text, {
    status: res.status,
    headers: { "content-type": res.headers.get("content-type") || "application/json" },
  });
}

export async function GET(req: NextRequest, { params }: { params: Promise<{ path: string[] }> }) {
  return forward(req, (await params).path);
}
export async function POST(req: NextRequest, { params }: { params: Promise<{ path: string[] }> }) {
  return forward(req, (await params).path);
}
export async function DELETE(req: NextRequest, { params }: { params: Promise<{ path: string[] }> }) {
  return forward(req, (await params).path);
}
