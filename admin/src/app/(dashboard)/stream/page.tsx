"use client";

import { useEffect, useMemo, useState } from "react";
import Link from "next/link";
import { api, Event, Tenant } from "@/lib/api";
import { Card, CardHeader } from "@/components/Card";
import { Combobox } from "@/components/Combobox";
import { Badge } from "@/components/Badge";
import { JsonView } from "@/components/JsonView";
import { EntityLink } from "@/components/EntityLink";
import { useTenant } from "@/components/TenantContext";
import { useTimeRange } from "@/components/TimeRangeContext";

export default function StreamPage() {
  const { tenantId: globalTenant } = useTenant();
  const { from: globalFrom, to: globalTo } = useTimeRange();
  const [events, setEvents] = useState<Event[]>([]), [tenants, setTenants] = useState<Tenant[]>([]);
  const [tenantSearch, setTenantSearch] = useState(""), [eventSearch, setEventSearch] = useState("");
  const [tenantIds, setTenantIds] = useState<string[]>([]), [eventNames, setEventNames] = useState<string[]>([]);
  const [timeMode, setTimeMode] = useState("relative"), [days, setDays] = useState("1"), [from, setFrom] = useState(""), [to, setTo] = useState("");
  const [style, setStyle] = useState("cards"), [running, setRunning] = useState(true), [compact, setCompact] = useState(true), [loading, setLoading] = useState(true), [error, setError] = useState("");

  useEffect(() => { api.tenants.list().then(setTenants).catch((e) => setError(e.message)); }, []);
  useEffect(() => { if (globalTenant && !tenantIds.includes(globalTenant)) setTenantIds([globalTenant]); }, [globalTenant, tenantIds]);
  useEffect(() => {
    if (!running) return;
    const now = globalTo, fromTS = globalFrom, toTS = now;
    const load = () => {
      setLoading(true);
      const tenantFilter = tenantIds.length ? tenantIds : [undefined];
      const eventFilter = eventNames.length ? eventNames : [undefined];
      const requests = tenantFilter.flatMap((tenant_id) => eventFilter.map((event_name) => api.events.list({ sort: "desc", limit: 100, offset: 0, from_ts: fromTS, to_ts: toTS, tenant_id, event_name })));
      Promise.all(requests).then((batches) => {
        const unique = new Map<string, Event>();
        batches.flat().forEach((event) => unique.set(event.EventID || `${event.TenantID}:${event.SessionID}:${event.ClientTS}:${event.EventName}`, event));
        setEvents([...unique.values()].sort((a, b) => b.ClientTS - a.ClientTS).slice(0, 100));
        setError("");
      }).catch((e) => setError(e.message)).finally(() => setLoading(false));
    };
    load(); const timer = window.setInterval(load, 2000); return () => window.clearInterval(timer);
  }, [running, timeMode, days, from, to, tenantIds, eventNames, globalFrom, globalTo]);

  const visible = useMemo(() => events, [events]);
  const tenantOptions = tenants.map((tenant) => ({ value: tenant.ID, label: tenant.Name }));
  const eventOptions = [...new Set(events.map((event) => event.EventName))].sort().map((value) => ({ value }));
  const add = (list: string[], value: string, set: (value: string[]) => void) => { if (value && !list.includes(value)) set([...list, value]); };
  const keyFor = (event: Event, index: number) => event.EventID || `${event.TenantID}-${event.SessionID}-${event.ClientTS}-${index}`;
  const copy = (event: Event) => void navigator.clipboard?.writeText(JSON.stringify(event, null, 2));

  return <div><div className="flex items-center justify-between"><div><h1 className="text-xl font-semibold text-fg">Live Stream</h1><p className="mt-1 text-xs text-fg-muted">Live events · 2s refresh</p></div><button className="rounded bg-accent px-2.5 py-1.5 text-xs text-accent-fg" onClick={() => setRunning(!running)}>{running ? "Pause" : "Resume"}</button></div>
    <details className="mt-4 rounded-lg border border-border bg-surface-2 px-4 py-3 text-sm"><summary className="cursor-pointer font-medium text-fg">Live stream help</summary><p className="mt-2 text-fg-muted">The stream polls the API every two seconds while running. Filters apply to the loaded window.</p></details>{error && <p className="mt-4 rounded border border-danger/30 bg-danger/5 p-3 text-sm text-danger">{error}</p>}
    <Card className="mt-4"><CardHeader title="Filters" /><div className="grid gap-2 p-3 md:grid-cols-2 lg:grid-cols-4"><Combobox value={tenantSearch} onChange={(value) => { add(tenantIds, value, setTenantIds); setTenantSearch(""); }} options={tenantOptions} placeholder="Tenant" /><Combobox value={eventSearch} onChange={(value) => { add(eventNames, value, setEventNames); setEventSearch(""); }} options={eventOptions} placeholder="Event" /><Combobox value={timeMode} onChange={setTimeMode} options={[{ value: "relative", label: "Relative" }, { value: "absolute", label: "Absolute" }]} placeholder="Time" />{timeMode === "relative" ? <Combobox value={days} onChange={setDays} options={[{ value: "1", label: "1 day" }, { value: "7", label: "7 days" }, { value: "30", label: "30 days" }]} placeholder="Range" /> : <><input type="datetime-local" value={from} onChange={(e) => setFrom(e.target.value)} className="rounded border border-border bg-surface px-2 py-1.5 text-xs" /><input type="datetime-local" value={to} onChange={(e) => setTo(e.target.value)} className="rounded border border-border bg-surface px-2 py-1.5 text-xs" /></>}<Combobox value={style} onChange={setStyle} options={[{ value: "cards", label: "Cards" }, { value: "table", label: "Table" }, { value: "json", label: "JSON" }]} placeholder="View" /><button type="button" onClick={() => setCompact(!compact)} className="rounded border border-border px-2 py-1.5 text-xs">{compact ? "Expand" : "Compact"}</button></div><div className="flex flex-wrap gap-1 px-3 pb-3">{tenantIds.map((id) => <button key={id} onClick={() => setTenantIds(tenantIds.filter((value) => value !== id))} className="rounded-full bg-accent/10 px-2 py-0.5 text-[11px] text-accent">Tenant: {tenants.find((tenant) => tenant.ID === id)?.Name || id} ×</button>)}{eventNames.map((name) => <button key={name} onClick={() => setEventNames(eventNames.filter((value) => value !== name))} className="rounded-full bg-accent/10 px-2 py-0.5 text-[11px] text-accent">Event: {name} ×</button>)}</div></Card>
    <Card className="mt-6"><CardHeader title={`Events (${visible.length})`} />{loading && !visible.length ? <p className="p-5 text-sm text-fg-muted">Loading live events…</p> : !visible.length ? <p className="p-5 text-sm text-fg-muted">No events match the current filters.</p> : style === "table" ? <EventTable events={visible} tenants={tenants} copy={copy} keyFor={keyFor} /> : <div className="max-h-[65vh] space-y-2 overflow-auto p-3">{visible.map((event, index) => <EventCard key={keyFor(event, index)} event={event} tenants={tenants} compact={compact} json={style === "json"} copy={() => copy(event)} />)}</div>}</Card>
  </div>;
}

function EventTable({ events, tenants, copy, keyFor }: { events: Event[]; tenants: Tenant[]; copy: (event: Event) => void; keyFor: (event: Event, index: number) => string }) { return <div className="max-h-[65vh] overflow-auto"><table className="w-full text-left text-sm"><thead><tr className="border-b border-border text-fg-muted"><th className="px-5 py-3">Event</th><th className="px-5 py-3">Tenant</th><th className="px-5 py-3">Time</th><th className="px-5 py-3 text-right">Actions</th></tr></thead><tbody>{events.map((event, index) => <tr key={keyFor(event, index)} className="border-b border-border"><td className="px-5 py-3">{event.EventID ? <Link href={`/events/${encodeURIComponent(event.EventID)}`} className="font-medium text-accent hover:underline">{event.EventName}</Link> : event.EventName}</td><td className="px-5 py-3"><EntityLink kind="tenant" id={event.TenantID} label={tenants.find((tenant) => tenant.ID === event.TenantID)?.Name || event.TenantID} /></td><td className="px-5 py-3">{new Date(event.ClientTS).toLocaleString()}</td><td className="px-5 py-3 text-right"><button className="text-xs text-accent" onClick={() => copy(event)}>Copy</button></td></tr>)}</tbody></table></div>; }
function EventCard({ event, tenants, compact, json, copy }: { event: Event; tenants: Tenant[]; compact: boolean; json: boolean; copy: () => void }) { return <div className="rounded border border-border px-2.5 py-1.5">{event.EventID ? <Link href={`/events/${encodeURIComponent(event.EventID)}`} className="font-medium text-xs text-accent hover:underline">{event.EventName}</Link> : <span className="font-medium text-xs">{event.EventName}</span>}<button className="ml-2 text-[11px] text-accent" onClick={copy}>Copy</button>{json ? <pre className="mt-1 max-h-32 overflow-auto rounded bg-surface-2 p-2 text-[11px]">{JSON.stringify(event, null, 2)}</pre> : <><div className="mt-0.5 flex items-center justify-between"><Badge tone="accent">{new Date(event.ClientTS).toLocaleString()}</Badge></div><p className="mt-1 text-[11px] text-fg-muted">Tenant <EntityLink kind="tenant" id={event.TenantID} label={tenants.find((tenant) => tenant.ID === event.TenantID)?.Name || event.TenantID} /> · Session <EntityLink kind="session" id={event.SessionID} /></p><div className="mt-1 grid gap-1 md:grid-cols-2"><div><p className="text-[10px] font-medium text-fg-muted">Payload</p>{compact ? <JsonView value={event.Payload} /> : <pre className="mt-1 max-h-32 overflow-auto rounded bg-surface-2 p-2 text-[11px]">{event.Payload || "{}"}</pre>}</div><div><p className="text-[10px] font-medium text-fg-muted">Dimensions</p>{compact ? <JsonView value={event.Dims} /> : <pre className="mt-1 max-h-32 overflow-auto rounded bg-surface-2 p-2 text-[11px]">{event.Dims || "{}"}</pre>}</div></div></>}</div>; }
