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
  { name: "Website", description: "Page views and clicks", pattern: "\\b(page|pageview|view|click|navigation|route)\\b" },
  { name: "Engagement", description: "Interactions and feature usage", pattern: "\\b(click|search|view|open|engage|engagement|scroll|share|play|download|interaction)\\b" },
  { name: "Acquisition", description: "Signups and referrals", pattern: "\\b(signup|sign up|register|registration|referral|referrer|campaign|install|acquisition|subscribe)\\b" },
  { name: "Commerce", description: "Shopping and purchases", pattern: "\\b(cart|checkout|check out|purchase|order|payment|refund|transaction|product|basket)\\b" },
  { name: "Search", description: "Search activity", pattern: "\\b(search|query|autocomplete|suggestion)\\b" },
  { name: "Mobile app", description: "App and screen activity", pattern: "\\b(app|screen|install|foreground|background|notification|push)\\b" },
  { name: "Audience", description: "Discover audience properties", pattern: "" },
  { name: "Technology", description: "Devices, browsers and platforms", pattern: "" },
  { name: "Geography", description: "Countries, regions and cities", pattern: "" },
  { name: "Custom", description: "Choose any tracked event", pattern: "" },
];
type ReportOption = { value: string; count: number; last_ts: number };
function WorkspaceTrend({points,previous,onSelect}:{points:{label:string;value:number}[];previous?:{label:string;value:number}[];onSelect:(label:string)=>void}) {
 const [hover,setHover]=useState<number|null>(null);
 const [mode,setMode]=useState<"line"|"bar">("line");
 const max=Math.max(1,...points.map(p=>p.value),...(previous??[]).map(p=>p.value));
 const x=(i:number)=>points.length===1?400:40+i*740/Math.max(1,points.length-1);
 const y=(v:number)=>185-v/max*155;
 const active=hover===null?null:points[hover];
 const prior=hover===null?undefined:previous?.[hover];
 const priorPath=(previous??[]).map((p,i)=>(i?"L":"M")+x(i)+" "+y(p.value)).join(" ");
 const path=points.map((p,i)=>(i?"L":"M")+x(i)+" "+y(p.value)).join(" ");
 return <div className="space-y-3"><div className="flex flex-wrap items-center justify-between gap-2"><div><p className="text-xs text-fg-muted">Events · UTC {previous?.length?" · dashed: previous period":""}</p><p className="text-xl font-semibold tabular-nums">{points.reduce((n,p)=>n+p.value,0).toLocaleString()}</p></div><div className="flex rounded-lg border border-border p-1">{(["line","bar"] as const).map(m=><button key={m} aria-pressed={mode===m} onClick={()=>setMode(m)} className={"rounded px-3 py-1 text-xs capitalize "+(mode===m?"bg-accent/10 text-accent":"text-fg-muted")}>{m}</button>)}</div></div><div className="relative" onMouseLeave={()=>setHover(null)}><svg viewBox="0 0 800 215" className="w-full overflow-visible text-accent" role="img" aria-label="Activity trend chart" onClick={()=>{if(active)onSelect(active.label);}} onPointerMove={e=>{const r=e.currentTarget.getBoundingClientRect();setHover(Math.max(0,Math.min(points.length-1,Math.round(((e.clientX-r.left)/r.width*800-40)/740*(points.length-1)))));}}>
 {[0,.25,.5,.75,1].map(v=><g key={v}><line x1="40" x2="780" y1={y(v*max)} y2={y(v*max)} stroke="currentColor" opacity=".12"/><text x="34" y={y(v*max)+4} textAnchor="end" fontSize="10" fill="currentColor">{Math.round(v*max).toLocaleString()}</text></g>)}
 {mode==="line"?<><path d={path+" L "+x(points.length-1)+" 185 L "+x(0)+" 185 Z"} fill="currentColor" opacity=".08"/><path d={path} fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinejoin="round"/></>:points.map((p,i)=><rect key={p.label} x={x(i)-Math.min(12,300/points.length)} y={y(p.value)} width={Math.min(24,600/points.length)} height={185-y(p.value)} rx="2" fill="currentColor" opacity=".75"/>)}
 {previous?.length===points.length&&<path d={priorPath} fill="none" stroke="currentColor" opacity=".55" strokeWidth="2" strokeDasharray="5 5"/>}
 {hover!==null&&active&&<g><line x1={x(hover)} x2={x(hover)} y1="25" y2="185" stroke="currentColor" strokeDasharray="4 4"/><circle cx={x(hover)} cy={y(active.value)} r="4" fill="currentColor"/></g>}
 {points.filter((_,i)=>i===0||i===points.length-1||i%Math.max(1,Math.ceil(points.length/5))===0).map(p=><text key={p.label} x={x(points.indexOf(p))} y="207" fontSize="10" textAnchor="middle" fill="currentColor">{p.label.slice(5)}</text>)}
 </svg>{active&&<div className="pointer-events-none absolute right-2 top-2 rounded-lg border border-border bg-surface-2 px-3 py-2 text-sm shadow-lg"><p className="text-xs text-fg-muted">{active.label} UTC</p><p className="font-semibold tabular-nums">{active.value.toLocaleString()} events</p>{prior&&<p className="text-xs text-fg-muted">Previous: {prior.value.toLocaleString()} · {prior.label} UTC</p>}</div>}</div><label className="flex items-center gap-2 text-xs text-fg-muted">Inspect time bucket<input aria-label="Inspect time bucket" type="range" min={0} max={Math.max(0,points.length-1)} value={hover??0} onChange={e=>setHover(Number(e.target.value))} className="flex-1"/><button disabled={!active} onClick={()=>{if(active)onSelect(active.label);}} className="rounded border border-border px-2 py-1 disabled:opacity-40">Select</button></label></div>
}
function MetricSparkline({values}:{values:number[]}) {
 if(values.length<2)return null;
 const stride=Math.max(1,Math.ceil(values.length/48));
 const sampled:number[]=[];
 for(let i=0;i<values.length;i+=stride){const part=values.slice(i,i+stride);sampled.push(part.reduce((a,b)=>a+b,0)/part.length);}
 const lo=Math.min(0,...sampled),hi=Math.max(0,...sampled),span=Math.max(1,hi-lo);
 const y=(v:number)=>30-(v-lo)/span*26;
 const path=sampled.map((v,i)=>(i?"L":"M")+(2+i*76/Math.max(1,sampled.length-1))+" "+y(v)).join(" ");
 return <svg viewBox="0 0 80 34" className="h-8 w-20 shrink-0 text-accent" aria-hidden="true"><line x1="2" x2="78" y1={y(0)} y2={y(0)} stroke="currentColor" opacity=".2"/><path d={path} fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round"/></svg>;
}
function GuidedAnalytics() {
  const { tenantId, loading: scopeLoading } = useTenant();
  const { hours, from, to } = useTimeRange();
  const [template, setTemplate] = useState("Overview");
  const [mappingDraft,setMappingDraft]=useState<string[]>([]);
  const [mappingTemplate,setMappingTemplate]=useState("Website");
  const [mappingSearch,setMappingSearch]=useState("");
  const [event, setEvent] = useState("");
  const [revision, setRevision] = useState(0);
  const [live,setLive]=useState(false);
  useEffect(()=>{if(!live)return;const timer=window.setInterval(()=>{if(document.visibilityState==="visible")setRevision(x=>x+1);},30000);return ()=>window.clearInterval(timer);},[live]);
  const [compare, setCompare] = useState(true);
  const [dimension,setDimension]=useState("");
  const [discover,setDiscover]=useState(false);
  const [widgets,setWidgets]=useState(["trend","insights","events","dimensions"]);
  const showDimensions=widgets.includes("dimensions");
  useEffect(()=>{setDimension("");setEvent("");setDiscover(false);setMappingDraft([]);setMappingSearch("");},[tenantId]);
  const [details,setDetails]=useState<{key:string;profiles?:{key:string;present:number;examples:string[]}[];keys:string[];rows:{group:string;value:number}[];truncated:boolean;error?:string}>();
  const [bucketFocus,setBucketFocus]=useState<string|null>(null);
  useEffect(()=>setBucketFocus(null),[tenantId,from,to,event,template]);
  const [priorTrend,setPriorTrend]=useState<{key:string;points:{label:string;value:number}[]}>();
  const [trend,setTrend]=useState<{key:string;points:{label:string;value:number}[];truncated:boolean;error?:string}>();
  const [state, setState] = useState<{key:string; current:ReportOption[]; previous:ReportOption[]; error?:string}>();
  const key = JSON.stringify([tenantId,from,to,compare]);
  const catalogCache=useRef(new Map<string,{at:number;current:ReportOption[];previous:ReportOption[]}>());
  const lastRevision=useRef(revision);
  useEffect(()=>{catalogCache.current.clear();},[tenantId]);
  useEffect(() => {
    if (scopeLoading || !to) return;
    let active = true;
    const force=lastRevision.current!==revision;lastRevision.current=revision;
    const cached=catalogCache.current.get(key);
    if(!force&&cached&&Date.now()-cached.at<30000){setState({key,current:cached.current,previous:cached.previous});return;}
    const range = {from_ts:from,to_ts:to};
    Promise.all([
      api.eventOptions(tenantId,"",true,range),
      compare && from>0 ? api.eventOptions(tenantId,"",true,{from_ts:Math.max(0,from-(to-from)-1),to_ts:from-1}) : Promise.resolve([]),
    ]).then(([current,previous]) => {
      if(active){catalogCache.current.delete(key);catalogCache.current.set(key,{at:Date.now(),current,previous});while(catalogCache.current.size>12){const oldest=catalogCache.current.keys().next().value;if(oldest===undefined)break;catalogCache.current.delete(oldest);}setState({key,current,previous});}
    }).catch(e=>{if(active)setState({key,current:[],previous:[],error:String(e.message??e)});});
    return ()=>{active=false;};
  },[key,tenantId,from,to,compare,revision,scopeLoading]);
  const ready = !scopeLoading && state?.key===key ? state : undefined;
  const [savedTenant,setSavedTenant]=useState("");
  const [saved,setSaved]=useState<{id:string;name:string;query:string}[]>([]);
  const [saveError,setSaveError]=useState("");
  const [saving,setSaving]=useState(false);
  const [savedRevision,setSavedRevision]=useState(0);
  useEffect(()=>{let active=true;setSaved([]);setSavedTenant("");setSaveError("");if(tenantId)api.analytics.views(tenantId).then(v=>{if(active){setSavedTenant(tenantId??"");setSaved(v.filter(x=>{try{return ["guided-report-v1","guided-mapping-v1"].includes(JSON.parse(x.query).kind)}catch{return false}}))}}).catch(()=>{if(active)setSaveError("Saved reports unavailable.")});return()=>{active=false};},[tenantId,savedRevision]);
  const mappings=new Map<string,string[]>();
  for(const v of savedTenant===tenantId?saved:[]){try{const m=JSON.parse(v.query);if(m.kind==="guided-mapping-v1"&&Array.isArray(m.events)&&typeof m.template==="string")mappings.set(m.template,m.events.filter((e:unknown)=>typeof e==="string"));}catch{}}
  const classify=(name:string,pattern:string,category:string)=>mappings.has(category)?mappings.get(category)!.includes(name):!pattern||new RegExp(pattern,"i").test(name.replace(/([a-z0-9])([A-Z])/g,"$1 $2").replace(/[^a-zA-Z0-9]+/g," "));
  const suggestedReports=reportTemplates.filter(t=>t.pattern).map(t=>({name:t.name,count:(ready?.current??[]).reduce((n,r)=>n+(classify(r.value,t.pattern,t.name)?r.count:0),0)})).filter(t=>t.count>0).sort((a,b)=>b.count-a.count).slice(0,3);
  const pattern = reportTemplates.find(t=>t.name===template)?.pattern ?? "";
  const matches = (row:ReportOption) => (!event || row.value===event) && classify(row.value,pattern,template);
  async function saveMapping(){if(!tenantId||saving||!mappingDraft.length||mappings.has(mappingTemplate)||savedTenant!==tenantId)return;setSaving(true);setSaveError("");try{await api.analytics.saveView(tenantId,"Event mapping: "+mappingTemplate,JSON.stringify({kind:"guided-mapping-v1",template:mappingTemplate,events:mappingDraft}));setSavedRevision(x=>x+1);setMappingDraft([]);}catch{setSaveError("Could not save event mapping.");}finally{setSaving(false);}}
  const rows = (ready?.current??[]).filter(matches).sort((a,b)=>b.count-a.count);
  const total = rows.reduce((n,r)=>n+r.count,0);
  const previous = (ready?.previous??[]).filter(matches).reduce((n,r)=>n+r.count,0);
  const change = previous ? (total-previous)/previous*100 : null;
  const available = ready && !ready.error;
  const types = event ? [event] : (pattern || mappings.has(template)) ? Array.from(new Set([...rows,...(ready?.previous??[]).filter(matches)].map(r=>r.value))).sort() : [];
  const typesKey=JSON.stringify(types);
  const focusStart=bucketFocus?Date.parse(bucketFocus.length===10?bucketFocus+"T00:00:00Z":bucketFocus.replace(" ","T")+":00Z"):NaN;
  const detailFrom=Number.isFinite(focusStart)?Math.max(from,focusStart):from;
  const detailTo=Number.isFinite(focusStart)?Math.min(to,focusStart+(Number(hours)>168?86400000:3600000)-1):to;
  const detailKey=JSON.stringify([tenantId,detailFrom,detailTo,typesKey,dimension,revision]);
  const trendKey=JSON.stringify([tenantId,from,to,typesKey,revision,hours]);
  useEffect(()=>{
    if(scopeLoading||!tenantId||!to||!available||!rows.length||!showDimensions||(!dimension&&!discover))return;
    const controller=new AbortController();
    const selectedTypes=JSON.parse(typesKey) as string[];
    if(selectedTypes.length>100){setDetails({key:detailKey,keys:[],rows:[],truncated:false,error:"Select one event to analyze this report."});return;}
    const body={tenant_id:tenantId,from_ts:detailFrom,to_ts:detailTo,types:selectedTypes,operation:"count",group:dimension,full_range:Boolean(dimension)};
    fetch("/api/proxy/query-explorer?mode=dimensions",{method:"POST",headers:{"Content-Type":"application/json"},signal:controller.signal,body:JSON.stringify(body)}).then(async r=>{if(!r.ok)throw Error("Dimension report unavailable");return r.json();}).then(r=>{if(!controller.signal.aborted)setDetails({key:detailKey,keys:r.keys??[],profiles:r.profiles??[],rows:r.rows??[],truncated:r.truncated});}).catch(e=>{if(!controller.signal.aborted)setDetails({key:detailKey,keys:[],rows:[],truncated:false,error:String(e.message)});});
    return ()=>controller.abort();
  },[detailKey,tenantId,detailFrom,detailTo,typesKey,dimension,discover,scopeLoading,available,rows.length,showDimensions]);
  const currentDetails=details?.key===detailKey?details:undefined;
  const currentTrend=trend?.key===trendKey?trend:undefined;
  const bucketSize=Number(hours)>168?86400000:3600000;
  const chartPoints=(()=>{if(!currentTrend||currentTrend.truncated||currentTrend.error)return currentTrend?.points??[];const start=Math.floor(from/bucketSize)*bucketSize,end=Math.floor(to/bucketSize)*bucketSize;if(!Number.isFinite(start)||!Number.isFinite(end)||end<start||(end-start)/bucketSize>=5000)return currentTrend.points;const values=new Map(currentTrend.points.map(p=>[p.label,p.value]));const points:{label:string;value:number}[]=[];for(let t=start;t<=end;t+=bucketSize){const iso=new Date(t).toISOString();const label=bucketSize===86400000?iso.slice(0,10):iso.slice(0,13).replace("T"," ")+":00";points.push({label,value:values.get(label)??0});}return points;})();
  const [knownKeys,setKnownKeys]=useState<{scope:string;profiles?:{key:string;present:number;examples:string[]}[];keys:string[]}>({scope:"",keys:[]});
  const schemaScope=JSON.stringify([tenantId,detailFrom,detailTo,typesKey]);
  useEffect(()=>{if(currentDetails&&!dimension&&!currentDetails.error)setKnownKeys({scope:schemaScope,keys:currentDetails.keys,profiles:currentDetails.profiles});},[currentDetails,dimension,schemaScope]);
  const keys=knownKeys.scope===schemaScope?knownKeys.keys:[];
  const preferred=template==="Technology"?/device|browser|platform|os/i:template==="Geography"?/country|region|city/i:template==="Audience"?/user|language|segment/i:null;
  const profileMap=new Map((knownKeys.scope===schemaScope?knownKeys.profiles??[]:[]).map(p=>[p.key,p]));
  const score=(k:string)=>{const p=profileMap.get(k);return (preferred?.test(k)?1e9:0)+(p&&p.examples.length>1?p.present:0);};
  const orderedKeys=[...keys].sort((a,b)=>score(b)-score(a)||a.localeCompare(b));
  const recommendations=orderedKeys.filter(k=>(profileMap.get(k)?.examples.length??0)>1).slice(0,4);
  const semanticHints=keys.filter(k=>preferred?.test(k)).slice(0,4);

  function moveWidget(w:string,delta:number){setWidgets(old=>{const i=old.indexOf(w),j=i+delta;if(i<0||j<0||j>=old.length)return old;const next=[...old];[next[i],next[j]]=[next[j],next[i]];return next;});}
  const [reportName,setReportName]=useState("");
  const [businessEvent,setBusinessEvent]=useState("");
  const [businessLabel,setBusinessLabel]=useState("Conversions");
  useEffect(()=>{setBusinessEvent("");setBusinessLabel("Conversions");},[tenantId]);
  async function saveReport(){if(!tenantId||!reportName.trim()||saving)return;setSaving(true);setSaveError("");try{await api.analytics.saveView(tenantId,reportName.trim(),JSON.stringify({kind:"guided-report-v1",template,event,dimension,compare,widgets,businessEvent,businessLabel}));setReportName("");setSavedRevision(x=>x+1);}catch{setSaveError("Could not save report.");}finally{setSaving(false);}}
  function openReport(query:string){try{const v=JSON.parse(query);if(v.kind!=="guided-report-v1"||!reportTemplates.some(t=>t.name===v.template))throw Error();setTemplate(v.template);setEvent(typeof v.event==="string"?v.event:"");setDimension(typeof v.dimension==="string"?v.dimension:"");setCompare(v.compare!==false);setBusinessEvent(typeof v.businessEvent==="string"?v.businessEvent:"");setBusinessLabel(typeof v.businessLabel==="string"?v.businessLabel:"Conversions");setWidgets(Array.isArray(v.widgets)?v.widgets.filter((w:unknown)=>typeof w==="string"&&["trend","dimensions","events","insights","journey"].includes(w)):["trend","dimensions","events","insights","journey"]);}catch{setSaveError("Invalid saved report.");}}
  async function deleteReport(id:string){if(!tenantId||saving)return;setSaving(true);try{await api.analytics.deleteView(tenantId,id);setSavedRevision(x=>x+1);}catch{setSaveError("Could not delete report.");}finally{setSaving(false);}}
  const [journey,setJourney]=useState<{key:string;funnel:{event_name:string;sessions:number;conversion_rate:number}[];retention:{cohort_period:string;cohort_size:number;retained_pct:number[]}[];errors:string[]}>();
  const journeyKey=JSON.stringify([tenantId,from,to,revision]);
  useEffect(()=>{if(scopeLoading||!tenantId||!to||!widgets.includes("journey"))return;let active=true;const range={from_ts:from,to_ts:to};Promise.allSettled([api.analytics.suggestedFunnel(tenantId,from,to).then(steps=>steps.length>1?api.analytics.funnel(tenantId,steps.slice(0,5).map(x=>x.event_name),from,to):[]),api.analytics.retention(tenantId,"daily",range)]).then(([f,r])=>{if(active)setJourney({key:journeyKey,funnel:f.status==="fulfilled"?f.value:[],retention:r.status==="fulfilled"?r.value:[],errors:[...(f.status==="rejected"?["Funnel unavailable"]:[]),...(r.status==="rejected"?["Retention unavailable"]:[])]});});return()=>{active=false};},[journeyKey,tenantId,from,to,scopeLoading,widgets]);
  const currentJourney=journey?.key===journeyKey?journey:undefined;
  const summaryRevision=useRef(revision);
  const [summary,setSummary]=useState<{key:string;users:number;sessions:number;matched:number;identified_events:number;error?:string}>();
  const summaryKey=JSON.stringify([tenantId,from,to,typesKey,revision]);
  const needsTrend=widgets.includes("trend")||widgets.includes("insights");
  useEffect(()=>{
    if(scopeLoading||!tenantId||!to||!available||!rows.length)return;
    const controller=new AbortController();
    const selectedTypes=JSON.parse(typesKey) as string[];
    const forceRefresh=summaryRevision.current!==revision;summaryRevision.current=revision;
    if(selectedTypes.length>100){setSummary({key:summaryKey,users:0,sessions:0,matched:0,identified_events:0,error:"Select one event for identity metrics."});setTrend({key:trendKey,points:[],truncated:false,error:"Select one event to view the trend."});return;}
    fetch("/api/proxy/query-explorer?mode=dimensions",{method:"POST",headers:{"Content-Type":"application/json"},signal:controller.signal,body:JSON.stringify({tenant_id:tenantId,from_ts:from,to_ts:to,types:selectedTypes,operation:"count",summary:true,cache_summary:!forceRefresh,include_trend:needsTrend,time_bucket:needsTrend?(Number(hours)>168?"day":"hour"):""})}).then(async r=>{if(!r.ok)throw Error("Identity metrics unavailable");return r.json();}).then(r=>{if(!controller.signal.aborted){setSummary({key:summaryKey,users:r.users,sessions:r.sessions,matched:r.matched,identified_events:r.identified_events});setTrend({key:trendKey,points:(r.rows??[]).map((x:{group:string;value:number})=>({label:x.group,value:x.value})),truncated:r.truncated});}}).catch(e=>{if(!controller.signal.aborted){setSummary({key:summaryKey,users:0,sessions:0,matched:0,identified_events:0,error:String(e.message)});setTrend({key:trendKey,points:[],truncated:false,error:String(e.message)});}});
    return ()=>controller.abort();
  },[summaryKey,trendKey,tenantId,from,to,typesKey,hours,scopeLoading,available,rows.length,needsTrend]);
  const priorKey=JSON.stringify([tenantId,from,to,typesKey,revision,hours]);
  const showTrend=widgets.includes("trend");
  useEffect(()=>{
    if(!compare||!showTrend||scopeLoading||!tenantId||!available||!rows.length||from<=0)return;
    const selectedTypes=JSON.parse(typesKey) as string[];if(selectedTypes.length>100)return;
    const controller=new AbortController(), start=Math.max(0,from-(to-from)-1), end=from-1;
    const size=Number(hours)>168?86400000:3600000;
    fetch("/api/proxy/query-explorer?mode=dimensions",{method:"POST",headers:{"Content-Type":"application/json"},signal:controller.signal,body:JSON.stringify({tenant_id:tenantId,from_ts:start,to_ts:end,types:selectedTypes,operation:"count",time_bucket:size===86400000?"day":"hour",full_range:true})}).then(async r=>{if(!r.ok)throw Error();return r.json();}).then(r=>{if(controller.signal.aborted||r.truncated)return;const values=new Map<string,number>((r.rows??[]).map((x:{group:string;value:number})=>[x.group,x.value]));const points:{label:string;value:number}[]=[];for(let t=Math.floor(start/size)*size;t<=end&&points.length<5000;t+=size){const iso=new Date(t).toISOString(),label=size===86400000?iso.slice(0,10):iso.slice(0,13).replace("T"," ")+":00";points.push({label,value:values.get(label)??0});}setPriorTrend({key:priorKey,points});}).catch(()=>{if(!controller.signal.aborted)setPriorTrend(undefined);});
    return()=>controller.abort();
  },[priorKey,compare,scopeLoading,tenantId,available,rows.length,from,to,typesKey,hours,showTrend]);
  const comparisonPoints=compare&&showTrend&&!currentTrend?.truncated&&!currentTrend?.error&&priorTrend?.key===priorKey&&priorTrend.points.length===chartPoints.length?priorTrend.points:undefined;
  const identity=summary?.key===summaryKey&&!summary.error?summary:undefined;
  const [realtime,setRealtime]=useState<{tenant:string;users:number;sessions:number;matched:number;error?:string}>();
  const [showRealtime,setShowRealtime]=useState(false);
  useEffect(()=>{if(!showRealtime||scopeLoading||!tenantId)return;let active=true;let busy=false;setRealtime(undefined);async function load(){if(busy||document.visibilityState!=="visible")return;busy=true;try{const now=Date.now();const response=await fetch("/api/proxy/query-explorer?mode=dimensions",{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({tenant_id:tenantId,from_ts:now-300000,to_ts:now,operation:"count",summary:true})});if(!response.ok)throw Error();const r=await response.json();if(active)setRealtime({tenant:tenantId,users:r.users,sessions:r.sessions,matched:r.matched});}catch{if(active)setRealtime({tenant:tenantId,users:0,sessions:0,matched:0,error:"Realtime report unavailable"});}finally{busy=false;}}void load();const timer=window.setInterval(load,15000);return()=>{active=false;window.clearInterval(timer)};},[showRealtime,scopeLoading,tenantId]);



  const currentByEvent=new Map(rows.map(r=>[r.value,r.count]));
  const priorByEvent=new Map((ready?.previous??[]).filter(matches).map(r=>[r.value,r.count]));
  const movers=compare?Array.from(new Set([...rows.map(r=>r.value),...priorByEvent.keys()])).map(name=>{const now=currentByEvent.get(name)??0;const before=priorByEvent.get(name)??0;return {name,now,before,delta:now-before};}).filter(x=>Math.max(x.now,x.before)>=20&&Math.abs(x.delta)>=10).sort((a,b)=>Math.abs(b.delta)-Math.abs(a.delta)).slice(0,3):[];
  const anomaly=(()=>{
    if(!needsTrend||!currentTrend||currentTrend.truncated||currentTrend.error||chartPoints.length<9)return null;
    const stamp=(label:string)=>Date.parse(label.length===10?label+"T00:00:00Z":label.replace(" ","T")+":00Z");
    const complete=chartPoints.filter(x=>{const t=stamp(x.label);return Number.isFinite(t)&&t>=from&&t+bucketSize<=to+1;});
    if(complete.length<8)return null;
    const last=complete[complete.length-1], prior=complete.slice(0,-1), lastTime=stamp(last.label);
    const period=bucketSize===86400000?7*86400000:86400000;
    const seasonal=prior.filter(x=>(lastTime-stamp(x.label))%period===0).slice(-28);
    const useSeasonal=seasonal.length>=7;
    const baseline=useSeasonal?seasonal:prior.slice(-28);
    const mean=baseline.reduce((n,x)=>n+x.value,0)/baseline.length;
    const sd=Math.sqrt(baseline.reduce((n,x)=>n+(x.value-mean)**2,0)/Math.max(1,baseline.length-1));
    const z=(last.value-mean)/Math.max(sd,Math.sqrt(mean),1);
    return mean>=10&&Math.abs(z)>=3?{last,mean,z,baselineCount:baseline.length,method:useSeasonal?(bucketSize===86400000?"same weekday (UTC)":"same hour of day (UTC)"):"recent complete buckets"}:null;
  })();
  const focusedPoint=bucketFocus?chartPoints.find(p=>p.label===bucketFocus):undefined;
  const businessCount=ready?.current.find(r=>r.value===businessEvent)?.count??0;
  function exportReport() {
    const csv = ["Event,Count",...rows.map(r=>'"'+r.value.replace(/^[=+@-]/,"'$&").replace(/"/g,'""')+'",'+r.count)].join("\n");
    const url=URL.createObjectURL(new Blob([csv],{type:"text/csv"}));
    const a=document.createElement("a");a.href=url;a.download="analytics.csv";a.click();URL.revokeObjectURL(url);
  }
  return <div className="grid items-start gap-5 xl:grid-cols-[180px_minmax(0,1fr)]"><nav aria-label="Analytics reports" className="rounded-xl border border-border p-2 xl:sticky xl:top-4"><p className="px-3 py-2 text-[10px] font-semibold uppercase tracking-widest text-fg-muted">Reports</p><div className="flex gap-1 overflow-x-auto xl:flex-col">{reportTemplates.map(t=><button key={t.name} title={t.description} aria-current={template===t.name?"page":undefined} onClick={()=>{setTemplate(t.name);setEvent("");setDimension("");setDiscover(false);}} className={"whitespace-nowrap rounded-md px-3 py-2 text-left text-sm transition-colors "+(template===t.name?"bg-accent/10 font-semibold text-accent":"text-fg-muted hover:bg-surface-2 hover:text-fg")}>{t.name}</button>)}</div></nav><main className="min-w-0 space-y-4">
    <div className="flex flex-wrap items-center justify-between gap-3"><div><h1 className="text-xl font-semibold text-fg">{template}</h1><p className="text-sm text-fg-muted">{reportTemplates.find(t=>t.name===template)?.description} · {formatTimeRange(hours)}</p></div><div className="flex gap-2"><button className="rounded border border-border px-3 py-2 text-sm" onClick={()=>setRevision(x=>x+1)}>Refresh</button><button disabled={!available} className="rounded border border-border px-3 py-2 text-sm disabled:opacity-50" onClick={exportReport}>Export CSV</button><button onClick={()=>window.print()} className="rounded border border-border px-3 py-2 text-sm print:hidden">Print / PDF</button></div></div>
    <details className="rounded border border-border p-3"><summary className="cursor-pointer text-sm font-medium">Saved reports & dashboard settings</summary><div className="mt-3 flex flex-wrap gap-3">{["trend","dimensions","events","insights","journey"].map(w=><label key={w} className="flex items-center gap-1 text-sm capitalize"><input type="checkbox" checked={widgets.includes(w)} onChange={e=>setWidgets(old=>e.target.checked?[...old,w]:old.filter(x=>x!==w))}/>{w}</label>)}<label className="flex items-center gap-1 text-sm"><input type="checkbox" checked={live} onChange={e=>setLive(e.target.checked)}/> Refresh every 30 seconds</label></div><div className="mt-3 flex flex-wrap gap-2">{widgets.map((w,i)=><div key={w} className="flex items-center gap-2 rounded border border-border px-2 py-1 text-xs capitalize">{w}<button aria-label={"Move "+w+" earlier"} disabled={i===0} onClick={()=>moveWidget(w,-1)} className="disabled:opacity-30">↑</button><button aria-label={"Move "+w+" later"} disabled={i===widgets.length-1} onClick={()=>moveWidget(w,1)} className="disabled:opacity-30">↓</button></div>)}</div><div className="mt-3 flex flex-wrap gap-2"><input aria-label="Report name" placeholder="Name this report" value={reportName} maxLength={100} onChange={e=>setReportName(e.target.value)} className="rounded border border-border bg-surface-2 px-3 py-2 text-sm"/><button disabled={!tenantId||!reportName.trim()||saving} onClick={()=>void saveReport()} className="rounded bg-accent px-3 py-2 text-sm text-accent-fg disabled:opacity-50">Save report</button></div>{saveError&&<p role="alert" className="mt-2 text-sm text-danger">{saveError}</p>}<div className="mt-2 space-y-2">{saved.filter(v=>{try{return JSON.parse(v.query).kind==="guided-report-v1"}catch{return false}}).map(v=><div key={v.id} className="flex justify-between gap-2 text-sm"><button onClick={()=>openReport(v.query)} className="text-accent">{v.name}</button><button disabled={saving} onClick={()=>{if(window.confirm("Delete this saved report?"))void deleteReport(v.id)}} className="text-danger">Delete</button></div>)}</div></details>
    <details className="rounded-lg border border-border p-3"><summary className="cursor-pointer text-sm font-medium">Report event mappings</summary><div className="mt-3 space-y-2"><select aria-label="Report category" value={mappingTemplate} onChange={e=>{setMappingTemplate(e.target.value);setMappingDraft(mappings.get(e.target.value)??[]);}} className="rounded border border-border bg-surface-2 p-2 text-sm">{reportTemplates.filter(t=>t.pattern).map(t=><option key={t.name}>{t.name}</option>)}</select><p className="text-xs text-fg-muted">Choose exact events to override automatic classification for this category.</p><input aria-label="Search events for mapping" placeholder="Find an event…" value={mappingSearch} onChange={e=>setMappingSearch(e.target.value)} className="w-full rounded border border-border bg-surface-2 p-2 text-sm"/><p className="text-xs text-fg-muted">{mappingDraft.length} selected</p><div className="max-h-48 overflow-auto">{Array.from(new Set([...(ready?.current??[]).map(r=>r.value),...mappingDraft])).filter(name=>name.toLowerCase().includes(mappingSearch.toLowerCase())).map(name=><label key={name} className="flex gap-2 py-1 text-xs"><input type="checkbox" checked={mappingDraft.includes(name)} onChange={e=>setMappingDraft(old=>e.target.checked?[...old,name]:old.filter(x=>x!==name))}/>{name}</label>)}</div><button disabled={saving||!mappingDraft.length||mappings.has(mappingTemplate)||savedTenant!==tenantId} onClick={()=>void saveMapping()} className="rounded border border-border px-3 py-2 text-xs disabled:opacity-40">Save mapping</button>{saved.filter(v=>{try{return JSON.parse(v.query).kind==="guided-mapping-v1"}catch{return false}}).map(v=><div key={v.id} className="flex justify-between text-xs"><span>{v.name}</span><button disabled={saving} onClick={()=>void deleteReport(v.id)} className="text-danger">Remove override</button></div>)}</div></details>
    <details className="rounded-lg border border-border"><summary className="cursor-pointer px-3 py-2 text-sm font-medium">Business metric {businessEvent&&available?` · ${businessLabel}: ${businessCount.toLocaleString()}`:""}</summary><div className="flex flex-wrap items-end gap-3 p-4"><label className="text-xs text-fg-muted">Metric name<input maxLength={80} value={businessLabel} onChange={e=>setBusinessLabel(e.target.value)} className="mt-1 block rounded border border-border bg-surface-2 p-2 text-sm"/></label><label className="text-xs text-fg-muted">Count this event<select value={businessEvent} onChange={e=>setBusinessEvent(e.target.value)} className="mt-1 block rounded border border-border bg-surface-2 p-2 text-sm"><option value="">Choose an event</option>{businessEvent&&!ready?.current.some(r=>r.value===businessEvent)&&<option value={businessEvent}>{businessEvent}</option>}{(ready?.current??[]).map(r=><option key={r.value} value={r.value}>{r.value}</option>)}</select></label>{businessEvent&&<div><p className="text-xs text-fg-muted">{businessLabel||"Business metric"}</p><p className="text-2xl font-semibold">{available?businessCount.toLocaleString():"—"}</p></div>}<p className="basis-full text-xs text-fg-muted">Counts occurrences of your chosen business event across the tenant and global time range; not revenue or a unique-user conversion rate.</p></div></details>

    <div className="flex flex-wrap items-center gap-3"><label className="text-sm">Event <select className="ml-2 rounded border border-border bg-surface-2 p-2" value={event} onChange={e=>{setEvent(e.target.value);setDimension("");}}><option value="">All matching events</option>{event && !ready?.current.some(r=>r.value===event) && <option value={event}>{event}</option>}{(ready?.current??[]).filter(r=>classify(r.value,pattern,template)).map(r=><option key={r.value} value={r.value}>{r.value}</option>)}</select></label><label className="flex items-center gap-2 text-sm"><input type="checkbox" checked={compare} onChange={e=>setCompare(e.target.checked)}/> Compare previous period</label></div>
    {template==="Overview"&&!event&&suggestedReports.length>0&&<div className="flex flex-wrap items-center gap-2 rounded-lg border border-border p-3"><span className="text-xs font-medium text-fg-muted">Suggested reports</span>{suggestedReports.map(t=><button key={t.name} onClick={()=>{setTemplate(t.name);setEvent("");setDimension("");setDiscover(false);}} className="rounded-md bg-surface-2 px-3 py-1.5 text-xs hover:text-accent">{t.name} · {t.count.toLocaleString()} events</button>)}<span className="text-[10px] text-fg-muted">Based on event names or saved mappings; categories may overlap.</span></div>}
    {!ready && <p role="status" className="text-sm text-fg-muted">Loading report…</p>}
    {ready?.error && <p role="alert" className="text-sm text-danger">{ready.error}</p>}
    <div className="grid gap-2 sm:grid-cols-3">{[{label:"Total events",value:total.toLocaleString(),help:"Recorded actions in this report",spark:currentTrend&&!currentTrend.truncated&&!currentTrend.error?chartPoints.map(p=>p.value):undefined},{label:"Event types",value:String(rows.length),help:"Different tracked actions",spark:undefined},{label:"Change",value:compare&&change!==null?(change>=0?"+":"")+change.toFixed(1)+"%":"—",help:"Compared with the preceding equal period",spark:comparisonPoints?.map((p,i)=>chartPoints[i].value-p.value)}].map(m=><Card key={m.label}><div className="p-3"><p className="text-xs font-medium text-fg-muted">{m.label}</p><div className="flex items-center justify-between gap-2"><p className="mt-1 text-2xl font-semibold tabular-nums">{available?m.value:"—"}</p>{available&&m.spark&&<MetricSparkline values={m.spark}/>}</div><p className="text-xs text-fg-muted">{m.help}</p></div></Card>)}</div>
    <div className="grid gap-3 sm:grid-cols-3">{[{label:"Identified active users",value:identity?.users,help:"Distinct non-empty user IDs"},{label:"Active sessions",value:identity?.sessions,help:"Distinct non-empty session IDs"},{label:"Identity coverage",value:identity&&identity.matched?Math.round(identity.identified_events/identity.matched*100)+"%":undefined,help:"Events with a recorded user ID"}].map(m=><Card key={m.label}><div className="p-4"><p className="text-xs text-fg-muted">{m.label}</p><p className="mt-1 text-2xl font-semibold">{m.value?.toLocaleString()??"—"}</p><p className="text-xs text-fg-muted">{m.help}</p></div></Card>)}</div>
    {summary?.key===summaryKey&&summary.error&&<p className="text-xs text-danger">{summary.error}</p>}
    <Card><CardHeader title="Realtime · last 5 minutes" action={<button className="text-sm text-accent" onClick={()=>setShowRealtime(x=>!x)}>{showRealtime?"Stop":"Start live report"}</button>}/>{showRealtime&&<div className="grid gap-3 p-4 sm:grid-cols-3">{realtime?.tenant===tenantId&&realtime.error?<p role="alert" className="text-sm text-danger">{realtime.error}</p>:[{label:"Identified users",value:realtime?.users},{label:"Sessions",value:realtime?.sessions},{label:"Events",value:realtime?.matched}].map(m=><div key={m.label}><p className="text-xs text-fg-muted">{m.label}</p><p className="text-xl font-semibold">{realtime?.tenant===tenantId?m.value?.toLocaleString()??"—":"—"}</p></div>)}<p className="text-xs text-fg-muted sm:col-span-3">Entire selected tenant, independent of global time and event filters. Refreshes every 15 seconds.</p></div>}</Card>
    {available && !rows.length && <p className="rounded border border-border p-4 text-sm text-fg-muted">No matching events. Try Overview or change the global time range. Templates match event names automatically.</p>}
    <div className="grid gap-4 lg:grid-cols-2">{widgets.map(w=><section key={w} className={"min-w-0 "+(["trend","journey"].includes(w)?"lg:col-span-2":"")}>{w==="trend"&&(available && rows.length>0 && widgets.includes("trend") && <Card><CardHeader title="Activity over time"/><div className="p-4">{currentTrend?.error?<p className="text-sm text-danger">{currentTrend.error}</p>:chartPoints.length?<WorkspaceTrend points={chartPoints} previous={comparisonPoints} onSelect={label=>{setBucketFocus(label);setDiscover(true);}}/>:<p className="text-sm text-fg-muted">{currentTrend?"No activity available.":"Loading activity…"}</p>}{focusedPoint&&<div className="mt-3 flex items-center justify-between gap-2 rounded-lg border border-border p-3 text-sm"><span>{focusedPoint.label} UTC · <strong>{focusedPoint.value.toLocaleString()} events</strong></span><button onClick={()=>setBucketFocus(null)} className="text-xs text-accent">Clear selection</button></div>}{currentTrend?.truncated&&<p className="text-xs text-fg-muted">Partial results — analysis limits reached.</p>}</div></Card>)}{w==="dimensions"&&(available && rows.length>0 && widgets.includes("dimensions") && <Card><CardHeader title={bucketFocus?"Breakdown · "+bucketFocus+" UTC":"Break down activity"}/><div className="space-y-3 p-4"><label className="text-sm"><button onClick={()=>setDiscover(true)} className="mr-2 rounded border border-border px-2 py-1 text-xs text-accent">Discover properties</button>Group by <select value={dimension} onChange={e=>setDimension(e.target.value)} className="ml-2 rounded border border-border bg-surface-2 p-2"><option value="">Choose a property</option>{dimension&&!orderedKeys.includes(dimension)&&<option value={dimension}>{dimension}</option>}{orderedKeys.map(k=><option key={k} value={k}>{k}</option>)}</select></label>{!dimension&&semanticHints.length>0&&<p className="text-xs text-fg-muted">Relevant properties for {template}: {semanticHints.join(", ")}</p>}{!dimension&&recommendations.length>0&&<div className="flex flex-wrap items-center gap-2"><span className="text-xs text-fg-muted">Recommended breakdowns</span>{recommendations.map(k=><button key={k} onClick={()=>setDimension(k)} className="rounded-full border border-border px-3 py-1 text-xs hover:border-accent hover:text-accent">{k}</button>)}</div>}{currentDetails?.error?<p className="text-sm text-danger">{currentDetails.error}</p>:dimension&&currentDetails?<><BarBreakdown data={[...currentDetails.rows].sort((a,b)=>b.value-a.value).slice(0,15).map(r=>({label:r.group,value:r.value}))}/>{currentDetails.truncated&&<p className="text-xs text-fg-muted">Partial results — analysis limits reached.</p>}</>:<p className="text-sm text-fg-muted">{dimension?"Loading breakdown…":keys.length?"Choose an automatically discovered event property.":currentDetails?"No properties discovered." :discover?"Discovering properties…":"Discover properties when needed."}</p>}</div></Card>)}{w==="events"&&(widgets.includes("events")&&<Card><CardHeader title="Top events"/><div className="space-y-3 p-4">{rows.slice(0,10).map(r=><button key={r.value} onClick={()=>{setEvent(r.value);setDimension("");}} className="block w-full text-left"><div className="flex justify-between gap-2 text-sm"><span className="truncate">{r.value}</span><span>{r.count.toLocaleString()}</span></div><div className="mt-1 h-2 rounded bg-surface-2"><div className="h-2 rounded bg-accent" style={{width:(total?r.count/total*100:0)+"%"}}/></div></button>)}</div></Card>)}{w==="insights"&&(widgets.includes("insights")&&<Card><CardHeader title="Activity intelligence"/><div className="space-y-3 p-4 text-sm text-fg-muted">{available && rows.length>0?<><p>{rows[0].value} is the most frequent action in this report ({(rows[0].count/Math.max(total,1)*100).toFixed(1)}% of recorded actions).</p>{compare&&change!==null&&<p>Activity {change>=0?"increased":"decreased"} by {Math.abs(change).toFixed(1)}% compared with the previous period.</p>}{anomaly&&<div className="rounded-lg border border-border p-3"><p className="font-medium text-fg">Unusual activity {anomaly.z>0?"spike":"drop"}</p><p className="text-xs">{anomaly.last.label} UTC · {anomaly.last.value.toLocaleString()} events versus {Math.round(anomaly.mean).toLocaleString()} average across {anomaly.baselineCount} earlier buckets ({anomaly.method}).</p><p className="text-xs">Deviation score {anomaly.z.toFixed(1)}; exploratory signal; not a formal significance test.</p></div>}{compare&&movers.map(m=><button key={m.name} onClick={()=>{setEvent(m.name);setDimension("");}} className="block w-full rounded-lg border border-border p-3 text-left hover:bg-surface-2"><div className="flex items-center justify-between gap-2"><span className="truncate font-medium text-fg">{m.name}</span><span className={"font-semibold tabular-nums "+(m.delta>0?"text-emerald-500":"text-amber-500")}>{m.delta>0?"+":""}{m.delta.toLocaleString()}</span></div><p className="mt-1 text-xs">{m.before.toLocaleString()} → {m.now.toLocaleString()} events · {m.before?((m.delta/m.before)*100).toFixed(1)+"% change":"new activity"}</p></button>)}{identity&&identity.matched>0&&identity.identified_events/identity.matched<0.5&&<p className="rounded border border-border p-2 text-xs">Identity coverage is below 50%. Review user identification before interpreting audience metrics.</p>}{compare&&!movers.length&&<p className="text-xs">No event changes meet the minimum volume and absolute-change thresholds.</p>}<p className="text-xs">Ranked by absolute event-count change; not a statistical anomaly test.</p></>:<p>Insights appear when matching data is available.</p>}</div></Card>)}{w==="journey"&&(widgets.includes("journey")&&<div className="grid gap-4 lg:grid-cols-2"><Card><CardHeader title="Suggested conversion funnel"/><div className="space-y-2 p-4">{currentJourney?.funnel.map((step,i)=><div key={i} className="text-sm"><div className="flex justify-between gap-2"><span>{i+1}. {step.event_name}</span><span>{step.sessions.toLocaleString()} sessions</span></div><div className="mt-1 h-2 rounded bg-surface-2"><div className="h-2 rounded bg-accent" style={{width:Math.min(100,step.sessions/Math.max(1,currentJourney.funnel[0].sessions)*100)+"%"}}/></div></div>)}{!currentJourney?.funnel.length&&<p className="text-sm text-fg-muted">{currentJourney?currentJourney.errors.includes("Funnel unavailable")?"Funnel unavailable.":"Not enough events for a suggested funnel.":"Loading funnel…"}</p>}<p className="text-xs text-fg-muted">Suggested steps, not a configured business conversion. Uses the entire tenant and global time range.</p><Link href="/analytics/funnel" className="text-sm text-accent">Configure funnel</Link></div></Card><Card><CardHeader title="Returning sessions"/><div className="overflow-x-auto p-4">{currentJourney?.retention.length?<table className="w-full text-left text-xs"><thead><tr><th className="p-2">Cohort</th><th>Size</th><th>Day 1</th><th>Day 7</th></tr></thead><tbody>{currentJourney.retention.slice(-10).map(r=><tr key={r.cohort_period} className="border-t border-border"><td className="p-2">{r.cohort_period}</td><td>{r.cohort_size}</td><td>{r.retained_pct[1]==null?"—":r.retained_pct[1].toFixed(1)+"%"}</td><td>{r.retained_pct[7]==null?"—":r.retained_pct[7].toFixed(1)+"%"}</td></tr>)}</tbody></table>:<p className="text-sm text-fg-muted">{currentJourney?currentJourney.errors.includes("Retention unavailable")?"Retention unavailable.":"No cohorts available.":"Loading retention…"}</p>}<p className="mt-2 text-xs text-fg-muted">Entire tenant and global time range.</p><Link href="/analytics/retention" className="text-sm text-accent">Full retention report</Link></div></Card></div>)}</section>)}</div>
    <Card><CardHeader title="Explore further"/><div className="grid gap-2 p-4 sm:grid-cols-3">{[{href:"/analytics/funnel",title:"Conversions & funnels",desc:"See where users drop off"},{href:"/analytics/retention",title:"Retention",desc:"See whether users return"},{href:"/dimensions",title:"Dimensions",desc:"Explore devices, geography and other properties"}].map(l=><Link key={l.href} href={l.href} className="rounded border border-border p-3 hover:border-accent"><span className="block text-sm font-medium text-accent">{l.title}</span><span className="text-xs text-fg-muted">{l.desc}</span></Link>)}</div></Card>
  </main></div>;
}
export default function AnalyticsPage() {
  const [advanced,setAdvanced]=useState(()=>typeof window!=="undefined"&&new URLSearchParams(window.location.search).has("events"));
  return <div><div className="mb-4 flex gap-2 border-b border-border pb-2">{["Reports","Advanced explorer"].map((label,i)=><button key={label} aria-pressed={advanced===Boolean(i)} onClick={()=>setAdvanced(Boolean(i))} className={"rounded px-3 py-2 text-sm "+(advanced===Boolean(i)?"bg-accent text-accent-fg":"text-fg-muted")}>{label}</button>)}</div>{advanced?<AdvancedAnalytics/>:<GuidedAnalytics/>}</div>;
}
