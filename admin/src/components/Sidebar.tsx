"use client";
import Link from "next/link";
import { useState } from "react";
import { usePathname, useRouter } from "next/navigation";
import { Combobox } from "@/components/Combobox";
import { useTenant } from "@/components/TenantContext";

const NAV = [
  { href: "/dashboard", label: "Dashboard", icon: "⌂" },
  { href: "/stream", label: "Live Stream", icon: "◉" },
  { href: "/analytics", label: "Analytics", icon: "▥" },
  { href: "/analytics/funnel", label: "Funnels", icon: "↘" },
  { href: "/analytics/retention", label: "Retention", icon: "↺" },
  { href: "/tenants", label: "Tenants", icon: "◆" },
  { href: "/sessions", label: "Sessions", icon: "⎘" },
  { href: "/users", label: "Users", icon: "♙" },
  { href: "/events", label: "Events", icon: "▤" },
  { href: "/alerts", label: "Alerts", icon: "!" },
  { href: "/operations", label: "Operations", icon: "⚑" },
  { href: "/admin-users", label: "Admin Users", icon: "♙" },
  { href: "/admin-teams", label: "Admin Teams", icon: "♧" },
  { href: "/graphs", label: "Graphs", icon: "◒" },
  { href: "/questions", label: "Questions", icon: "?" },
  { href: "/test-event", label: "Test Event", icon: "▶" },
  { href: "/cdn-test", label: "CDN Test", icon: "⧉" },
  { href: "/schemas", label: "Schemas", icon: "{ }" },
  { href: "/cohorts", label: "Cohorts", icon: "◔" },
  { href: "/feature-flags", label: "Feature Flags", icon: "⚙" },
  { href: "/aggregations", label: "Aggregations", icon: "Σ" },
  { href: "/anomalies", label: "Anomalies", icon: "△" },
  { href: "/audit-log", label: "Audit Log", icon: "≡" },
  { href: "/dead-letters", label: "Dead Letters", icon: "!" },
  { href: "/replay", label: "Replay", icon: "↻" },
  { href: "/experiments", label: "Experiments", icon: "◈" },
  { href: "/settings", label: "Settings", icon: "⚙" },
  { href: "/slo", label: "SLO & Error Budget", icon: "◷" },
];

export function Sidebar() {
  const pathname = usePathname();
  const router = useRouter();
  const [moreOpen, setMoreOpen] = useState(false);
  const { tenants, tenantId, setTenantId, loading } = useTenant();
  const primary = NAV.slice(0, 10);
  const secondary = NAV.slice(10);
  // Analytics has dedicated child routes. Keep the parent exact-match only so
  // Funnel and Retention never appear selected at the same time as Analytics.
  const isActive = (href: string) => pathname === href || (href !== "/analytics" && pathname?.startsWith(href + "/"));
  const secondaryActive = secondary.some((item) => isActive(item.href));

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
      <div className="px-3 pb-4">
        <label className="mb-1 block px-2 text-[10px] font-medium uppercase tracking-wide text-fg-muted">Tenant scope</label>
        <Combobox value={tenantId} onChange={setTenantId} options={[{ value: "", label: "All tenants" }, ...tenants.map((item) => ({ value: item.ID, label: item.Name }))]} placeholder={loading ? "Loading tenants…" : "All tenants"} maxVisible={6} />
      </div>
      <nav className="flex-1 space-y-0.5 px-3">
        {primary.map((item) => {
          const active = isActive(item.href);
          return (
            <Link
              key={item.href}
              onClick={() => setMoreOpen(false)}
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
        <button type="button" aria-expanded={moreOpen} onClick={() => setMoreOpen((open) => !open)} className={`flex w-full items-center justify-between rounded-lg px-3 py-2 text-sm ${secondaryActive ? "bg-accent/10 text-accent font-medium" : "text-fg-muted hover:bg-surface-2 hover:text-fg"}`}>
          <span className="flex items-center gap-3"><span className="w-4 text-center text-xs">⋯</span>More <span className="text-xs text-fg-muted">({secondary.length})</span></span>
          <span className="text-xs">{moreOpen ? "▴" : "▾"}</span>
        </button>
        {moreOpen && <div data-testid="sidebar-more-items" className="mt-1 border-l border-border pl-2">{secondary.map((item) => {
          const active = isActive(item.href);
          return <Link key={item.href} href={item.href} onClick={() => setMoreOpen(false)} className={`flex items-center gap-3 rounded-lg px-3 py-2 text-sm transition-colors ${active ? "bg-accent/10 text-accent font-medium" : "text-fg-muted hover:bg-surface-2 hover:text-fg"}`}><span className="w-4 text-center text-xs">{item.icon}</span>{item.label}</Link>;
        })}</div>}
      </nav>
      <div className="border-t border-border p-3">
        <div className="mb-2 rounded-md bg-surface-2 px-3 py-2 text-[11px] text-fg-muted" title="Current release commit">
          <div className="uppercase tracking-wide">Release</div>
          <code className="font-mono text-accent">e391dec</code>
        </div>
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
