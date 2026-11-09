"use client";
import { useEffect, useMemo, useState } from "react";
import { useParams } from "next/navigation";
import { api, Tenant, Session, Event, ApiError } from "@/lib/api";
import { Card, CardHeader, StatCard } from "@/components/Card";
import { Button } from "@/components/Button";
import { CopyButton } from "@/components/CopyButton";
import { EntityLink } from "@/components/EntityLink";
import { DataTable, Column } from "@/components/DataTable";
import { ConfirmDialog } from "@/components/ConfirmDialog";
import { LineTrend, ChartPoint } from "@/components/Chart";
import { LiveCounter } from "@/components/LiveCounter";
import { useToast } from "@/components/Toast";

function eventsPerHour(events: Event[]): ChartPoint[] {
  const buckets = new Map<string, number>();
  for (const e of events) {
    const d = new Date(e.ClientTS);
    const label = `${String(d.getHours()).padStart(2, "0")}:00`;
    buckets.set(label, (buckets.get(label) || 0) + 1);
  }
  return [...buckets.entries()].sort(([a], [b]) => a.localeCompare(b)).map(([label, value]) => ({ label, value }));
}

export default function TenantDetailClient() {
  const params = useParams<{ id: string }>();
  const tenantId = params.id;
  const { push } = useToast();

  const [tenant, setTenant] = useState<Tenant | null>(null);
  const [sessions, setSessions] = useState<Session[]>([]);
  const [events, setEvents] = useState<Event[]>([]);
  const [loading, setLoading] = useState(true);
  const [notFound, setNotFound] = useState(false);
  const [rotated, setRotated] = useState<string | null>(null);
  const [confirmingRotate, setConfirmingRotate] = useState(false);

  useEffect(() => {
    let cancelled = false;
    async function load() {
      setLoading(true);
      try {
        // There is no GET /tenants/{id} endpoint — the admin API only
        // supports listing all tenants, so the detail page finds this one
        // from that list rather than inventing a backend call that doesn't
        // exist.
        const [tenants, allSessions, tenantEvents] = await Promise.all([
          api.tenants.list(),
          api.sessions.list(),
          api.events.list({ tenant_id: tenantId }),
        ]);
        if (cancelled) return;
        const t = tenants.find((t) => t.ID === tenantId) || null;
        setTenant(t);
        setNotFound(!t);
        setSessions(allSessions.filter((s) => s.TenantID === tenantId));
        setEvents(tenantEvents);
      } catch (e) {
        if (!cancelled) push("error", e instanceof ApiError ? e.message : "Failed to load tenant");
      } finally {
        if (!cancelled) setLoading(false);
      }
    }
    load();
    return () => {
      cancelled = true;
    };
  }, [tenantId, push]);

  const eventCount = useMemo(() => events.length, [events]);

  const sessionColumns: Column<Session>[] = [
    { key: "id", header: "Session ID", render: (s) => <EntityLink kind="session" id={s.ID} truncate={14} /> },
    { key: "events", header: "Events", render: (s) => s.EventCount, sortValue: (s) => s.EventCount },
    { key: "started", header: "Started", render: (s) => (s.StartedAt ? new Date(s.StartedAt).toLocaleString() : "—"), sortValue: (s) => s.StartedAt },
    { key: "last_seen", header: "Last seen", render: (s) => (s.LastSeen ? new Date(s.LastSeen).toLocaleString() : "—"), sortValue: (s) => s.LastSeen },
  ];

  async function rotate() {
    if (!tenant) return;
    try {
      const { api_key } = await api.tenants.rotateKey(tenant.ID);
      setRotated(api_key);
    } catch (e) {
      push("error", e instanceof ApiError ? e.message : "Failed to rotate key");
    } finally {
      setConfirmingRotate(false);
    }
  }

  if (loading) {
    return <p className="text-sm text-fg-muted">Loading tenant…</p>;
  }

  if (notFound || !tenant) {
    return (
      <div>
        <h1 className="text-xl font-semibold text-fg">Tenant not found</h1>
        <p className="mt-1 text-sm text-fg-muted">
          No tenant with id <span className="font-mono">{tenantId}</span>.
        </p>
        <a href="/tenants" className="mt-4 inline-block text-sm text-accent hover:underline">
          Back to tenants
        </a>
      </div>
    );
  }

  return (
    <div>
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-xl font-semibold text-fg">{tenant.Name}</h1>
          <p className="mt-1 flex items-center gap-2 text-sm text-fg-muted">
            <span>Tenant</span>
            <CopyButton value={tenant.ID} label={tenant.ID} />
          </p>
        </div>
        <Button variant="secondary" onClick={() => setConfirmingRotate(true)}>
          Rotate key
        </Button>
      </div>

      <div className="mt-6 grid grid-cols-1 gap-4 sm:grid-cols-4">
        <StatCard label="Sessions" value={sessions.length} />
        <StatCard label="Events (loaded)" value={eventCount} hint="Capped by /query's page size" />
        <StatCard label="Created" value={tenant.CreatedAt ? new Date(tenant.CreatedAt).toLocaleDateString() : "—"} />
        <LiveCounter tenantId={tenant.ID} />
      </div>

      <Card className="mt-6">
        <CardHeader title="Events by hour (loaded sample)" />
        <div className="p-5">
          <LineTrend data={eventsPerHour(events)} />
        </div>
      </Card>

      <Card className="mt-6">
        <CardHeader title={`Sessions (${sessions.length})`} />
        <DataTable columns={sessionColumns} rows={sessions} rowKey={(s) => s.ID} emptyMessage="No sessions for this tenant yet." />
      </Card>

      <Card className="mt-6">
        <CardHeader title="Retention settings" />
        <div className="p-5 text-sm text-fg-muted">
          Retention is configured globally via the server&apos;s <code>RETENTION_HOURS</code> env var — there is no
          per-tenant retention setting or update API yet, so this is read-only, same as the main{" "}
          <a href="/settings" className="text-accent hover:underline">
            Settings
          </a>{" "}
          page.
        </div>
      </Card>

      <ConfirmDialog
        open={confirmingRotate}
        title={`Rotate API key for "${tenant.Name}"?`}
        description="The old key stops working immediately."
        confirmLabel="Rotate key"
        danger
        onConfirm={rotate}
        onCancel={() => setConfirmingRotate(false)}
      />

      {rotated && (
        <ConfirmDialog
          open
          title={`New API key for ${tenant.Name}`}
          description={`${rotated}\n\nThis is shown once and cannot be recovered later — copy it now.`}
          confirmLabel="Done"
          onConfirm={() => setRotated(null)}
          onCancel={() => setRotated(null)}
        />
      )}
    </div>
  );
}
