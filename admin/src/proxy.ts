import { NextRequest, NextResponse } from "next/server";

export function proxy(req: NextRequest) {
  const hasKey = req.cookies.has("admin_key");
  const { pathname } = req.nextUrl;

  if (!hasKey && pathname !== "/login") {
    return NextResponse.redirect(new URL("/login", req.url));
  }
  if (hasKey && pathname === "/login") {
    return NextResponse.redirect(new URL("/dashboard", req.url));
  }
  return NextResponse.next();
}

export const config = {
  matcher: ["/((?!api|_next/static|_next/image|favicon.ico).*)"],
};
