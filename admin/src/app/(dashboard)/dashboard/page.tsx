"use client";
import { useCallback, useEffect, useState } from "react";
import { api, ApiError, DashboardCache } from "@/lib/api";
import { StatCard, Card, CardHeader } from "@/components/Card";
import { LineTrend, BarBreakdown } from "@/components/Chart";
import { Button } from "@/components/Button";
import Link from "next/link";

export default function Dashboard() {
  const [cache, setCache] = useState<DashboardCache | null>(null);
  const [error, setError] = useState(""), [loading, setLoading] = useState(true);

  const load = useCallback(async () => { setLoading(true); setError(""); try { setCache(await api.dashboardCache()); } catch (e) { setError(e instanceof ApiError ? e.message : "Failed to load dashboard metrics"); } finally { setLoading(false); } }, []);
  useEffect(() => { void load(); }, [load]);

  return (
    <div>
      <div className="flex items-start justify-between gap-4"><div><h1 className="text-xl font-semibold text-fg">Dashboard</h1>
      <p className="mt-1 text-sm text-fg-muted">Cached metrics (refreshed every minute).</p>
      </div><Button variant="secondary" onClick={() => void load()} loading={loading}>Refresh</Button></div>
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
    </div>
  );
}
