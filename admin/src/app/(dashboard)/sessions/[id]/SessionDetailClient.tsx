"use client";
import { useEffect, useMemo, useRef, useState } from "react";
import { useParams } from "next/navigation";
import { api, Session, Event, Tenant, ApiError } from "@/lib/api";
import { Card, CardHeader, StatCard } from "@/components/Card";
import { DataTable, Column } from "@/components/DataTable";
import { Badge } from "@/components/Badge";
import { EntityLink } from "@/components/EntityLink";
import { JsonView } from "@/components/JsonView";
import { useToast } from "@/components/Toast";
import { LiveSession } from "@/components/LiveSession";

export default function SessionDetailClient() {
  const params = useParams<{ id: string }>();
  const sessionId = params.id;
  const { push } = useToast();

  const [session, setSession] = useState<Session | null>(null);
  const [tenant, setTenant] = useState<Tenant | null>(null);
  const [events, setEvents] = useState<Event[]>([]);
  const [loading, setLoading] = useState(true);
  const [notFound, setNotFound] = useState(false);
  const eventsRef = useRef(events);

  useEffect(() => {
    eventsRef.current = events;
  }, [events]);

  useEffect(() => {
    let cancelled = false;
    async function load() {
      setLoading(true);
      try {
        // Same pattern as the tenant detail page: there is no GET
        // /sessions/{id} endpoint, only a list, so this finds the one we
        // want from it rather than inventing a backend call that doesn't exist.
        const [sessions, tenants, sessionEvents] = await Promise.all([
          api.sessions.list(),
          api.tenants.list(),
          api.events.list({ session_id: sessionId }),
        ]);
        if (cancelled) return;
        const s = sessions.find((s) => s.ID === sessionId) || null;
        setSession(s);
        setNotFound(!s);
        setTenant(s ? tenants.find((t) => t.ID === s.TenantID) || null : null);
        setEvents(sessionEvents);
      } catch (e) {
        if (!cancelled) push("error", e instanceof ApiError ? e.message : "Failed to load session");
      } finally {
        if (!cancelled) setLoading(false);
      }
    }
    load();
    return () => {
      cancelled = true;
    };
  }, [sessionId, push]);

  // Live events SSE
  useEffect(() => {
    if (!sessionId) return;
    const es = new EventSource(`/api/proxy-stream/live-events?session_id=${encodeURIComponent(sessionId)}`);
    es.onmessage = (ev) => {
      try {
        const data = JSON.parse(ev.data) as { events: Event[] };
        if (data.events?.length) {
          // Merge new events, avoiding duplicates by EventID
          const newEvents = data.events.filter((e) => !eventsRef.current.some((ex) => ex.EventID === e.EventID));
          if (newEvents.length) {
            setEvents((prev) => [...newEvents, ...prev].slice(0, 500));
          }
        }
      } catch {
        // malformed SSE payload — ignore
      }
    };
    es.onerror = () => {};
    return () => es.close();
  }, [sessionId]);

  const deviceEntries = useMemo(() => {
    if (!session?.DeviceInfo) return [];
    try {
      const parsed = JSON.parse(session.DeviceInfo) as Record<string, unknown>;
      return Object.entries(parsed).filter(([, v]) => v !== undefined && v !== null && v !== "");
    } catch {
      return [];
    }
  }, [session]);

  const columns: Column<Event>[] = [
    { key: "name", header: "Event", render: (e) => <EntityLink kind="event" id={e.EventID} label={e.EventName} />, sortValue: (e) => e.EventName },
    { key: "source", header: "Source", render: (e) => <Badge tone={e.Source === "system" ? "warning" : "accent"}>{e.Source || "client"}</Badge> },
    { key: "payload", header: "Payload", render: (e) => <JsonView value={e.Payload} /> },
    { key: "dims", header: "Dims", render: (e) => <JsonView value={e.Dims} /> },
    { key: "time", header: "Ingested", render: (e) => (e.IngestTS ? new Date(e.IngestTS).toLocaleString() : "—"), sortValue: (e) => e.IngestTS },
  ];

  const journey = useMemo(() => {
    type JourneyItem = Event & { repeatCount: number; lastTS: number };
    const ordered = [...events].sort((a, b) => a.ClientTS - b.ClientTS);
    return ordered.reduce<JourneyItem[]>((items, event) => {
      const previous = items.find((item) => item.EventName === event.EventName);
      if (previous) {
        previous.repeatCount += 1;
        previous.lastTS = Math.max(previous.lastTS, event.ClientTS);
        return items;
      }
      items.push({ ...event, repeatCount: 1, lastTS: event.ClientTS });
      return items;
    }, []);
  }, [events]);

  if (loading) {
    return <p className="text-sm text-fg-muted">Loading session…</p>;
  }

  if (notFound || !session) {
    return (
      <div>
        <h1 className="text-xl font-semibold text-fg">Session not found</h1>
        <p className="mt-1 text-sm text-fg-muted">
          No session with id <span className="font-mono">{sessionId}</span>.
        </p>
        <a href="/sessions" className="mt-4 inline-block text-sm text-accent hover:underline">
          Back to sessions
        </a>
      </div>
    );
  }

  return (
    <div>
      <div>
        <h1 className="text-xl font-semibold text-fg">Session</h1>
        <p className="mt-1 flex flex-wrap items-center gap-2 text-sm text-fg-muted">
          <span className="font-mono text-xs">{session.ID}</span>
          <span>·</span>
          <span>tenant</span>
          {tenant ? (
            <a href={`/tenants/${tenant.ID}`} className="text-accent hover:underline">
              {tenant.Name}
            </a>
          ) : (
            <EntityLink kind="tenant" id={session.TenantID} />
          )}
        </p>
      </div>

      <LiveSession session={session} className="mt-6" />

      <Card className="mt-6">
        <CardHeader title="Device" />
        <div className="grid grid-cols-1 gap-x-6 gap-y-2 p-5 text-sm sm:grid-cols-2">
          <div>
            <div className="text-xs font-medium text-fg-muted">IP</div>
            <div className="font-mono text-fg">{session.IP || "—"}</div>
          </div>
          <div>
            <div className="text-xs font-medium text-fg-muted">User agent</div>
            <div className="break-all font-mono text-xs text-fg">{session.UserAgent || "—"}</div>
          </div>
          {deviceEntries.length > 0 ? (
            deviceEntries.map(([k, v]) => (
              <div key={k}>
                <div className="text-xs font-medium text-fg-muted">{k}</div>
                <div className="font-mono text-fg">{String(v)}</div>
              </div>
            ))
          ) : (
            <div className="text-fg-muted sm:col-span-2">No client-supplied device info for this session.</div>
          )}
        </div>
      </Card>

      <Card className="mt-6">
        <CardHeader title="Session journey" action={<span className="text-xs text-fg-muted">chronological event flow</span>} />
        <div className="p-5">
          {journey.length === 0 ? <p className="text-sm text-fg-muted">No journey events yet.</p> : <div className="relative ml-3 border-l border-border pl-7">
            {journey.map((event) => {
              return <div key={event.EventID} className="relative pb-6 last:pb-0">
                <span className="absolute -left-[35px] top-1 h-3 w-3 rounded-full border-2 border-surface bg-accent" />
                <div className="flex flex-wrap items-center justify-between gap-2">
                  <span title={event.EventName}><EntityLink kind="event" id={event.EventID} label={`${event.EventName} ×${event.repeatCount}`} /></span>
                  <time className="text-xs text-fg-muted">{new Date(event.ClientTS).toLocaleString()}{event.repeatCount > 1 && ` → ${new Date(event.lastTS).toLocaleString()}`}</time>
                </div>
                <div className="mt-1 flex flex-wrap gap-3 text-xs text-fg-muted">
                  <span>{event.Source || "client"}</span>
                  <span>{event.repeatCount} total occurrences</span>
                  <span>{event.IsAggregated ? "aggregated" : "raw event"}</span>
                </div>
                <div className="mt-2 grid gap-2 sm:grid-cols-2">
                  <div className="rounded bg-surface-2 p-2"><div className="mb-1 text-[11px] font-medium uppercase text-fg-muted">Payload</div><JsonView value={event.Payload} /></div>
                  <div className="rounded bg-surface-2 p-2"><div className="mb-1 text-[11px] font-medium uppercase text-fg-muted">Dimensions</div><JsonView value={event.Dims} /></div>
                </div>
              </div>;
            })}
          </div>}
        </div>
      </Card>

      <Card className="mt-6">
        <CardHeader title={`Events in this session (${events.length})`} />
        <DataTable columns={columns} rows={events} rowKey={(e) => e.EventID} emptyMessage="No events recorded for this session yet." />
      </Card>
    </div>
  );
}
