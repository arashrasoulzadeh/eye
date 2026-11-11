import { NextResponse } from "next/server";
import { randomBytes } from "node:crypto";

export async function GET(req: Request) {
  const issuer = process.env.OIDC_ISSUER;
  const clientId = process.env.OIDC_CLIENT_ID;
  if (!issuer || !clientId) return NextResponse.json({ error: "OIDC is not configured" }, { status: 503 });
  const discovery = await fetch(`${issuer.replace(/\/$/, "")}/.well-known/openid-configuration`, { cache: "no-store" });
  if (!discovery.ok) return NextResponse.json({ error: "OIDC discovery failed" }, { status: 502 });
  const metadata = await discovery.json() as { authorization_endpoint: string };
  const state = randomBytes(32).toString("hex");
  const nonce = randomBytes(32).toString("hex");
  const redirectUri = process.env.OIDC_REDIRECT_URI || new URL("/api/oidc/callback", req.url).toString();
  const url = new URL(metadata.authorization_endpoint);
  url.searchParams.set("client_id", clientId);
  url.searchParams.set("redirect_uri", redirectUri);
  url.searchParams.set("response_type", "code");
  url.searchParams.set("scope", "openid profile email");
  url.searchParams.set("state", state);
  url.searchParams.set("nonce", nonce);
  const res = NextResponse.redirect(url);
  const cookie = { httpOnly: true, secure: process.env.NODE_ENV === "production", sameSite: "lax" as const, path: "/", maxAge: 600 };
  res.cookies.set("oidc_state", state, cookie);
  res.cookies.set("oidc_nonce", nonce, cookie);
  return res;
}
