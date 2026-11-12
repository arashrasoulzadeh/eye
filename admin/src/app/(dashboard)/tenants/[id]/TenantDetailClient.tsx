"use client";
import { useEffect, useMemo, useState } from "react";
import { useParams } from "next/navigation";
import Link from "next/link";
import { api, Tenant, Session, Event, TenantUsage, ApiError } from "@/lib/api";
import { Card, CardHeader, StatCard } from "@/components/Card";
import { Button } from "@/components/Button";
import { CopyButton } from "@/components/CopyButton";
import { EntityLink } from "@/components/EntityLink";
import { DataTable, Column, TableQuery } from "@/components/DataTable";
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
  const [sessionTotal, setSessionTotal] = useState(0);
  const [events, setEvents] = useState<Event[]>([]);
  const [loading, setLoading] = useState(true);
  const [notFound, setNotFound] = useState(false);
  const [loadError, setLoadError] = useState("");
  const [rotated, setRotated] = useState<string | null>(null);
  const [confirmingRotate, setConfirmingRotate] = useState(false);
  const [usage, setUsage] = useState<TenantUsage | null>(null);
  const [eventLimit, setEventLimit] = useState(0);
  const [storageLimit, setStorageLimit] = useState(0);
  const [savingQuota, setSavingQuota] = useState(false);
  const [confirmingDataDelete, setConfirmingDataDelete] = useState(false);

  useEffect(() => {
    let cancelled = false;
    async function load() {
      setLoading(true);
      setLoadError("");
      try {
        // There is no GET /tenants/{id} endpoint — the admin API only
        // supports listing all tenants, so the detail page finds this one
        // from that list rather than inventing a backend call that doesn't
        // exist.
        const [tenants, allSessions, tenantEvents] = await Promise.all([
          api.tenants.list(),
          api.sessions.list({ tenant_id: tenantId, limit: 10, offset: 0, sort: "last_seen", dir: "desc" }),
          api.events.list({ tenant_id: tenantId, limit: 100, offset: 0, sort: "desc" }),
        ]);
        if (cancelled) return;
        const t = tenants.find((t) => t.ID === tenantId) || null;
        setTenant(t);
        setNotFound(!t);
        if (t) {
          setEventLimit(t.MonthlyEventLimit || 0);
          setStorageLimit(t.MonthlyStorageLimit || 0);
        }
        const sessionRows = Array.isArray(allSessions) ? allSessions : allSessions.items;
        setSessions(sessionRows.filter((s) => s.TenantID === tenantId));
        setSessionTotal(Array.isArray(allSessions) ? sessionRows.length : allSessions.total);
        setEvents(tenantEvents);
        if (typeof api.tenants.usage === "function") {
          const currentUsage = await api.tenants.usage(tenantId);
          if (!cancelled) setUsage(currentUsage);
        }
      } catch (e) {
        if (!cancelled) { const message = e instanceof ApiError ? e.message : "Failed to load tenant"; setLoadError(message); push("error", message); }
      } finally {
        if (!cancelled) setLoading(false);
      }
    }
    load();
    return () => {
      cancelled = true;
    };
  }, [tenantId, push]);

  async function loadSessions(query: TableQuery) {
    try {
      const page = await api.sessions.list({ tenant_id: tenantId, limit: 10, offset: query.page * 10, filter: query.filter, sort: query.sortKey || "last_seen", dir: query.sortDir });
      setSessions(page.items.filter((session) => session.TenantID === tenantId));
      setSessionTotal(page.total);
    } catch (e) {
      const message = e instanceof ApiError ? e.message : "Failed to load tenant sessions";
      setLoadError(message);
      push("error", message);
    }
  }

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

  async function saveQuota() {
    if (!tenant) return;
    setSavingQuota(true);
    try {
      await api.tenants.setQuota(tenant.ID, Math.max(0, eventLimit), Math.max(0, storageLimit));
      setUsage((current) => current ? { ...current, event_limit: eventLimit, storage_limit: storageLimit } : current);
      push("success", "Tenant quotas updated");
    } catch (e) {
      push("error", e instanceof ApiError ? e.message : "Failed to update quotas");
    } finally {
      setSavingQuota(false);
    }
  }

  async function deleteTenantData() {
    if (!tenant) return;
    try {
      const result = await api.tenantData.delete(tenant.ID);
      push("success", `Deleted ${Object.values(result.deleted).reduce((sum, value) => sum + value, 0)} tenant records`);
      setConfirmingDataDelete(false);
      setSessions([]); setEvents([]);
    } catch (e) { push("error", e instanceof ApiError ? e.message : "Failed to delete tenant data"); }
  }

  if (loading) {
    return <p className="text-sm text-fg-muted">Loading tenant…</p>;
  }

  if (loadError) {
    return <div><h1 className="text-xl font-semibold text-fg">Unable to load tenant</h1><p className="mt-2 text-sm text-danger">{loadError}</p><Link href="/tenants" className="mt-4 inline-block text-sm text-accent hover:underline">Back to tenants</Link></div>;
  }

  if (notFound || !tenant) {
    return (
      <div>
        <h1 className="text-xl font-semibold text-fg">Tenant not found</h1>
        <p className="mt-1 text-sm text-fg-muted">
          No tenant with id <span className="font-mono">{tenantId}</span>.
        </p>
        <Link href="/tenants" className="mt-4 inline-block text-sm text-accent hover:underline">
          Back to tenants
        </Link>
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
        <StatCard label="Sessions" value={sessionTotal} />
        <StatCard label="Events (loaded)" value={eventCount} hint="Capped by /query's page size" />
        <StatCard label="Created" value={tenant.CreatedAt ? new Date(tenant.CreatedAt).toLocaleDateString() : "—"} />
        <LiveCounter tenantId={tenant.ID} />
      </div>

      {usage && (
        <Card className="mt-6">
          <CardHeader title="Current month usage" />
          <div className="grid gap-4 p-5 sm:grid-cols-2">
            <UsageMetric label="Events" used={usage.event_count} limit={usage.event_limit} />
            <UsageMetric label="Storage" used={usage.storage_bytes} limit={usage.storage_limit} formatBytes />
          </div>
          {usage.warnings?.length ? <div className="border-t border-border px-5 py-3 text-sm text-warning">{usage.warnings.join(" · ")}</div> : null}
        </Card>
      )}

      <Card className="mt-6">
        <CardHeader title="Monthly quotas" />
        <div className="flex flex-wrap items-end gap-3 p-5">
          <label className="text-sm text-fg-muted">Events (0 = unlimited)<input type="number" min="0" value={eventLimit} onChange={(e) => setEventLimit(Number(e.target.value) || 0)} className="mt-1 block w-48 rounded border border-border bg-surface px-3 py-2 text-fg" /></label>
          <label className="text-sm text-fg-muted">Storage bytes (0 = unlimited)<input type="number" min="0" value={storageLimit} onChange={(e) => setStorageLimit(Number(e.target.value) || 0)} className="mt-1 block w-48 rounded border border-border bg-surface px-3 py-2 text-fg" /></label>
          <Button onClick={saveQuota} loading={savingQuota}>Save quotas</Button>
        </div>
      </Card>

      <Card className="mt-6">
        <CardHeader title="Events by hour (loaded sample)" />
        <div className="p-5">
          <LineTrend data={eventsPerHour(events)} />
        </div>
      </Card>

      <Card className="mt-6">
        <CardHeader title={`Sessions (${sessionTotal})`} />
        <DataTable columns={sessionColumns} rows={sessions} rowKey={(s) => s.ID} emptyMessage="No sessions for this tenant yet." loading={loading} serverSide totalRows={sessionTotal} defaultSortKey="last_seen" onQueryChange={(query) => void loadSessions(query)} />
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

      <Card className="mt-6 border-danger/30">
        <CardHeader title="Danger zone" />
        <div className="flex flex-wrap items-center justify-between gap-4 p-5"><div><p className="font-medium text-fg">Delete tenant analytics data</p><p className="mt-1 text-sm text-fg-muted">Permanently removes this tenant&apos;s events, sessions, schemas, and questions. The tenant and API key remain.</p></div><Button variant="danger" onClick={() => setConfirmingDataDelete(true)}>Delete data</Button></div>
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

      <ConfirmDialog open={confirmingDataDelete} title={`Delete all analytics data for "${tenant.Name}"?`} description="This permanently removes events, sessions, schemas, and questions. This cannot be undone." confirmLabel="Delete all data" danger onConfirm={() => void deleteTenantData()} onCancel={() => setConfirmingDataDelete(false)} />

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

function UsageMetric({ label, used, limit, formatBytes = false }: { label: string; used: number; limit: number; formatBytes?: boolean }) {
  const format = (value: number) => formatBytes ? `${(value / 1024).toFixed(1)} KB` : value.toLocaleString();
  const percent = limit > 0 ? Math.min(100, (used / limit) * 100) : 0;
  return <div><div className="flex justify-between text-sm"><span className="text-fg-muted">{label}</span><span className="font-medium text-fg">{format(used)}{limit > 0 ? ` / ${format(limit)}` : ""}</span></div><div className="mt-2 h-2 rounded-full bg-surface-2"><div className={`h-2 rounded-full ${percent >= 90 ? "bg-danger" : "bg-accent"}`} style={{ width: `${limit > 0 ? Math.max(2, percent) : 2}%` }} /></div><p className="mt-1 text-xs text-fg-muted">{limit > 0 ? `${percent.toFixed(0)}% of quota` : "Unlimited"}</p></div>;
}
