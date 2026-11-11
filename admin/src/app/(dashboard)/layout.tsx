import { Suspense } from "react";
import { Sidebar } from "@/components/Sidebar";
import { ToastProvider } from "@/components/Toast";
import { AdminGuide } from "@/components/AdminGuide";

export default function DashboardLayout({ children }: { children: React.ReactNode }) {
  return (
    <ToastProvider>
      <div className="flex min-h-screen">
        {/* Sidebar reads usePathname(), which needs a Suspense boundary for
            routes (like /tenants/[id]) that can't be prerendered statically
            under cacheComponents. */}
        <Suspense>
          <Sidebar />
        </Suspense>
        <main className="flex-1 overflow-y-auto">
          <div className="mx-auto max-w-6xl px-8 py-8"><AdminGuide />{children}</div>
        </main>
      </div>
    </ToastProvider>
  );
}
