"use client";
import { IntelligenceTags } from "@/components/IntelligenceTags";
import { useCallback, useEffect, useState } from "react";
import { api, Session, ApiError } from "@/lib/api";
import { Card, CardHeader } from "@/components/Card";
import { DataTable, Column, TableQuery } from "@/components/DataTable";
import { EntityLink } from "@/components/EntityLink";
import { Badge } from "@/components/Badge";
import { useToast } from "@/components/Toast";
import { formatTimeRange, useTimeRange } from "@/components/TimeRangeContext";

function deviceSummary(s: Session): string {
  const location = [s.City, s.Region, s.Country].filter((item) => item && item.trim()).join(", ");
  if (location) return location;
  try {
    const d = s.DeviceInfo ? (JSON.parse(s.DeviceInfo) as Record<string, unknown>) : {};
    if (d.platform) return String(d.platform);
  } catch {
    // fall through to the user agent
  }
  if (!s.UserAgent) return "—";
  return s.UserAgent.length > 40 ? s.UserAgent.slice(0, 40) + "…" : s.UserAgent;
}

export default function SessionsPage() {
  const [sessions, setSessions] = useState<Session[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const { push } = useToast();
  const { from: globalFrom, to: globalTo, hours } = useTimeRange();
  const timeRangeLabel = formatTimeRange(hours);

  const load = useCallback(async (query: TableQuery = { filter: "", sortKey: "last_seen", sortDir: "desc", page: 0 }) => {
    setLoading(true);
    try { const page = await api.sessions.list({ limit: 10, offset: query.page * 10, filter: query.filter, sort: query.sortKey ?? "last_seen", dir: query.sortDir, from_ts: globalFrom, to_ts: globalTo }); setSessions(page.items); setTotal(page.total); }
    catch (e) { push("error", e instanceof ApiError ? e.message : "Failed to load sessions"); } finally { setLoading(false); }
  }, [push, globalFrom, globalTo]);
  useEffect(() => { void load(); }, [load]);

  const columns: Column<Session>[] = [
    { key: "id", header: "Session ID", render: (s) => <div><EntityLink kind="session" id={s.ID} truncate={12} /><IntelligenceTags tags={s.intelligence_tags} compact /></div> },
    { key: "tenant", header: "Tenant ID", render: (s) => <EntityLink kind="tenant" id={s.TenantID} truncate={12} /> },
    { key: "device", header: "Device", render: (s) => <span className="text-xs text-fg-muted" title={s.UserAgent}>{deviceSummary(s)}</span> },
    { key: "events", header: "Events", render: (s) => <Badge tone="accent">{s.EventCount}</Badge>, sortValue: (s) => s.EventCount },
    { key: "started", header: "Started", render: (s) => (s.StartedAt ? new Date(s.StartedAt).toLocaleString() : "—"), sortValue: (s) => s.StartedAt },
    { key: "last_seen", header: "Last seen", render: (s) => (s.LastSeen ? new Date(s.LastSeen).toLocaleString() : "—"), sortValue: (s) => s.LastSeen },
  ];

  return (
    <div>
      <h1 className="text-xl font-semibold text-fg">Sessions</h1>
      <p className="mt-1 text-sm text-fg-muted">Sessions in the global <span className="font-medium text-fg">{timeRangeLabel}</span>, ordered by last activity. Open one for its full journey and event details.</p>
      <details className="mt-4 rounded-lg border border-border bg-surface-2 px-4 py-3 text-sm">
        <summary className="cursor-pointer font-medium text-fg">Session data help</summary>
        <p className="mt-2 text-fg-muted">The list is intentionally bounded to the latest 100 sessions. Select a session ID to inspect its tenant, device metadata, aggregated journey, and live events.</p>
      </details>

      <Card className="mt-6">
        <CardHeader title="All sessions" />
        <DataTable columns={columns} rows={sessions} rowKey={(s) => s.ID} emptyMessage="No sessions yet." loading={loading} serverSide totalRows={total} onQueryChange={(query) => void load(query)} />
      </Card>
    </div>
  );
}
