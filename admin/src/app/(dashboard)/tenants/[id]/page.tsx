// Tenant ids are only known at runtime from the backend, so this route
// can't be prerendered as a static shell with cacheComponents on — render
// it on demand (a blocking, non-prerendered route) behind Suspense instead.
export const instant = false;

import { Suspense } from "react";
import TenantDetailClient from "./TenantDetailClient";

export default function TenantDetailPage() {
  return (
    <Suspense fallback={<p className="text-sm text-fg-muted">Loading tenant…</p>}>
      <TenantDetailClient />
    </Suspense>
  );
}
