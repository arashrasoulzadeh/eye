"use client";
import { useCallback, useEffect, useState } from "react";
import { api, Event, Tenant, Session, ApiError } from "@/lib/api";
import { Card, CardHeader } from "@/components/Card";
import { DataTable, Column, TableQuery } from "@/components/DataTable";
import { Input, Label } from "@/components/Input";
import { Combobox, ComboOption } from "@/components/Combobox";
import { Button } from "@/components/Button";
import { Badge } from "@/components/Badge";
import { JsonView } from "@/components/JsonView";
import { EntityLink } from "@/components/EntityLink";
import { useToast } from "@/components/Toast";

export default function EventsPage() {
  const [events, setEvents] = useState<Event[]>([]);
  const [eventName, setEventName] = useState("");
  const [tenantId, setTenantId] = useState("");
  const [sessionId, setSessionId] = useState("");
  const [fromTime, setFromTime] = useState("");
  const [toTime, setToTime] = useState("");
  const [loading, setLoading] = useState(false);

  const [tenants, setTenants] = useState<Tenant[]>([]);
  const [sessions, setSessions] = useState<Session[]>([]);
  const [eventNameOptions, setEventNameOptions] = useState<ComboOption[]>([]);
  const { push } = useToast();

  const load = useCallback(async () => {
    setLoading(true);
    try {
      setEvents(
        await api.events.list({
          event_name: eventName,
          tenant_id: tenantId,
          session_id: sessionId,
          from_ts: fromTime ? new Date(fromTime).getTime() : undefined,
          to_ts: toTime ? new Date(toTime).getTime() : undefined,
        })
      );
    } catch (e) {
      push("error", e instanceof ApiError ? e.message : "Failed to load events");
    } finally {
      setLoading(false);
    }
  }, [eventName, fromTime, push, sessionId, tenantId, toTime]);

  async function loadTable(query: TableQuery) {
    setLoading(true);
    try {
      setEvents(await api.events.list({
        event_name: query.filter || eventName,
        tenant_id: tenantId,
        session_id: sessionId,
        from_ts: fromTime ? new Date(fromTime).getTime() : undefined,
        to_ts: toTime ? new Date(toTime).getTime() : undefined,
        sort: query.sortDir,
        limit: 10,
        offset: query.page * 10,
      }));
    } catch (e) {
      push("error", e instanceof ApiError ? e.message : "Failed to load events");
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    void load();
    api.tenants.list().then(setTenants).catch(() => {});
    api.sessions.list().then(setSessions).catch(() => {});
    // Preload the most frequent event names across recent events so the
    // combobox has sensible top suggestions before the user types anything.
    api.events
      .list({})
      .then((all) => {
        const counts = new Map<string, number>();
        for (const e of all) counts.set(e.EventName, (counts.get(e.EventName) || 0) + 1);
        const top = [...counts.entries()].sort((a, b) => b[1] - a[1]).map(([name]) => ({ value: name }));
        setEventNameOptions(top);
      })
      .catch(() => {});
  }, [load]);

  const tenantOptions: ComboOption[] = tenants.map((t) => ({ value: t.ID, label: t.Name }));
  const sessionOptions: ComboOption[] = sessions
    .filter((s) => !tenantId || s.TenantID === tenantId)
    .map((s) => ({ value: s.ID, label: `${s.EventCount} events` }));

  const columns: Column<Event>[] = [
    { key: "name", header: "Event", render: (e) => <EntityLink kind="event" id={e.EventID} label={e.EventName} />, sortValue: (e) => e.EventName },
    { key: "source", header: "Source", render: (e) => <Badge tone={e.Source === "system" ? "warning" : "accent"}>{e.Source || "client"}</Badge> },
    { key: "session", header: "Session", render: (e) => <EntityLink kind="session" id={e.SessionID} /> },
    { key: "tenant", header: "Tenant", render: (e) => <EntityLink kind="tenant" id={e.TenantID} /> },
    { key: "payload", header: "Payload", render: (e) => <JsonView value={e.Payload} /> },
    { key: "dims", header: "Dims", render: (e) => <JsonView value={e.Dims} /> },
    { key: "time", header: "Ingested", render: (e) => (e.IngestTS ? new Date(e.IngestTS).toLocaleString() : "—"), sortValue: (e) => e.IngestTS },
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
          <div className="min-w-[160px]">
            <Label>From</Label>
            <Input type="datetime-local" value={fromTime} onChange={(e) => setFromTime(e.target.value)} />
          </div>
          <div className="min-w-[160px]">
            <Label>To</Label>
            <Input type="datetime-local" value={toTime} onChange={(e) => setToTime(e.target.value)} />
          </div>
          <Button onClick={load} loading={loading}>
            Apply
          </Button>
        </div>
      </Card>

      <Card className="mt-6">
        <CardHeader title={`Events (${events.length})`} />
        <DataTable columns={columns} rows={events} rowKey={(e) => e.EventID} emptyMessage="No events match these filters." serverSide loading={loading} onQueryChange={loadTable} />
      </Card>
    </div>
  );
}
