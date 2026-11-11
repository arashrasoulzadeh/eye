"use client";
import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";

const NAV = [
  { href: "/dashboard", label: "Dashboard", icon: "⌂" },
  { href: "/stream", label: "Live Stream", icon: "◉" },
  { href: "/analytics", label: "Analytics", icon: "▥" },
  { href: "/graphs", label: "Graphs", icon: "◒" },
  { href: "/questions", label: "Questions", icon: "?" },
  { href: "/tenants", label: "Tenants", icon: "◆" },
  { href: "/sessions", label: "Sessions", icon: "⎘" },
  { href: "/events", label: "Events", icon: "▤" },
  { href: "/test-event", label: "Test Event", icon: "▶" },
  { href: "/cdn-test", label: "CDN Test", icon: "⧉" },
  { href: "/schemas", label: "Schemas", icon: "{ }" },
  { href: "/cohorts", label: "Cohorts", icon: "◔" },
  { href: "/alerts", label: "Alerts", icon: "!" },
  { href: "/operations", label: "Operations", icon: "⚑" },
  { href: "/feature-flags", label: "Feature Flags", icon: "⚙" },
  { href: "/aggregations", label: "Aggregations", icon: "Σ" },
  { href: "/anomalies", label: "Anomalies", icon: "△" },
  { href: "/audit-log", label: "Audit Log", icon: "≡" },
  { href: "/dead-letters", label: "Dead Letters", icon: "!" },
  { href: "/replay", label: "Replay", icon: "↻" },
  { href: "/settings", label: "Settings", icon: "⚙" },
];

export function Sidebar() {
  const pathname = usePathname();
  const router = useRouter();

  async function logout() {
    await fetch("/api/logout", { method: "POST" });
    router.push("/login");
  }

  return (
    <aside className="sticky top-0 flex h-screen w-56 shrink-0 flex-col overflow-y-auto border-r border-border bg-surface">
      <div className="flex items-center gap-2 px-5 py-5">
        <div className="flex h-7 w-7 items-center justify-center rounded-lg bg-accent text-sm font-bold text-accent-fg">
          e
        </div>
        <span className="text-sm font-semibold text-fg">eye admin</span>
      </div>
      <nav className="flex-1 space-y-0.5 px-3">
        {NAV.map((item) => {
          const active = pathname === item.href || pathname?.startsWith(item.href + "/");
          return (
            <Link
              key={item.href}
              href={item.href}
              className={`flex items-center gap-3 rounded-lg px-3 py-2 text-sm transition-colors ${
                active ? "bg-accent/10 text-accent font-medium" : "text-fg-muted hover:bg-surface-2 hover:text-fg"
              }`}
            >
              <span className="w-4 text-center text-xs">{item.icon}</span>
              {item.label}
            </Link>
          );
        })}
      </nav>
      <div className="border-t border-border p-3">
        <button
          onClick={logout}
          className="flex w-full items-center gap-3 rounded-lg px-3 py-2 text-sm text-fg-muted hover:bg-surface-2 hover:text-fg"
        >
          <span className="w-4 text-center text-xs">↩</span>
          Log out
        </button>
      </div>
    </aside>
  );
}
