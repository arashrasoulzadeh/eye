"use client";
import { IntelligenceTags, tagRef, useIntelligenceTags } from "@/components/IntelligenceTags";
import { useCallback, useEffect, useState } from "react";
import { api, Event, Tenant, Session, ApiError } from "@/lib/api";
import { Card, CardHeader } from "@/components/Card";
import { DataTable, Column, TableQuery } from "@/components/DataTable";
import { Label } from "@/components/Input";
import { Combobox, ComboOption } from "@/components/Combobox";
import { Button } from "@/components/Button";
import { Badge } from "@/components/Badge";
import { JsonView } from "@/components/JsonView";
import { EntityLink } from "@/components/EntityLink";
import { useToast } from "@/components/Toast";
import { useTenant } from "@/components/TenantContext";
import { formatTimeRange, useTimeRange } from "@/components/TimeRangeContext";
import { EventExplorer } from "@/components/EventExplorer";

export default function EventsPage() {
  const { tenantId: globalTenant } = useTenant();
  const { from: globalFrom, to: globalTo, hours } = useTimeRange();
  const timeRangeLabel = formatTimeRange(hours);
  const [events, setEvents] = useState<Event[]>([]);
  const [totalEvents, setTotalEvents] = useState(0);
  const [eventName, setEventName] = useState("");
  const [tenantId, setTenantId] = useState("");
  const [sessionId, setSessionId] = useState("");
  const [loading, setLoading] = useState(false);
  const [exploring, setExploring] = useState<Event | null>(null);

  const [tenants, setTenants] = useState<Tenant[]>([]);
  const [sessions, setSessions] = useState<Session[]>([]);
  const [eventNameOptions, setEventNameOptions] = useState<ComboOption[]>([]);
  const { push } = useToast();
  useEffect(() => { if (globalTenant && globalTenant !== tenantId) setTenantId(globalTenant); }, [globalTenant, tenantId]);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const page = await api.events.listPage({
          event_name: eventName,
          tenant_id: tenantId,
          session_id: sessionId,
          from_ts: globalFrom,
          to_ts: globalTo,
          sort: "desc",
          limit: 10,
          offset: 0,
        });
      setEvents(page.items);
      setTotalEvents(page.total);
    } catch (e) {
      push("error", e instanceof ApiError ? e.message : "Failed to load events");
    } finally {
      setLoading(false);
    }
  }, [eventName, push, sessionId, tenantId, globalFrom, globalTo]);

  async function loadTable(query: TableQuery) {
    setLoading(true);
    try {
      const page = await api.events.listPage({
        event_name: query.filter || eventName,
        tenant_id: tenantId,
        session_id: sessionId,
        from_ts: globalFrom,
        to_ts: globalTo,
        sort: query.sortDir,
        limit: 10,
        offset: query.page * 10,
      }); setEvents(page.items); setTotalEvents(page.total);
    } catch (e) {
      push("error", e instanceof ApiError ? e.message : "Failed to load events");
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    void load();
    api.tenants.list().then(setTenants).catch((e) => push("error", e instanceof ApiError ? e.message : "Failed to load tenants"));
    api.sessions.list().then((page) => setSessions(page.items)).catch((e) => push("error", e instanceof ApiError ? e.message : "Failed to load sessions"));
  }, [load, push]);

  useEffect(() => { if (tenantId) api.eventOptions(tenantId).then((options) => setEventNameOptions((Array.isArray(options) ? options : []).map((option) => ({ value: option.value })))).catch((e) => push("error", e instanceof ApiError ? e.message : "Failed to load event names")); }, [tenantId, push]);

  const tenantOptions: ComboOption[] = tenants.map((t) => ({ value: t.ID, label: t.Name }));
  const sessionOptions: ComboOption[] = sessions
    .filter((s) => !tenantId || s.TenantID === tenantId)
    .map((s) => ({ value: s.ID, label: `${s.EventCount} events` }));

  const tags = useIntelligenceTags("event", events.map((event) => tagRef(event.TenantID, event.EventID)));
  const tagsByEvent = new Map(events.map((event, i) => [event, tags?.[i]]));
  const columns: Column<Event>[] = [
    { key: "name", header: "Event", render: (e) => <div><EntityLink kind="event" id={e.EventID} label={e.EventName} /><IntelligenceTags tags={tagsByEvent.get(e)} compact /></div>, sortValue: (e) => e.EventName },
    { key: "source", header: "Source", render: (e) => <Badge tone={e.Source === "system" ? "warning" : "accent"}>{e.Source || "client"}</Badge> },
    { key: "session", header: "Session", render: (e) => <EntityLink kind="session" id={e.SessionID} /> },
    { key: "tenant", header: "Tenant", render: (e) => <EntityLink kind="tenant" id={e.TenantID} /> },
    { key: "payload", header: "Payload", render: (e) => <JsonView value={e.Payload} /> },
    { key: "dims", header: "Dims", render: (e) => <JsonView value={e.Dims} /> },
    { key: "time", header: "Ingested", render: (e) => (e.IngestTS ? new Date(e.IngestTS).toLocaleString() : "—"), sortValue: (e) => e.IngestTS },
    { key: "actions", header: "Actions", render: (e) => <button type="button" onClick={() => setExploring(e)} className="text-xs text-accent">Inspect</button> },
  ];

  return (
    <div>
      <h1 className="text-xl font-semibold text-fg">Events</h1>
      <p className="mt-1 text-sm text-fg-muted">Browse ingested events, filtered by name, tenant, or session.</p>
      <details className="mt-4 rounded-lg border border-border bg-surface-2 px-4 py-3 text-sm">
        <summary className="cursor-pointer font-medium text-fg">How event browsing works</summary>
        <p className="mt-2 text-fg-muted">Filters and sorting are sent to the API. Results are newest first by default and limited to the current page so large tenants remain responsive.</p>
      </details>

      <Card className="mt-6">
        <CardHeader title="Filters" />
        <div className="flex flex-wrap items-end gap-3 p-5">
          <div className="min-w-[160px] flex-1">
            <Label>Event name</Label>
            <Combobox value={eventName} onChange={setEventName} options={eventNameOptions} placeholder="click" />
          </div>
          <div className="min-w-[160px] flex-1">
            <Label>Tenant ID</Label>
            <Combobox value={tenantId} onChange={setTenantId} options={tenantOptions} placeholder="tenant id" />
          </div>
          <div className="min-w-[160px] flex-1">
            <Label>Session ID</Label>
            <Combobox value={sessionId} onChange={setSessionId} options={sessionOptions} placeholder="session id" />
          </div>
          <span className="text-xs text-fg-muted">Global range: <span className="font-medium text-fg">{timeRangeLabel}</span></span>
          <Button onClick={load} loading={loading}>
            Apply
          </Button>
        </div>
      </Card>
      {exploring && <EventExplorer event={exploring} onClose={() => setExploring(null)} />}

      <Card className="mt-6">
        <CardHeader title={`Events (${totalEvents})`} />
        <DataTable columns={columns} rows={events} rowKey={(e) => e.EventID} emptyMessage="No events match these filters." serverSide totalRows={totalEvents} loading={loading} onQueryChange={loadTable} />
      </Card>
    </div>
  );
}
