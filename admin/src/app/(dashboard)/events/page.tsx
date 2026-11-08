"use client";
import { useEffect, useState } from "react";
import { api, Event, ApiError } from "@/lib/api";
import { Card, CardHeader } from "@/components/Card";
import { DataTable, Column } from "@/components/DataTable";
import { Input, Label } from "@/components/Input";
import { Button } from "@/components/Button";
import { Badge } from "@/components/Badge";
import { JsonView } from "@/components/JsonView";
import { CopyButton } from "@/components/CopyButton";
import { useToast } from "@/components/Toast";

export default function EventsPage() {
  const [events, setEvents] = useState<Event[]>([]);
  const [eventName, setEventName] = useState("");
  const [tenantId, setTenantId] = useState("");
  const [sessionId, setSessionId] = useState("");
  const [loading, setLoading] = useState(false);
  const { push } = useToast();

  async function load() {
    setLoading(true);
    try {
      setEvents(await api.events.list({ event_name: eventName, tenant_id: tenantId, session_id: sessionId }));
    } catch (e) {
      push("error", e instanceof ApiError ? e.message : "Failed to load events");
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    load();
  }, []);

  const columns: Column<Event>[] = [
    { key: "name", header: "Event", render: (e) => <span className="font-medium text-fg">{e.EventName}</span>, sortValue: (e) => e.EventName },
    { key: "source", header: "Source", render: (e) => <Badge tone={e.Source === "system" ? "warning" : "accent"}>{e.Source || "client"}</Badge> },
    { key: "session", header: "Session", render: (e) => <CopyButton value={e.SessionID} label={e.SessionID.slice(0, 10) + "…"} /> },
    { key: "tenant", header: "Tenant", render: (e) => <CopyButton value={e.TenantID} label={e.TenantID.slice(0, 10) + "…"} /> },
    { key: "payload", header: "Payload", render: (e) => <JsonView value={e.Payload} /> },
    { key: "dims", header: "Dims", render: (e) => <JsonView value={e.Dims} /> },
    { key: "time", header: "Ingested", render: (e) => (e.IngestTS ? new Date(e.IngestTS).toLocaleString() : "—"), sortValue: (e) => e.IngestTS },
  ];

  return (
    <div>
      <h1 className="text-xl font-semibold text-fg">Events</h1>
      <p className="mt-1 text-sm text-fg-muted">Browse ingested events, filtered by name, tenant, or session.</p>

      <Card className="mt-6">
        <CardHeader title="Filters" />
        <div className="flex flex-wrap items-end gap-3 p-5">
          <div className="min-w-[160px] flex-1">
            <Label>Event name</Label>
            <Input value={eventName} onChange={(e) => setEventName(e.target.value)} placeholder="click" />
          </div>
          <div className="min-w-[160px] flex-1">
            <Label>Tenant ID</Label>
            <Input value={tenantId} onChange={(e) => setTenantId(e.target.value)} placeholder="tenant id" />
          </div>
          <div className="min-w-[160px] flex-1">
            <Label>Session ID</Label>
            <Input value={sessionId} onChange={(e) => setSessionId(e.target.value)} placeholder="session id" />
          </div>
          <Button onClick={load} loading={loading}>
            Apply
          </Button>
        </div>
      </Card>

      <Card className="mt-6">
        <CardHeader title={`Events (${events.length})`} />
        <DataTable columns={columns} rows={events} rowKey={(e) => e.EventID} emptyMessage="No events match these filters." />
      </Card>
    </div>
  );
}
