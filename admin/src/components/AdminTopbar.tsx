"use client";

import { usePathname } from "next/navigation";

const LABELS: Record<string, string> = {
  dashboard: "Dashboard", stream: "Live Stream", analytics: "Analytics", graphs: "Graphs", questions: "Questions",
  tenants: "Tenants", sessions: "Sessions", events: "Events", schemas: "Schemas", cohorts: "Cohorts", alerts: "Alerts",
  operations: "Operations", settings: "Settings", replay: "Replay", experiments: "Experiments", "feature-flags": "Feature Flags",
  aggregations: "Aggregations", anomalies: "Anomalies", "audit-log": "Audit Log", "dead-letters": "Dead Letters", "test-event": "Test Event", "cdn-test": "CDN Test",
};

export function AdminTopbar() {
  const pathname = usePathname() || "/dashboard";
  const segment = pathname.split("/").filter(Boolean)[0] || "dashboard";
  const title = LABELS[segment] || "Admin";
  return <header className="sticky top-0 z-20 flex min-h-14 items-center justify-between border-b border-border bg-bg/90 px-8 backdrop-blur">
    <div className="flex min-w-0 items-center gap-2 text-sm"><span className="font-medium text-fg">{title}</span>{pathname !== `/${segment}` && <><span className="text-fg-muted">/</span><span className="truncate text-fg-muted">{pathname.split("/").filter(Boolean).slice(1).join(" / ")}</span></>}</div>
    <div className="flex items-center gap-3 text-xs text-fg-muted"><span className="hidden rounded-full border border-success/30 bg-success/5 px-2.5 py-1 text-success sm:inline-flex">● API connected</span><span className="rounded-full border border-border bg-surface px-2.5 py-1">Admin</span></div>
  </header>;
}
