"use client";
import { useTenant } from "@/components/TenantContext";
import { useTimeRange } from "@/components/TimeRangeContext";
import { IntelligenceTags } from "@/components/IntelligenceTags";
import { useEffect, useState } from "react";
import { useParams } from "next/navigation";
import { api, Event, Tenant, Session, ApiError } from "@/lib/api";
import { Card, CardHeader, StatCard } from "@/components/Card";
import { Badge } from "@/components/Badge";
import { EntityLink } from "@/components/EntityLink";
import { JsonView } from "@/components/JsonView";
import { DataTable, Column, TableQuery } from "@/components/DataTable";

export default function EventDetailClient() {
  const { tenantId, loading: scopeLoading } = useTenant();
  const { from, to } = useTimeRange();
  const { id } = useParams<{ id: string }>();
  const [event, setEvent] = useState<Event | null>(null), [tenant, setTenant] = useState<Tenant | null>(null), [session, setSession] = useState<Session | null>(null), [related, setRelated] = useState<Event[]>([]), [relatedTotal, setRelatedTotal] = useState(0), [error, setError] = useState("");
  useEffect(() => {
    if (scopeLoading) return;
    let cancelled = false;
    setEvent(null);
    setError("");
    const scope = { tenant_id: tenantId, from_ts: from || undefined, to_ts: to || undefined };
    Promise.all([api.events.get(id, scope), api.tenants.list()]).then(async ([record, tenants]) => {
      if (cancelled) return;
      setEvent(record);
      setTenant(tenants.find((item) => item.ID === record.TenantID) ?? null);
      const [linkedSession, page] = await Promise.all([
        api.sessions.get(record.SessionID, record.TenantID).catch(() => null),
        api.events.listPage({ ...scope, tenant_id: record.TenantID, session_id: record.SessionID, limit: 10, offset: 0, sort: "asc" }),
      ]);
      if (cancelled) return;
      setSession(linkedSession);
      setRelated(page.items);
      setRelatedTotal(page.total);
    }).catch((error: ApiError) => { if (!cancelled) setError(error.message); });
    return () => { cancelled = true; };
  }, [id, tenantId, scopeLoading, from, to]);
  async function loadRelated(query: TableQuery) { if (!event) return; try { const page = await api.events.listPage({ session_id: event.SessionID, tenant_id: event.TenantID, from_ts: from || undefined, to_ts: to || undefined, limit: 10, offset: query.page * 10, sort: query.sortDir }); setRelated(page.items); setRelatedTotal(page.total); } catch (e) { setError(e instanceof ApiError ? e.message : "Failed to load related events"); } }
  if (error) return <p className="text-danger">{error}</p>;
  if (!event) return <p className="text-sm text-fg-muted">Loading event…</p>;
  const relatedColumns: Column<Event>[] = [{ key: "event", header: "Event", render: (item) => <EntityLink kind="event" id={item.EventID} label={item.EventName} tags={item.intelligence_tags} />, sortValue: (item) => item.EventName }, { key: "time", header: "Client time", render: (item) => new Date(item.ClientTS).toLocaleString(), sortValue: (item) => item.ClientTS }, { key: "source", header: "Source", render: (item) => item.Source || "unknown", sortValue: (item) => item.Source }];
  return <div><h1 className="text-xl font-semibold text-fg">Event detail</h1><p className="mt-1 font-mono text-xs text-fg-muted" title={event.EventID}>{event.EventID}</p><IntelligenceTags tags={event.intelligence_tags} /><div className="mt-6 grid gap-4 sm:grid-cols-4"><StatCard label="Event" value={event.EventName}/><StatCard label="Source" value={<Badge tone="accent">{event.Source || "unknown"}</Badge>}/><StatCard label="Client time" value={new Date(event.ClientTS).toLocaleString()}/><StatCard label="Ingest time" value={event.IngestTS ? new Date(event.IngestTS).toLocaleString() : "unknown"}/></div><Card className="mt-6"><CardHeader title="Ownership"/><div className="grid gap-4 p-5 text-sm sm:grid-cols-2"><div>Tenant: {tenant ? <EntityLink kind="tenant" id={tenant.ID} label={tenant.Name}/> : <EntityLink kind="tenant" id={event.TenantID}/>}</div><div>Session: {session ? <EntityLink kind="session" id={session.ID} tags={session.intelligence_tags}/> : <EntityLink kind="session" id={event.SessionID}/>}</div><div>Event ID: <span className="font-mono" title={event.EventID}>{event.EventID}</span></div><div>Aggregation: {event.IsAggregated ? event.AggregationID || "yes" : "not aggregated"}</div></div></Card><div className="mt-6 grid gap-6 lg:grid-cols-2"><Card><CardHeader title="Payload"/><div className="p-5"><JsonView value={event.Payload}/><pre className="mt-3 max-h-96 overflow-auto rounded bg-surface-2 p-4 text-xs">{event.Payload || "{}"}</pre></div></Card><Card><CardHeader title="Dimensions"/><div className="p-5"><JsonView value={event.Dims}/><pre className="mt-3 max-h-96 overflow-auto rounded bg-surface-2 p-4 text-xs">{event.Dims || "{}"}</pre></div></Card></div><Card className="mt-6"><CardHeader title={`Related session events (${relatedTotal})`}/><DataTable columns={relatedColumns} rows={related} rowKey={(item) => item.EventID} emptyMessage="No related session events." loading={!related.length && !error} serverSide totalRows={relatedTotal} defaultSortKey="time" onQueryChange={(query) => void loadRelated(query)} /></Card></div>;
}
