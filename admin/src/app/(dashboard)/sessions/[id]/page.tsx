// Session ids are only known at runtime from the backend, so this route
// can't be prerendered as a static shell with cacheComponents on — render
// it on demand (a blocking, non-prerendered route) behind Suspense instead
// (same reasoning as the tenant detail route).
export const instant = false;

import { Suspense } from "react";
import { connection } from "next/server";
import SessionDetailClient from "./SessionDetailClient";

export default async function SessionDetailPage() {
  await connection();
  return (
    <Suspense fallback={<p className="text-sm text-fg-muted">Loading session…</p>}>
      <SessionDetailClient />
    </Suspense>
  );
}
