"use client";
import { IntelligenceTags } from "@/components/IntelligenceTags";
import { useCallback, useEffect, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import { api, Event, GraphResult, Question, Tenant } from "@/lib/api";
import { Card, CardHeader } from "@/components/Card";
import { Combobox } from "@/components/Combobox";
import { LineTrend, BarBreakdown, AreaTrend, PieBreakdown } from "@/components/Chart";
import { JsonView } from "@/components/JsonView";
import { useTenant } from "@/components/TenantContext";
import { formatTimeRange, useTimeRange } from "@/components/TimeRangeContext";
import Link from "next/link";


type Panel = { id: string; title: string; eventNames: string[]; chart: "line" | "area" | "bar" | "pie" | "table"; result?: GraphResult; events?: Event[] };
const groupingOptions = [{ value: "auto", label: "Auto time buckets" }, { value: "minute", label: "By minute" }, { value: "hour", label: "By hour" }, { value: "day", label: "By day" }, { value: "dim:country", label: "Event dimension: country" }, { value: "dim:region", label: "Event dimension: region" }, { value: "dim:city", label: "Event dimension: city" }, { value: "dim:device", label: "Event dimension: device" }, { value: "dim:campaign", label: "Event dimension: campaign" }];
function autoGrouping(hours: string): "minute" | "hour" | "day" { const value = Number(hours); return value <= 6 ? "minute" : value <= 168 ? "hour" : "day"; }
function dimensionValue(raw: string, key: string): string { try { const value = JSON.parse(raw || "{}")[key]; return value === undefined || value === null || value === "" ? "(unknown)" : String(value); } catch { return "(unknown)"; } }
function AdvancedAnalytics() {
  const router = useRouter();
  const { tenantId: globalTenant, setTenantId: setGlobalTenant, loading: scopeLoading } = useTenant();
  const { hours, from: globalFrom, to: globalTo } = useTimeRange();
  const timeRangeLabel = formatTimeRange(hours);
  void timeRangeLabel;
  const urlParams = typeof window !== "undefined" ? new URLSearchParams(window.location.search) : null;

  const [tenants, setTenants] = useState<Tenant[]>([]), [tenant, setTenant] = useState(globalTenant);
  const [events, setEvents] = useState<string[]>([]), [query, setQuery] = useState(""), [selected, setSelected] = useState<string[]>(() => urlParams?.get("events")?.split(",").filter(Boolean) || []), [chart, setChart] = useState(() => urlParams?.get("chart") || "line"), [aggregation, setAggregation] = useState(() => urlParams?.get("aggregation") || "count"), [groupBy, setGroupBy] = useState(() => urlParams?.get("group") || "auto");
  const [panels, setPanels] = useState<Panel[]>([{ id: "main", title: "Event volume", eventNames: [], chart: "line" }]), [refresh, setRefresh] = useState(false), [error, setError] = useState(""), [queryLoading, setQueryLoading] = useState(false);
  const [copied, setCopied] = useState(false);
  useEffect(() => { api.tenants.list().then(setTenants).catch(e => setError(e.message)); }, []);
  useEffect(() => { setTenant(globalTenant); }, [globalTenant]);
  useEffect(() => { const params = new URLSearchParams(); if (tenant) params.set("tenant", tenant); if (selected.length) params.set("events", selected.join(",")); params.set("time_hours", hours); params.set("chart", chart); params.set("aggregation", aggregation); params.set("group", groupBy); router.replace(`/analytics?${params.toString()}`, { scroll: false }); }, [tenant, selected, hours, chart, aggregation, groupBy, router]);
  const loadEventOptions = useCallback(async (q: string) => { if (scopeLoading || !globalTo) return; try { const options = await api.eventOptions(tenant, q, q === "", { from_ts: globalFrom, to_ts: globalTo }); setEvents((Array.isArray(options) ? options : []).map(x => x.value)); } catch (e) { setError((e as Error).message); } }, [tenant, scopeLoading, globalFrom, globalTo]);
  const panelsRef = useRef(panels);
  useEffect(() => { panelsRef.current = panels; }, [panels]);
  const inFlight = useRef(new Set<string>());
  const loadPanel = useCallback(async (panel: Panel) => {
    if (scopeLoading || !globalTo || !panel.eventNames.length) return;
    if (inFlight.current.has(panel.id)) return;
    inFlight.current.add(panel.id);
    setQueryLoading(true);
    try {
      const now = globalTo, from = globalFrom, effectiveGroup = groupBy === "auto" ? autoGrouping(hours) : groupBy;
      const bucketName = effectiveGroup === "minute" ? "minute" : effectiveGroup === "hour" ? "hour" : "";
      // Bound concurrency across event names; preserve per-event query semantics.
      const data: Event[][] = [];
      const bucketData: Awaited<ReturnType<typeof api.analytics.buckets>>[] = [];
      for (let i = 0; i < panel.eventNames.length; i += 4) {
        const batch = panel.eventNames.slice(i, i + 4);
        const [rows, points] = await Promise.all([
          Promise.all(batch.map(name => api.events.list({ tenant_id: tenant, event_name: name, from_ts: from, to_ts: now, sort: "desc", limit: bucketName ? 100 : 500, offset: 0 }))),
          aggregation === "count" && bucketName ? Promise.all(batch.map(name => api.analytics.buckets(tenant, { event_name: name, bucket: bucketName as "minute" | "hour", from_ts: from, to_ts: now }))) : Promise.resolve([]),
        ]);
        data.push(...rows);
        bucketData.push(...points);
      }
      const flat = data.flat(), buckets = new Map<string, Set<string>>();
      const counts = new Map<string, number>();
      if (bucketData.length) {
        bucketData.flat().forEach(point => { const date = new Date(point.bucket), key = effectiveGroup === "minute" ? date.toISOString().slice(0, 16) : date.toISOString().slice(0, 13) + ":00"; counts.set(key, (counts.get(key) ?? 0) + point.count); });
      }
      if (!bucketData.length) flat.forEach(e => { const date = new Date(e.ClientTS), key = effectiveGroup.startsWith("dim:") ? dimensionValue(e.Dims, effectiveGroup.slice(4)) : effectiveGroup === "minute" ? date.toISOString().slice(0, 16) : effectiveGroup === "hour" ? date.toISOString().slice(0, 13) + ":00" : date.toISOString().slice(0, 10); if (!buckets.has(key)) buckets.set(key, new Set()); buckets.get(key)!.add(aggregation === "unique_sessions" ? e.SessionID : e.EventID); });
      const points = (bucketData.length ? [...counts] : [...buckets].map(([key, values]) => [key, values.size] as const)).sort(([a], [b]) => a.localeCompare(b)).map(([label, value]) => ({ label, value }));
      setPanels(current => current.map(p => p.id === panel.id ? { ...p, events: flat, result: { question: {} as Question, from_ts: from, to_ts: now, points } } : p));
    } catch (e) { setError((e as Error).message); } finally { inFlight.current.delete(panel.id); setQueryLoading(inFlight.current.size > 0); }
  }, [aggregation, groupBy, hours, tenant, globalFrom, globalTo, scopeLoading]);
  useEffect(() => { void loadEventOptions(""); }, [loadEventOptions, tenant]);
  useEffect(() => { if (!refresh || !panels.length) return; const t = window.setInterval(() => { if (document.visibilityState === "visible") panelsRef.current.forEach(loadPanel); }, 60000); return () => window.clearInterval(t); }, [loadPanel, panels.length, refresh]);
  useEffect(() => { panelsRef.current.forEach(panel => { if (panel.eventNames.length) void loadPanel(panel); }); }, [loadPanel]);
  function addEvent(v: string) { if (v && !selected.includes(v)) { setSelected([...selected, v]); setQuery(""); } }
  useEffect(() => {
    if (scopeLoading || !globalTo || !selected.length || panels[0]?.eventNames.length) return;
    const panel: Panel = { id: "main", title: selected.join(" + "), eventNames: selected, chart: chart as Panel["chart"] };
    setPanels([panel]);
    loadPanel(panel);
  }, [chart, loadPanel, panels, selected, tenant, scopeLoading, globalTo]);
  function runQuery() { if (!selected.length || queryLoading) return; const panel = { id: "main", title: selected.join(" + "), eventNames: selected, chart: chart as Panel["chart"] }; setPanels([panel]); setTimeout(() => loadPanel(panel), 0); }
  function addPanel() { if (!selected.length) return; const panel = { id: `panel-${Date.now()}`, title: selected.join(" + "), eventNames: selected, chart: chart as Panel["chart"] }; setPanels(current => [...current, panel]); void loadPanel(panel); }
  function removePanel(id: string) { if (panels.length === 1) return; setPanels(current => current.filter(panel => panel.id !== id)); }
  function queryString() { const params = new URLSearchParams({ tenant, events: selected.join(","), time_hours: hours, chart, aggregation, group: groupBy, panels: JSON.stringify(panels.map(({ id, title, eventNames, chart }) => ({ id, title, eventNames, chart }))) }); return params.toString(); }
  async function copyShareLink() { try { await navigator.clipboard.writeText(`${window.location.origin}/analytics?${queryString()}`); setCopied(true); window.setTimeout(() => setCopied(false), 1800); } catch { setError("Could not copy the share link"); } }
  const matchingEvents = panels.flatMap(p => p.events || []);
  const sessions = new Map<string, Set<string>>();
  matchingEvents.forEach(event => { const key = event.SessionID || event.EventID; if (!sessions.has(key)) sessions.set(key, new Set()); sessions.get(key)!.add(event.EventName); });
  const convertedSessions = [...sessions.values()].filter(names => selected.every(name => names.has(name))).length;
  const conversionRate = sessions.size ? Math.round((convertedSessions / sessions.size) * 100) : 0;
  return <div><div className="flex flex-wrap items-start justify-between gap-4"><div><h1 className="text-xl font-semibold text-fg">Analytics</h1><p className="mt-1 text-sm text-fg-muted">Build, save, monitor, and explore event queries in one workspace.</p><details className="mt-4 rounded-lg border border-border bg-surface-2 px-4 py-3 text-sm"><summary className="cursor-pointer font-medium text-fg">Analytics help</summary><p className="mt-2 text-fg-muted">Choose a tenant, add one or more event names, select aggregation and grouping, then run the query. Query state is shareable in the URL and panels refresh from the API.</p></details></div><div className="flex gap-2"><button onClick={() => setRefresh(!refresh)} className="rounded border border-border px-3 py-2 text-sm">{refresh ? "Auto-refresh on" : "Auto-refresh off"}</button><button onClick={() => panels.forEach(loadPanel)} disabled={queryLoading} className="rounded bg-accent px-3 py-2 text-sm text-accent-fg disabled:opacity-50">{queryLoading ? "Loading…" : "Refresh now"}</button></div></div>{error && <p className="mt-4 rounded border border-danger/30 bg-danger/5 p-3 text-sm text-danger">{error}</p>}
    <Card className="mt-6"><CardHeader title="Query builder"/><div className="grid gap-3 p-5 md:grid-cols-3"><Combobox value={tenant} onChange={(value) => { setTenant(value); setGlobalTenant(value); }} options={tenants.map(t => ({ value: t.ID, label: t.Name }))} placeholder="Tenant"/><Combobox value={query} onChange={v => { if (events.includes(v)) addEvent(v); else setQuery(v); loadEventOptions(v); }} onSearch={loadEventOptions} options={events.map(value => ({ value }))} placeholder="Search event…"/><Combobox value={chart} onChange={v=>{setChart(v);setPanels(x=>x.map(p=>({...p,chart:v as Panel["chart"]})))} } options={[{ value: "line", label: "Line chart" }, { value: "area", label: "Area chart" }, { value: "bar", label: "Bar chart" }, { value: "pie", label: "Pie chart" }, { value: "table", label: "Table" }]} placeholder="Visualization"/><Combobox value={aggregation} onChange={setAggregation} options={[{value:"count",label:"Count events"},{value:"unique_sessions",label:"Unique sessions"}]} placeholder="Aggregation"/><Combobox value={groupBy} onChange={setGroupBy} options={groupingOptions} placeholder="Group by"/></div><div className="flex flex-wrap items-center gap-2 px-5 pb-5"><span className="text-xs text-fg-muted">Global: {timeRangeLabel}</span>{selected.map(x => <button key={x} onClick={() => setSelected(selected.filter(y => y !== x))} className="rounded-full bg-accent/10 px-3 py-1 text-xs text-accent">{x} ×</button>)}<button disabled={!selected.length || queryLoading} onClick={runQuery} className="rounded bg-accent px-3 py-2 text-sm text-accent-fg disabled:opacity-50">Run query</button><button disabled={!selected.length} onClick={addPanel} className="rounded border border-border px-3 py-2 text-sm disabled:opacity-50">Add panel</button><button disabled={!selected.length} onClick={()=>void copyShareLink()} className="rounded border border-border px-3 py-2 text-sm disabled:opacity-50">{copied ? "Link copied" : "Copy share link"}</button></div></Card>
    <div className="mt-6 w-full"><div className="grid w-full gap-5">{panels.map(p => <Card key={p.id}><CardHeader title={p.title} action={<div className="flex items-center gap-2"><span className="text-xs text-fg-muted">Live panel</span><button type="button" onClick={() => removePanel(p.id)} disabled={panels.length === 1} className="text-xs text-danger disabled:opacity-30">Remove</button></div>}/><div className="min-h-[460px] p-6">{p.result && p.chart === "line" && <LineTrend data={p.result.points}/>} {p.result && p.chart === "area" && <AreaTrend data={p.result.points}/>} {p.result && p.chart === "bar" && <BarBreakdown data={p.result.points}/>} {p.result && p.chart === "pie" && <PieBreakdown data={p.result.points}/>} {p.result && p.chart === "table" && <div className="space-y-2">{p.result.points.map(x => <div key={x.label} className="flex justify-between border-b border-border py-2 text-sm"><span>{x.label}</span><span className="font-medium">{x.value.toLocaleString()}</span></div>)}</div>} {!p.result && <p className="text-sm text-fg-muted">{p.eventNames.length ? "Loading panel…" : "Select events and run a query."}</p>}<div className="mt-4 text-xs text-fg-muted">{p.events?.length || 0} matching events · {timeRangeLabel} global window</div></div></Card>)}</div></div>
    <Card className="mt-6"><CardHeader title="Conversion breakdown" action={<span className="text-xs text-fg-muted">Session based</span>}/><div className="grid gap-4 p-5 sm:grid-cols-3"><div><p className="text-xs text-fg-muted">Sessions in range</p><p className="mt-1 text-2xl font-semibold text-fg">{sessions.size.toLocaleString()}</p></div><div><p className="text-xs text-fg-muted">Completed sequence</p><p className="mt-1 text-2xl font-semibold text-fg">{convertedSessions.toLocaleString()}</p></div><div><p className="text-xs text-fg-muted">Conversion rate</p><p className="mt-1 text-2xl font-semibold text-accent">{conversionRate}%</p></div></div><div className="flex flex-wrap gap-2 px-5 pb-5">{selected.map((eventName, index) => <Link key={eventName} href={`/events?tenant_id=${encodeURIComponent(tenant)}&event_name=${encodeURIComponent(eventName)}`} className="rounded-full border border-border px-3 py-1 text-xs text-fg-muted hover:border-accent hover:text-accent">{index + 1}. {eventName}</Link>)}</div></Card>
    <Card className="mt-6"><CardHeader title="Recent matching events"/><div className="divide-y divide-border">{panels.flatMap(p => p.events || []).slice(0, 12).map((e, index) => <div key={e.EventID || `${e.EventName}-${e.ClientTS}-${index}`} className="flex items-center justify-between px-5 py-3 text-sm"><span className="font-medium text-fg">{e.EventID ? <a href={`/events/${encodeURIComponent(e.EventID)}`} className="text-accent hover:underline">{e.EventName}</a> : <Link href={`/events?tenant_id=${encodeURIComponent(e.TenantID)}&event_name=${encodeURIComponent(e.EventName)}`} className="text-accent hover:underline">{e.EventName}</Link>}<IntelligenceTags tags={e.intelligence_tags} compact /></span><span className="text-fg-muted">{new Date(e.ClientTS).toLocaleString()}</span><JsonView value={e.Payload}/></div>)}{!panels.some(p => p.events?.length) && <p className="p-5 text-sm text-fg-muted">No matching events loaded.</p>}</div></Card>
  </div>;
}

const reportTemplates = [
  { name: "Overview", description: "Understand overall activity", pattern: "" },
  { name: "Website", description: "Page views and clicks", pattern: "page|view|click" },
  { name: "Engagement", description: "Interactions and feature usage", pattern: "click|search|view|open|engage" },
  { name: "Acquisition", description: "Signups and referrals", pattern: "signup|register|referr|campaign|install" },
  { name: "Commerce", description: "Shopping and purchases", pattern: "cart|checkout|purchase|order|payment" },
  { name: "Search", description: "Search activity", pattern: "search" },
  { name: "Mobile app", description: "App and screen activity", pattern: "app|screen|install" },
  { name: "Custom", description: "Choose any tracked event", pattern: "" },
];
type ReportOption = { value: string; count: number; last_ts: number };
function GuidedAnalytics() {
  const { tenantId, loading: scopeLoading } = useTenant();
  const { hours, from, to } = useTimeRange();
  const [template, setTemplate] = useState("Overview");
  const [event, setEvent] = useState("");
  const [revision, setRevision] = useState(0);
  const [compare, setCompare] = useState(true);
  const [state, setState] = useState<{key:string; current:ReportOption[]; previous:ReportOption[]; points:{label:string;value:number}[]; error?:string}>();
  const key = JSON.stringify([tenantId,from,to,event,compare,revision]);
  useEffect(() => {
    if (scopeLoading || !to) return;
    let active = true;
    const range = {from_ts:from,to_ts:to};
    const bucket = Number(hours) <= 6 ? "minute" : "hour";
    Promise.all([
      api.eventOptions(tenantId,"",true,range),
      compare ? api.eventOptions(tenantId,"",true,{from_ts:from-(to-from),to_ts:from}) : Promise.resolve([]),
      api.analytics.buckets(tenantId,{...range,event_name:event,bucket}),
    ]).then(([current,previous,buckets]) => {
      const grouped = new Map<string,number>();
      for (const point of buckets) {
        const date = new Date(point.bucket);
        if (!Number.isFinite(date.getTime())) continue;
        const label = date.toISOString().slice(0,Number(hours)>168?10:16);
        grouped.set(label,(grouped.get(label)??0)+point.count);
      }
      if(active) setState({key,current,previous,points:[...grouped].sort(([a],[b])=>a.localeCompare(b)).map(([label,value])=>({label,value}))});
    }).catch(e=>{if(active)setState({key,current:[],previous:[],points:[],error:String(e.message??e)});});
    return ()=>{active=false;};
  },[key,tenantId,from,to,event,compare,hours,scopeLoading]);
  const ready = !scopeLoading && state?.key===key ? state : undefined;
  const pattern = reportTemplates.find(t=>t.name===template)?.pattern ?? "";
  const matches = (row:ReportOption) => (!event || row.value===event) && (!pattern || new RegExp(pattern,"i").test(row.value));
  const rows = (ready?.current??[]).filter(matches).sort((a,b)=>b.count-a.count);
  const total = rows.reduce((n,r)=>n+r.count,0);
  const previous = (ready?.previous??[]).filter(matches).reduce((n,r)=>n+r.count,0);
  const change = previous ? (total-previous)/previous*100 : null;
  const available = ready && !ready.error;
  function exportReport() {
    const csv = ["Event,Count",...rows.map(r=>'"'+r.value.replace(/^[=+@-]/,"'$&").replace(/"/g,'""')+'",'+r.count)].join("\n");
    const url=URL.createObjectURL(new Blob([csv],{type:"text/csv"}));
    const a=document.createElement("a");a.href=url;a.download="analytics.csv";a.click();URL.revokeObjectURL(url);
  }
  return <div className="space-y-4">
    <div className="flex flex-wrap items-center justify-between gap-3"><div><h1 className="text-xl font-semibold text-fg">Analytics overview</h1><p className="text-sm text-fg-muted">Understand your activity without writing a query. · {formatTimeRange(hours)}</p></div><div className="flex gap-2"><button className="rounded border border-border px-3 py-2 text-sm" onClick={()=>setRevision(x=>x+1)}>Refresh</button><button disabled={!available} className="rounded border border-border px-3 py-2 text-sm disabled:opacity-50" onClick={exportReport}>Export CSV</button></div></div>
    <Card><CardHeader title="Choose a report"/><div className="grid grid-cols-2 gap-2 p-3 md:grid-cols-4">{reportTemplates.map(t=><button key={t.name} aria-pressed={template===t.name} onClick={()=>{setTemplate(t.name);setEvent("");}} className={"rounded-lg border p-3 text-left "+(template===t.name?"border-accent bg-accent/10":"border-border hover:border-accent")}><span className="block text-sm font-medium">{t.name}</span><span className="text-xs text-fg-muted">{t.description}</span></button>)}</div></Card>
    <div className="flex flex-wrap items-center gap-3"><label className="text-sm">Event <select className="ml-2 rounded border border-border bg-surface-2 p-2" value={event} onChange={e=>setEvent(e.target.value)}><option value="">All matching events</option>{event && !ready?.current.some(r=>r.value===event) && <option value={event}>{event}</option>}{(ready?.current??[]).filter(r=>!pattern||new RegExp(pattern,"i").test(r.value)).map(r=><option key={r.value} value={r.value}>{r.value}</option>)}</select></label><label className="flex items-center gap-2 text-sm"><input type="checkbox" checked={compare} onChange={e=>setCompare(e.target.checked)}/> Compare previous period</label></div>
    {!ready && <p role="status" className="text-sm text-fg-muted">Loading report…</p>}
    {ready?.error && <p role="alert" className="text-sm text-danger">{ready.error}</p>}
    <div className="grid gap-3 sm:grid-cols-3">{[{label:"Total events",value:total.toLocaleString(),help:"Recorded actions in this report"},{label:"Event types",value:String(rows.length),help:"Different tracked actions"},{label:"Change",value:compare&&change!==null?(change>=0?"+":"")+change.toFixed(1)+"%":"—",help:"Compared with the preceding equal period"}].map(m=><Card key={m.label}><div className="p-4"><p className="text-xs text-fg-muted">{m.label}</p><p className="mt-1 text-2xl font-semibold tabular-nums">{available?m.value:"—"}</p><p className="text-xs text-fg-muted">{m.help}</p></div></Card>)}</div>
    {available && !rows.length && <p className="rounded border border-border p-4 text-sm text-fg-muted">No matching events. Try Overview or change the global time range. Templates match event names automatically.</p>}
    {available && (!pattern||event) && <Card><CardHeader title="Activity over time"/><div className="p-4">{ready.points.length?<LineTrend data={ready.points}/>:<p className="text-sm text-fg-muted">No activity buckets available.</p>}</div></Card>}
    <div className="grid gap-4 lg:grid-cols-2"><Card><CardHeader title="Top events"/><div className="space-y-3 p-4">{rows.slice(0,10).map(r=><button key={r.value} onClick={()=>setEvent(r.value)} className="block w-full text-left"><div className="flex justify-between gap-2 text-sm"><span className="truncate">{r.value}</span><span>{r.count.toLocaleString()}</span></div><div className="mt-1 h-2 rounded bg-surface-2"><div className="h-2 rounded bg-accent" style={{width:(total?r.count/total*100:0)+"%"}}/></div></button>)}</div></Card><Card><CardHeader title="Insights"/><div className="space-y-3 p-4 text-sm text-fg-muted">{available && rows.length>0?<><p>{rows[0].value} is the most frequent action in this report ({(rows[0].count/Math.max(total,1)*100).toFixed(1)}% of recorded actions).</p>{compare&&change!==null&&<p>Activity {change>=0?"increased":"decreased"} by {Math.abs(change).toFixed(1)}% compared with the previous period.</p>}<p>Click an event to focus the report on that action.</p></>:<p>Insights appear when matching data is available.</p>}</div></Card></div>
    <Card><CardHeader title="Explore further"/><div className="grid gap-2 p-4 sm:grid-cols-3">{[{href:"/analytics/funnel",title:"Conversions & funnels",desc:"See where users drop off"},{href:"/analytics/retention",title:"Retention",desc:"See whether users return"},{href:"/dimensions",title:"Dimensions",desc:"Explore devices, geography and other properties"}].map(l=><Link key={l.href} href={l.href} className="rounded border border-border p-3 hover:border-accent"><span className="block text-sm font-medium text-accent">{l.title}</span><span className="text-xs text-fg-muted">{l.desc}</span></Link>)}</div></Card>
  </div>;
}
export default function AnalyticsPage() {
  const [advanced,setAdvanced]=useState(()=>typeof window!=="undefined"&&new URLSearchParams(window.location.search).has("events"));
  return <div><div className="mb-4 flex gap-2 border-b border-border pb-2">{["Reports","Advanced explorer"].map((label,i)=><button key={label} aria-pressed={advanced===Boolean(i)} onClick={()=>setAdvanced(Boolean(i))} className={"rounded px-3 py-2 text-sm "+(advanced===Boolean(i)?"bg-accent text-accent-fg":"text-fg-muted")}>{label}</button>)}</div>{advanced?<AdvancedAnalytics/>:<GuidedAnalytics/>}</div>;
}
