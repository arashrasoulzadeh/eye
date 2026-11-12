import { Suspense } from "react";
import { Sidebar } from "@/components/Sidebar";
import { ToastProvider } from "@/components/Toast";
import { AdminTopbar } from "@/components/AdminTopbar";
import { TenantProvider } from "@/components/TenantContext";
import { TimeRangeProvider } from "@/components/TimeRangeContext";

export default function DashboardLayout({ children }: { children: React.ReactNode }) {
  return (
    <ToastProvider><TenantProvider><TimeRangeProvider>
      <div className="flex min-h-screen">
        {/* Sidebar reads usePathname(), which needs a Suspense boundary for
            routes (like /tenants/[id]) that can't be prerendered statically
            under cacheComponents. */}
        <Suspense>
          <Sidebar />
        </Suspense>
        <main className="flex-1 overflow-y-auto">
          <Suspense fallback={<header className="min-h-14 border-b border-border" />}><AdminTopbar /></Suspense>
          <div className="admin-content mx-auto max-w-6xl px-8 py-8">{children}</div>
        </main>
      </div>
    </TimeRangeProvider></TenantProvider></ToastProvider>
  );
}
