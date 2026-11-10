"use client";
import { useEffect, useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import { api, Event, GraphResult, Question, Tenant } from "@/lib/api";
import { Card, CardHeader, StatCard } from "@/components/Card";
import { Combobox } from "@/components/Combobox";
import { LineTrend, BarBreakdown, AreaTrend, PieBreakdown } from "@/components/Chart";
import { JsonView } from "@/components/JsonView";


type Panel = { id: string; title: string; eventNames: string[]; chart: "line" | "area" | "bar" | "pie" | "table"; result?: GraphResult; events?: Event[] };
const ranges = [{ value: "1", label: "Last 1 day" }, { value: "7", label: "Last 7 days" }, { value: "30", label: "Last 30 days" }, { value: "90", label: "Last 90 days" }];
export default function AnalyticsPage() {
  const router = useRouter();
  const [tenants, setTenants] = useState<Tenant[]>([]), [tenant, setTenant] = useState("");
  const urlParams = typeof window !== "undefined" ? new URLSearchParams(window.location.search) : null;
  const [events, setEvents] = useState<string[]>([]), [query, setQuery] = useState(""), [selected, setSelected] = useState<string[]>(() => urlParams?.get("events")?.split(",").filter(Boolean) || []), [range, setRange] = useState(() => urlParams?.get("range") || "7"), [chart, setChart] = useState(() => urlParams?.get("chart") || "line");
  const [panels, setPanels] = useState<Panel[]>([{ id: "main", title: "Event volume", eventNames: [], chart: "line" }]), [refresh, setRefresh] = useState(true), [error, setError] = useState("");
  useEffect(() => { api.tenants.list().then(x => { setTenants(x); const fromURL = new URLSearchParams(window.location.search).get("tenant"); setTenant(fromURL || x[0]?.ID || ""); }).catch(e => setError(e.message)); }, []);
  useEffect(() => { const params = new URLSearchParams(); if (tenant) params.set("tenant", tenant); if (selected.length) params.set("events", selected.join(",")); params.set("range", range); params.set("chart", chart); router.replace(`/analytics?${params.toString()}`, { scroll: false }); }, [tenant, selected, range, chart, router]);
  useEffect(() => { if (!tenant) return; loadEventOptions(""); }, [tenant]);
  useEffect(() => { if (!refresh || !panels.length) return; const t = window.setInterval(() => panels.forEach(loadPanel), 15000); return () => window.clearInterval(t); }, [refresh, panels, tenant]);
  async function loadEventOptions(q: string) { if (!tenant) return; try { setEvents((await api.eventOptions(tenant, q)).map(x => x.value)); } catch {} }
  async function loadPanel(panel: Panel) {
    if (!tenant || !panel.eventNames.length) return;
    try {
      const now = Date.now(), from = now - Number(range) * 86400000;
      const data = await Promise.all(panel.eventNames.map(name => api.events.list({ tenant_id: tenant, event_name: name, from_ts: from, to_ts: now, sort: "desc" })));
      const flat = data.flat(), buckets = new Map<string, number>();
      flat.forEach(e => { const key = new Date(e.ClientTS).toISOString().slice(0, 10); buckets.set(key, (buckets.get(key) || 0) + 1); });
      const points = [...buckets.entries()].sort(([a], [b]) => a.localeCompare(b)).map(([label, value]) => ({ label, value }));
      setPanels(current => current.map(p => p.id === panel.id ? { ...p, events: flat, result: { question: {} as Question, from_ts: from, to_ts: now, points } } : p));
    } catch (e) { setError((e as Error).message); }
  }
  function addEvent(v: string) { if (v && !selected.includes(v)) { setSelected([...selected, v]); setQuery(""); } }
  useEffect(() => {
    if (!tenant || !selected.length || panels[0]?.eventNames.length) return;
    const panel: Panel = { id: "main", title: selected.join(" + "), eventNames: selected, chart: chart as Panel["chart"] };
    setPanels([panel]);
    loadPanel(panel);
  }, [tenant, selected]);
  function runQuery() { if (!selected.length) return; const panel = { id: "main", title: selected.join(" + "), eventNames: selected, chart: chart as Panel["chart"] }; setPanels([panel]); setSelected([]); setQuery(""); setTimeout(() => loadPanel(panel), 0); }
    const totals = useMemo(() => panels.reduce((sum, p) => sum + (p.result?.points.reduce((a, x) => a + x.value, 0) || 0), 0), [panels]);
  return <div><div className="flex flex-wrap items-start justify-between gap-4"><div><h1 className="text-xl font-semibold text-fg">Analytics</h1><p className="mt-1 text-sm text-fg-muted">Build, save, monitor, and explore event queries in one workspace.</p></div><div className="flex gap-2"><button onClick={() => setRefresh(!refresh)} className="rounded border border-border px-3 py-2 text-sm">{refresh ? "Auto-refresh on" : "Auto-refresh off"}</button><button onClick={() => panels.forEach(loadPanel)} className="rounded bg-accent px-3 py-2 text-sm text-accent-fg">Refresh now</button></div></div>
    <Card className="mt-6"><CardHeader title="Query builder"/><div className="grid gap-3 p-5 md:grid-cols-4"><Combobox value={tenant} onChange={setTenant} options={tenants.map(t => ({ value: t.ID, label: t.Name }))} placeholder="Tenant"/><Combobox value={query} onChange={v => { if (events.includes(v)) addEvent(v); else setQuery(v); loadEventOptions(v); }} onSearch={loadEventOptions} options={events.map(value => ({ value }))} placeholder="Search event…"/><Combobox value={range} onChange={setRange} options={ranges} placeholder="Time range"/><Combobox value={chart} onChange={v=>{setChart(v);setPanels(x=>x.map(p=>({...p,chart:v as Panel["chart"]})))} } options={[{ value: "line", label: "Line chart" }, { value: "area", label: "Area chart" }, { value: "bar", label: "Bar chart" }, { value: "pie", label: "Pie chart" }, { value: "table", label: "Table" }]} placeholder="Visualization"/><Combobox value="count" onChange={()=>{}} options={[{value:"count",label:"Count events"},{value:"unique_sessions",label:"Unique sessions"}]} placeholder="Aggregation"/><Combobox value="day" onChange={()=>{}} options={[{value:"minute",label:"By minute"},{value:"hour",label:"By hour"},{value:"day",label:"By day"}]} placeholder="Group by"/></div><div className="flex flex-wrap items-center gap-2 px-5 pb-5">{selected.map(x => <button key={x} onClick={() => setSelected(selected.filter(y => y !== x))} className="rounded-full bg-accent/10 px-3 py-1 text-xs text-accent">{x} ×</button>)}<button disabled={!selected.length} onClick={runQuery} className="rounded bg-accent px-3 py-2 text-sm text-accent-fg disabled:opacity-50">Run query</button></div></Card>
    <div className="mt-6 w-full"><div className="w-full">{panels.map(p => <Card key={p.id}><CardHeader title={p.title} action={<span className="text-xs text-fg-muted">Live panel</span>}/><div className="p-5">{p.result && p.chart === "line" && <LineTrend data={p.result.points}/>} {p.result && p.chart === "area" && <AreaTrend data={p.result.points}/>} {p.result && p.chart === "bar" && <BarBreakdown data={p.result.points}/>} {p.result && p.chart === "pie" && <PieBreakdown data={p.result.points}/>} {p.result && p.chart === "table" && <div className="space-y-2">{p.result.points.map(x => <div key={x.label} className="flex justify-between border-b border-border py-2 text-sm"><span>{x.label}</span><span className="font-medium">{x.value.toLocaleString()}</span></div>)}</div>} {!p.result && <p className="text-sm text-fg-muted">Loading panel…</p>}<div className="mt-4 text-xs text-fg-muted">{p.events?.length || 0} matching events · {range} day window</div></div></Card>)}</div></div>
    <Card className="mt-6"><CardHeader title="Recent matching events"/><div className="divide-y divide-border">{panels.flatMap(p => p.events || []).slice(0, 12).map(e => <div key={e.EventID} className="flex items-center justify-between px-5 py-3 text-sm"><span className="font-medium text-fg">{e.EventName}</span><span className="text-fg-muted">{new Date(e.ClientTS).toLocaleString()}</span><JsonView value={e.Payload}/></div>)}{!panels.some(p => p.events?.length) && <p className="p-5 text-sm text-fg-muted">No matching events loaded.</p>}</div></Card>
  </div>;
}
