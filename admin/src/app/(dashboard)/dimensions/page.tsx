"use client";
import {useState,useEffect} from "react";
import {BarChart,Bar,XAxis,YAxis,Tooltip,ResponsiveContainer,PieChart,Pie,LineChart,Line} from "recharts";
import {useTenant} from "@/components/TenantContext";
import {useTimeRange} from "@/components/TimeRangeContext";
import {translate,useLanguage} from "@/components/LanguageContext";
type Result={types:string[];keys:string[];rows:{group:string;split:string;value:number;count:number}[];scanned:number;truncated:boolean};
const control="rounded border border-border bg-surface px-2 py-1 text-xs";
export default function DimensionsPage(){
 const {tenantId}=useTenant();const {from,to}=useTimeRange();const {language}=useLanguage();const t=(s:string)=>translate(s,language);
 const [types,setTypes]=useState<string[]>([]);const [group,setGroup]=useState("");const [split,setSplit]=useState("");const [metric,setMetric]=useState("");const [operation,setOperation]=useState("count");const [explode,setExplode]=useState(false);const [view,setView]=useState("Table");const [data,setData]=useState<Result|null>(null);const [busy,setBusy]=useState(false);const [error,setError]=useState("");const [search,setSearch]=useState("");
 useEffect(()=>{
 if(!tenantId||!to){setData(null);return}
 const controller=new AbortController();setBusy(true);setError("");setData(null);
 fetch("/api/proxy/query-explorer?mode=dimensions",{method:"POST",headers:{"Content-Type":"application/json"},signal:controller.signal,body:JSON.stringify({tenant_id:tenantId,from_ts:from,to_ts:to,types,group,split,metric,operation,explode})}).then(async r=>{if(!r.ok)throw Error("Dimension analysis failed");return await r.json() as Result}).then(r=>{if(!controller.signal.aborted)setData(r)}).catch(()=>{if(!controller.signal.aborted)setError("Dimension analysis failed")}).finally(()=>{if(!controller.signal.aborted)setBusy(false)});
 return ()=>controller.abort();
 },[tenantId,from,to,types,group,split,metric,operation,explode]);
 const rows=(data?.rows||[]).filter(r=>(r.group+" "+r.split).toLowerCase().includes(search.toLowerCase()));
 const chart=rows.slice(0,30).map(r=>({...r,label:r.split?r.group+" / "+r.split:r.group}));
 const keys=[...new Set([...(data?.keys||[]),group,split==="@event"?"":split,metric].filter(Boolean))];
 function exportJSON(){const url=URL.createObjectURL(new Blob([JSON.stringify(data,null,2)],{type:"application/json"}));const a=document.createElement("a");a.href=url;a.download="dimensions.json";a.click();setTimeout(()=>URL.revokeObjectURL(url),1000)}
 return <main className="space-y-3 p-4"><header><h1 className="text-xl font-semibold">{t("Dimensions Explorer")}</h1><p className="text-xs text-fg-muted">{t("Analyze event dimensions using the global tenant and time range")}</p></header>
 {!tenantId?<p>{t("Select a tenant before running a query.")}</p>:<>
 <section className="space-y-3 rounded-lg border border-border bg-surface p-3">
 <div className="flex flex-wrap items-center gap-2"><span className="text-xs font-medium">{t("Event types")}</span><select className={control} aria-label={t("Add event type")} value="" onChange={e=>{if(e.target.value)setTypes(v=>[...v,e.target.value])}}><option value="">{t("Add event type")}</option>{data?.types.filter(x=>!types.includes(x)).map(x=><option key={x}>{x}</option>)}</select>{types.map(x=><button className={control} key={x} title={t("Remove event type")} onClick={()=>setTypes(v=>v.filter(y=>y!==x))}>{x} ×</button>)}<button className={control} onClick={()=>setTypes([])}>{t("All event types")}</button></div>
 <div className="flex flex-wrap gap-3">
 <label className="text-xs">{t("Group by")} <select className={control} value={group} onChange={e=>setGroup(e.target.value)}><option value="">{t("Select dimension")}</option>{keys.map(k=><option key={k}>{k}</option>)}</select></label>
 <label className="text-xs">{t("Split by")} <select className={control} value={split} onChange={e=>setSplit(e.target.value)}><option value="">{t("None")}</option><option value="@event">{t("Event types")}</option>{keys.map(k=><option key={k}>{k}</option>)}</select></label>
 <label className="text-xs">{t("Aggregation")} <select className={control} value={operation} onChange={e=>setOperation(e.target.value)}>{["count","distinct","sum","avg","min","max"].map(k=><option key={k} value={k}>{t(k)}</option>)}</select></label>
 {operation!=="count"&&<label className="text-xs">{t("Metric dimension")} <select className={control} value={metric} onChange={e=>setMetric(e.target.value)}><option value="">{t("Select dimension")}</option>{keys.map(k=><option key={k}>{k}</option>)}</select></label>}
 <label className="text-xs" title={t("Count each unique array item separately")}><input type="checkbox" checked={explode} onChange={e=>setExplode(e.target.checked)}/> {t("Explode arrays")}</label>
 </div><div className="flex flex-wrap gap-1">{keys.slice(0,10).map(k=><button key={k} className={control} title={t("Group by")} onClick={()=>setGroup(k)}>{k}</button>)}</div>
 </section>
 {busy&&<p role="status" className="text-xs">{t("Loading…")}</p>}{error&&<p role="alert">{t(error)}</p>}
 {data&&<section className="space-y-2 rounded-lg border border-border bg-surface p-3"><div className="flex flex-wrap items-center gap-2 text-xs"><span>{t("Events scanned")}: {data.scanned}</span><span>{t("Rows")}: {rows.length}</span><input className={control} placeholder={t("Search")} aria-label={t("Search")} value={search} onChange={e=>setSearch(e.target.value)}/><select className={control} aria-label={t("View")} value={view} onChange={e=>setView(e.target.value)}>{["Table","Bar chart","Line chart","Pie chart"].map(k=><option key={k}>{t(k)}</option>)}</select><button className={control} onClick={exportJSON}>{t("Export JSON")}</button></div>
 {data.truncated&&<p role="status" className="text-xs">{t("Partial results: analysis limits reached")}</p>}
 {!group&&<p className="text-xs">{t("Select dimension")}</p>}
 {view!=="Table"&&<div className="h-72" dir="ltr"><ResponsiveContainer width="100%" height="100%">{view==="Bar chart"?<BarChart data={chart}><XAxis dataKey="label"/><YAxis/><Tooltip/><Bar dataKey="value" fill="#6366f1"/></BarChart>:view==="Line chart"?<LineChart data={chart}><XAxis dataKey="label"/><YAxis/><Tooltip/><Line dataKey="value" stroke="#6366f1"/></LineChart>:<PieChart><Tooltip/><Pie data={chart.filter(r=>r.value>0)} dataKey="value" nameKey="label" fill="#6366f1"/></PieChart>}</ResponsiveContainer></div>}
 <div className="max-h-[55vh] overflow-auto"><table className="w-full text-xs"><thead className="sticky top-0 bg-surface"><tr>{[group,split,operation,"count"].map((k,i)=><th className="border-b border-border px-2 py-1 text-start" key={i}>{t(k)}</th>)}</tr></thead><tbody>{rows.map((r,i)=><tr key={i}>{[r.group,r.split,r.value,r.count].map((v,j)=><td key={j} title={String(v)} className="max-w-48 truncate border-b border-border px-2 py-1">{v}</td>)}</tr>)}</tbody></table></div></section>}
 </>}</main>
}
