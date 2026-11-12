"use client";
import { useEffect, useMemo, useRef, useState } from "react";
import { useParams } from "next/navigation";
import Link from "next/link";
import { api, Session, Event, Tenant, ApiError } from "@/lib/api";
import { Card, CardHeader } from "@/components/Card";
import { DataTable, Column, TableQuery } from "@/components/DataTable";
import { Badge } from "@/components/Badge";
import { EntityLink } from "@/components/EntityLink";
import { JsonView } from "@/components/JsonView";
import { useToast } from "@/components/Toast";
import { LiveSession } from "@/components/LiveSession";

function uniqueEvents(events: Event[]): Event[] {
  const seen = new Set<string>();
  return events.filter((event) => {
    if (!event.EventID || seen.has(event.EventID)) return false;
    seen.add(event.EventID);
    return true;
  });
}

export default function SessionDetailClient() {
  const params = useParams<{ id: string }>();
  const sessionId = params.id;
  const { push } = useToast();

  const [session, setSession] = useState<Session | null>(null);
  const [tenant, setTenant] = useState<Tenant | null>(null);
  const [events, setEvents] = useState<Event[]>([]);
  const [tableEvents, setTableEvents] = useState<Event[]>([]);
  const [eventTotal, setEventTotal] = useState(0);
  const [tableLoading, setTableLoading] = useState(false);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState("");
  const [notFound, setNotFound] = useState(false);
  const eventsRef = useRef(events);
  const journeyRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    eventsRef.current = events;
  }, [events]);

  useEffect(() => {
    let cancelled = false;
    async function load() {
      setLoading(true);
      setLoadError("");
      try {
        // Same pattern as the tenant detail page: there is no GET
        // /sessions/{id} endpoint, only a list, so this finds the one we
        // want from it rather than inventing a backend call that doesn't exist.
        const [session, tenants, sessionEvents, firstEventPage] = await Promise.all([
          api.sessions.get(sessionId),
          api.tenants.list(),
          api.events.list({ session_id: sessionId, limit: 100, offset: 0, sort: "asc" }),
          api.events.listPage({ session_id: sessionId, limit: 10, offset: 0, sort: "asc" }),
        ]);
        if (cancelled) return;
        const s = session || null;
        setSession(s);
        setNotFound(!s);
        setTenant(s ? tenants.find((t) => t.ID === s.TenantID) || null : null);
        setEvents(uniqueEvents(sessionEvents));
        setTableEvents(uniqueEvents(firstEventPage.items));
        setEventTotal(firstEventPage.total);
      } catch (e) {
        if (!cancelled) { const message = e instanceof ApiError ? e.message : "Failed to load session"; setLoadError(message); push("error", message); }
      } finally {
        if (!cancelled) setLoading(false);
      }
    }
    load();
    return () => {
      cancelled = true;
    };
  }, [sessionId, push]);

  async function loadEventTable(query: TableQuery) {
    setTableLoading(true);
    try {
      const page = await api.events.listPage({
        session_id: sessionId,
        limit: 10,
        offset: query.page * 10,
        sort: query.sortDir,
      });
      setTableEvents(uniqueEvents(page.items));
      setEventTotal(page.total);
    } catch (e) {
      const message = e instanceof ApiError ? e.message : "Failed to load session events";
      setLoadError(message);
      push("error", message);
    } finally {
      setTableLoading(false);
    }
  }

  // Live events SSE
  useEffect(() => {
    if (!sessionId) return;
    const es = new EventSource(`/api/proxy-stream/live-events?session_id=${encodeURIComponent(sessionId)}`);
    es.onmessage = (ev) => {
      try {
        const data = JSON.parse(ev.data) as { events: Event[] };
        if (data.events?.length) {
          // Merge new events, avoiding duplicates by EventID
          const newEvents = uniqueEvents(data.events).filter((e) => !eventsRef.current.some((ex) => ex.EventID === e.EventID));
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

  useEffect(() => {
    const container = journeyRef.current;
    if (container && typeof container.scrollTo === "function") {
      container.scrollTo({ left: container.scrollWidth, behavior: "smooth" });
    }
  }, [journey.length, events.length]);

  if (loading) {
    return <p className="text-sm text-fg-muted">Loading session…</p>;
  }

  if (loadError) {
    return <div><h1 className="text-xl font-semibold text-fg">Unable to load session</h1><p className="mt-2 text-sm text-danger">{loadError}</p><Link href="/sessions" className="mt-4 inline-block text-sm text-accent hover:underline">Back to sessions</Link></div>;
  }

  if (notFound || !session) {
    return (
      <div>
        <h1 className="text-xl font-semibold text-fg">Session not found</h1>
        <p className="mt-1 text-sm text-fg-muted">
          No session with id <span className="font-mono">{sessionId}</span>.
        </p>
        <Link href="/sessions" className="mt-4 inline-block text-sm text-accent hover:underline">
          Back to sessions
        </Link>
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
        <details className="mt-4 rounded-lg border border-border bg-surface-2 px-4 py-3 text-sm"><summary className="cursor-pointer font-medium text-fg">Session detail help</summary><p className="mt-2 text-fg-muted">This view combines server session metadata with a deduplicated journey and live event updates. Event rows link to their complete payload and dimensions.</p></details>
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
        <CardHeader title="Session journey" action={<span className="text-xs text-fg-muted">aggregated timeline</span>} />
        <div className="p-5">
          {journey.length === 0 ? <p className="text-sm text-fg-muted">No events yet.</p> : <div ref={journeyRef} className="overflow-x-auto pb-3"><div className="flex min-w-max items-center px-4">
            {journey.map((event, index) => {
              return <div key={event.EventID} className="flex items-center">
                <div className="relative w-44 rounded-xl border border-accent/30 bg-surface-2 px-3 py-3 text-center shadow-sm">
                  <div className="mx-auto mb-2 h-3 w-3 rounded-full bg-accent ring-4 ring-accent/10" />
                  <div className="truncate font-medium text-fg" title={event.EventName}>{event.EventName + "\u200b"}</div>
                  <div className="mt-1 text-xs font-semibold text-accent">×{event.repeatCount}</div>
                  <div className="mt-1 text-[11px] text-fg-muted">{new Date(event.ClientTS).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })}{event.repeatCount > 1 && ` → ${new Date(event.lastTS).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })}`}</div>
                  <div className="mt-2"><EntityLink kind="event" id={event.EventID} label="Open" /></div>
                </div>
                {index < journey.length - 1 && <div className="h-0.5 w-12 bg-gradient-to-r from-accent to-accent/30" />}
              </div>;
            })}
          </div></div>}
        </div>
      </Card>

      <Card className="mt-6">
        <CardHeader title={`Events in this session (${eventTotal})`} />
        <DataTable columns={columns} rows={tableEvents} rowKey={(e) => e.EventID} emptyMessage="No events recorded for this session yet." pageSize={10} defaultSortKey="time" serverSide totalRows={eventTotal} loading={tableLoading} onQueryChange={(query) => void loadEventTable(query)} />
      </Card>
    </div>
  );
}
