"use client";
import { useEffect, useState } from "react";
import { api, Metrics, Event, Tenant, ApiError } from "@/lib/api";
import { StatCard, Card, CardHeader } from "@/components/Card";
import { LineTrend, BarBreakdown, ChartPoint } from "@/components/Chart";

// Buckets events by hour-of-day across whatever window /query returns (it's
// capped server-side), giving an honest "recent activity" shape rather than
// a fabricated time series.
function eventsPerHour(events: Event[]): ChartPoint[] {
  const buckets = new Map<string, number>();
  for (const e of events) {
    const d = new Date(e.ClientTS);
    const label = `${String(d.getHours()).padStart(2, "0")}:00`;
    buckets.set(label, (buckets.get(label) || 0) + 1);
  }
  return [...buckets.entries()].sort(([a], [b]) => a.localeCompare(b)).map(([label, value]) => ({ label, value }));
}

function eventsPerTenant(events: Event[], tenants: Tenant[]): ChartPoint[] {
  const names = new Map(tenants.map((t) => [t.ID, t.Name]));
  const counts = new Map<string, number>();
  for (const e of events) counts.set(e.TenantID, (counts.get(e.TenantID) || 0) + 1);
  return [...counts.entries()]
    .sort((a, b) => b[1] - a[1])
    .slice(0, 8)
    .map(([id, value]) => ({ label: names.get(id) || id.slice(0, 8), value }));
}

export default function Dashboard() {
  const [metrics, setMetrics] = useState<Metrics | null>(null);
  const [events, setEvents] = useState<Event[]>([]);
  const [tenants, setTenants] = useState<Tenant[]>([]);
  const [error, setError] = useState("");

  useEffect(() => {
    api.metrics().then(setMetrics).catch((e: ApiError) => setError(e.message));
    api.events.list({}).then(setEvents).catch(() => {});
    api.tenants.list().then(setTenants).catch(() => {});
  }, []);

  return (
    <div>
      <h1 className="text-xl font-semibold text-fg">Dashboard</h1>
      <p className="mt-1 text-sm text-fg-muted">Live counts straight from the backend.</p>

      {error && <p className="mt-4 text-sm text-danger">{error}</p>}

      <div className="mt-6 grid grid-cols-1 gap-4 sm:grid-cols-3">
        <StatCard label="Events" value={metrics ? metrics.events.toLocaleString() : "—"} />
        <StatCard label="Sessions" value={metrics ? metrics.sessions.toLocaleString() : "—"} />
        <StatCard label="Tenants" value={metrics ? metrics.tenants.toLocaleString() : "—"} />
      </div>

      <div className="mt-6 grid grid-cols-1 gap-6 lg:grid-cols-2">
        <Card>
          <CardHeader title="Events by hour (first 100, by client time)" />
          <div className="p-5">
            <LineTrend data={eventsPerHour(events)} />
          </div>
        </Card>
        <Card>
          <CardHeader title="Events by tenant (top 8 of first 100)" />
          <div className="p-5">
            <BarBreakdown data={eventsPerTenant(events, tenants)} />
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
