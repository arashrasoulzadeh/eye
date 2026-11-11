"use client";
import { useEffect, useState } from "react";
import { api, ApiError, DashboardCache } from "@/lib/api";
import { StatCard, Card, CardHeader } from "@/components/Card";
import { LineTrend, BarBreakdown } from "@/components/Chart";

export default function Dashboard() {
  const [cache, setCache] = useState<DashboardCache | null>(null);
  const [error, setError] = useState("");

  useEffect(() => {
    api.dashboardCache().then(setCache).catch((e: ApiError) => setError(e.message));
  }, []);

  return (
    <div>
      <h1 className="text-xl font-semibold text-fg">Dashboard</h1>
      <p className="mt-1 text-sm text-fg-muted">Cached metrics (refreshed every minute).</p>

      {error && <p className="mt-4 text-sm text-danger">{error}</p>}

      <div className="mt-6 grid grid-cols-1 gap-4 sm:grid-cols-3">
        <StatCard label="Events" value={cache ? cache.events.toLocaleString() : "—"} />
        <StatCard label="Sessions" value={cache ? cache.sessions.toLocaleString() : "—"} />
        <StatCard label="Tenants" value={cache ? cache.tenants.toLocaleString() : "—"} />
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
            <a
              key={href}
              href={href}
              className="rounded-lg border border-border px-3 py-2.5 text-sm text-fg hover:border-accent hover:text-accent"
            >
              {label}
            </a>
          ))}
        </div>
      </Card>
    </div>
  );
}
