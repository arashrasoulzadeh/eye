"use client";
import { useTenant } from "@/components/TenantContext";
import { useCallback, useEffect, useState } from "react";
import { api, ApiError, DashboardCache } from "@/lib/api";
import { StatCard, Card, CardHeader } from "@/components/Card";
import { LineTrend, BarBreakdown } from "@/components/Chart";
import { Button } from "@/components/Button";
import Link from "next/link";
import { formatTimeRange, useTimeRange } from "@/components/TimeRangeContext";

export default function Dashboard() {
  const [cache, setCache] = useState<DashboardCache | null>(null);
  const { from: globalFrom, to: globalTo, hours } = useTimeRange();
  const [error, setError] = useState(""), [loading, setLoading] = useState(true);
  const timeRangeLabel = formatTimeRange(hours);

  const { tenantId, loading: scopeLoading } = useTenant();
  const load = useCallback(async () => { if (scopeLoading || !globalFrom || !globalTo) return; setLoading(true); setError(""); try { setCache(await api.dashboardCache({ tenant_id: tenantId, from_ts: globalFrom, to_ts: globalTo })); } catch (e) { setError(e instanceof ApiError ? e.message : "Failed to load dashboard metrics"); } finally { setLoading(false); } }, [globalFrom, globalTo, tenantId, scopeLoading]);
  useEffect(() => { void load(); }, [load]);
  const hourlyAverage = cache?.events_by_hour?.length ? cache.events_by_hour.reduce((sum, point) => sum + point.value, 0) / cache.events_by_hour.length : 0;
  const cachedAtMs = cache ? (cache.cached_at < 100000000000 ? cache.cached_at * 1000 : cache.cached_at) : 0;
  const eventsPerMinute = hourlyAverage / 60;
  const eventsPerSecond = eventsPerMinute / 60;
  function downloadReport() {
    if (!cache) return;
    const rows = [["Metric", "Value"], ["Events", String(cache.events)], ["Sessions", String(cache.sessions)], ["Users", String(cache.users ?? cache.sessions)], ["Online users", String(cache.online_users ?? 0)], ["Retention", `${((cache.retention_rate ?? 0) * 100).toFixed(1)}%`], ["Tenants", String(cache.tenants)], ["Events per hour", String(Math.round(hourlyAverage))], ["Time range hours", hours]];
    const csv = rows.map((row) => row.map((value) => `"${value.replaceAll('"', '""')}"`).join(",")).join("\n");
    const url = URL.createObjectURL(new Blob([csv], { type: "text/csv;charset=utf-8" }));
    const link = document.createElement("a"); link.href = url; link.download = `eye-dashboard-${timeRangeLabel.toLowerCase().replaceAll(" ", "-")}.csv`; link.click(); URL.revokeObjectURL(url);
  }

  return (
    <div>
      <div className="flex items-start justify-between gap-4"><div><h1 className="text-xl font-semibold text-fg">Dashboard</h1>
      <p className="mt-1 text-sm text-fg-muted">Metrics for the global <span className="font-medium text-fg">{timeRangeLabel}</span> (refreshed every minute).</p>
      </div><div className="flex gap-2"><Button variant="secondary" onClick={() => window.print()}>Print report</Button><Button variant="secondary" onClick={downloadReport} disabled={!cache}>Download CSV</Button><Button variant="secondary" onClick={() => void load()} loading={loading}>Refresh</Button></div></div>
      <details className="mt-4 rounded-lg border border-border bg-surface-2 px-4 py-3 text-sm">
        <summary className="cursor-pointer font-medium text-fg">What am I seeing?</summary>
        <p className="mt-2 text-fg-muted">These totals come from the server-side dashboard cache, not the browser. The charts summarize the latest materialized event window and refresh automatically once per minute.</p>
      </details>

      {error && <p className="mt-4 rounded border border-danger/30 bg-danger/5 p-3 text-sm text-danger">{error}</p>}

      <div className="mt-6 grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-6">
        <StatCard label="Events" value={cache ? cache.events.toLocaleString() : "…"} hint="Server-cached total" />
        <StatCard label="Sessions" value={cache ? cache.sessions.toLocaleString() : "…"} hint="Server-cached total" />
        <StatCard label="Users" value={cache ? (cache.users ?? cache.sessions ?? 0).toLocaleString() : "…"} hint="Distinct sessions (user proxy)" />
        <StatCard label="Online users" value={cache ? (cache.online_users ?? 0).toLocaleString() : "…"} hint="Seen in the last 5 minutes" />
        <StatCard label="Retention" value={cache ? `${((cache.retention_rate ?? 0) * 100).toFixed(1)}%` : "…"} hint="7-day return rate" />
        <StatCard label="Tenants" value={cache ? cache.tenants.toLocaleString() : "…"} hint="Active tenant records" />
      </div>
      <div className="mt-4 grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-6">
        <StatCard label="Events / hour" value={cache ? Math.round(hourlyAverage).toLocaleString() : "…"} hint="Average cached hourly volume" />
        <StatCard label="Events / minute" value={cache ? eventsPerMinute.toFixed(1) : "…"} hint="Derived from hourly trend" />
        <StatCard label="Events / second" value={cache ? eventsPerSecond.toFixed(2) : "…"} hint="Derived average rate" />
        <StatCard label="Active sessions" value={cache ? (cache.online_users ?? 0).toLocaleString() : "…"} hint="Seen in the last 5 minutes" />
        <StatCard label="Hourly buckets" value={cache ? cache.events_by_hour.length.toLocaleString() : "…"} hint="Available trend points" />
        <StatCard label="Cache age" value={cache ? `${Math.max(0, Math.round((Date.now() - cachedAtMs) / 1000))}s` : "…"} hint="Since server snapshot" />
      </div>

      <div className="mt-6 grid grid-cols-1 gap-6 lg:grid-cols-2">
        <Card>
          <CardHeader title="Events by hour (cached)" />
          <div className="p-5">
            <LineTrend data={cache ? cache.events_by_hour : []} />
          </div>
        </Card>
        <Card>
          <CardHeader title="Events by tenant (cached, top 8)" />
          <div className="p-5">
            <BarBreakdown data={cache ? cache.events_by_tenant : []} />
          </div>
        </Card>
      </div>

      <Card className="mt-6">
        <CardHeader title="Quick links" />
        <div className="grid grid-cols-2 gap-3 p-5 sm:grid-cols-3">
          {[
            ["Manage tenants", "/tenants"],
            ["Browse events", "/events"],
            ["Send a test event", "/test-event"],
            ["Register a schema", "/schemas"],
            ["View sessions", "/sessions"],
          ].map(([label, href]) => (
            <Link
              key={href}
              href={href}
              className="rounded-lg border border-border px-3 py-2.5 text-sm text-fg hover:border-accent hover:text-accent"
            >
              {label}
            </Link>
          ))}
        </div>
      </Card>
      <Card className="mt-6"><CardHeader title="Recommended next steps"/><div className="grid gap-3 p-5 sm:grid-cols-3"><Link href="/health" className="rounded-lg border border-border p-3 text-sm hover:border-accent"><p className="font-medium text-fg">Review system health</p><p className="mt-1 text-xs text-fg-muted">SLO, anomalies, alerts, and dead letters.</p></Link><Link href="/analytics" className="rounded-lg border border-border p-3 text-sm hover:border-accent"><p className="font-medium text-fg">Investigate trends</p><p className="mt-1 text-xs text-fg-muted">Open a shareable query for the current range.</p></Link><Link href="/stream" className="rounded-lg border border-border p-3 text-sm hover:border-accent"><p className="font-medium text-fg">Watch live traffic</p><p className="mt-1 text-xs text-fg-muted">Inspect the latest events and sessions.</p></Link></div></Card>
    </div>
  );
}
