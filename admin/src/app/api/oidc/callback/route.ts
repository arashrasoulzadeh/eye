import { NextResponse } from "next/server";
import { cookies } from "next/headers";
import { createRemoteJWKSet, jwtVerify } from "jose";

export async function GET(req: Request) {
  const issuer = process.env.OIDC_ISSUER?.replace(/\/$/, "");
  const clientId = process.env.OIDC_CLIENT_ID;
  const clientSecret = process.env.OIDC_CLIENT_SECRET;
  const jar = await cookies();
  const expectedState = jar.get("oidc_state")?.value;
  const nonce = jar.get("oidc_nonce")?.value;
  const params = new URL(req.url).searchParams;
  if (!issuer || !clientId || !clientSecret || !expectedState || params.get("state") !== expectedState) return NextResponse.redirect(new URL("/login?error=oidc", req.url));
  const discovery = await fetch(`${issuer}/.well-known/openid-configuration`, { cache: "no-store" });
  const metadata = await discovery.json() as { token_endpoint: string; jwks_uri: string; issuer: string };
  const tokenRes = await fetch(metadata.token_endpoint, { method: "POST", headers: { "content-type": "application/x-www-form-urlencoded" }, body: new URLSearchParams({ grant_type: "authorization_code", code: params.get("code") || "", redirect_uri: process.env.OIDC_REDIRECT_URI || new URL("/api/oidc/callback", req.url).toString(), client_id: clientId, client_secret: clientSecret }) });
  if (!tokenRes.ok) return NextResponse.redirect(new URL("/login?error=oidc", req.url));
  const tokens = await tokenRes.json() as { id_token?: string };
  if (!tokens.id_token) return NextResponse.redirect(new URL("/login?error=oidc", req.url));
  const verified = await jwtVerify(tokens.id_token, createRemoteJWKSet(new URL(metadata.jwks_uri)), { issuer: metadata.issuer || issuer, audience: clientId });
  if (verified.payload.nonce !== nonce) return NextResponse.redirect(new URL("/login?error=oidc", req.url));
  const email = String(verified.payload.email || "");
  const allowed = (process.env.OIDC_ALLOWED_EMAILS || "").split(",").map((x) => x.trim()).filter(Boolean);
  if (allowed.length && !allowed.includes(email)) return NextResponse.redirect(new URL("/login?error=forbidden", req.url));
  const allowedGroups = (process.env.OIDC_ALLOWED_GROUPS || "").split(",").map((x) => x.trim()).filter(Boolean);
  const groups = Array.isArray(verified.payload.groups) ? verified.payload.groups.map(String) : [];
  if (allowedGroups.length && !allowedGroups.some((group) => groups.includes(group))) return NextResponse.redirect(new URL("/login?error=forbidden", req.url));
  const res = NextResponse.redirect(new URL("/dashboard", req.url));
  res.cookies.delete("oidc_state"); res.cookies.delete("oidc_nonce");
  res.cookies.set("admin_key", process.env.ADMIN_API_KEY!, { httpOnly: true, secure: process.env.NODE_ENV === "production", sameSite: "strict", path: "/" });
  if (email) res.cookies.set("admin_email", email, { httpOnly: true, secure: process.env.NODE_ENV === "production", sameSite: "strict", path: "/" });
  return res;
}
