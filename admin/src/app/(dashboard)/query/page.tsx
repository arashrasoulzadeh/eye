"use client";

import { useState, useEffect, useRef } from "react";
import { ResponsiveContainer, BarChart, Bar, LineChart, Line, PieChart, Pie, XAxis, YAxis, Tooltip } from "recharts";
import { useTenant } from "@/components/TenantContext";
import { useTimeRange } from "@/components/TimeRangeContext";
import { translate, useLanguage } from "@/components/LanguageContext";

type Result = { columns: string[]; rows: unknown[][]; truncated: boolean; duration_ms: number };
type Saved = { name: string; sql: string };
const examples = [
  { name: "Answer streak", sql: "SELECT * FROM events WHERE events.type = 'answer' AND event.streak > 1 ORDER BY timestamp DESC LIMIT 100;" },
  { name: "Event counts", sql: "SELECT type, COUNT(*) AS total FROM events GROUP BY type ORDER BY total DESC;" },
];
const fields = ["event_id","tenant_id","session_id","user_id","type","timestamp","ingest_ts","payload","dims","source","is_aggregated","aggregation_id","experiment_id","variant"];
const button = "rounded-lg border border-border px-3 py-2 text-sm hover:bg-surface-2 disabled:opacity-50";
function cell(value: unknown): string { return value == null ? "" : typeof value === "object" ? JSON.stringify(value) : String(value); }
function csv(value: unknown): string {
  const text = cell(value);
  return '"' + (/^[\s]*[=+@-]/.test(text) && typeof value === "string" ? "'" + text : text).replaceAll('"','""') + '"';
}
function download(name: string, body: string, type: string) {
  const url = URL.createObjectURL(new Blob([body], {type}));
  const a = document.createElement("a"); a.href = url; a.download = name; a.click();
  setTimeout(() => URL.revokeObjectURL(url), 1000);
}

export default function QueryPage() {
  const { tenantId, loading } = useTenant();
  const { from, to } = useTimeRange();
  const { language } = useLanguage();
  const t = (key: string) => translate(key, language);
  const [sql, setSql] = useState(examples[0].sql);
  const [result, setResult] = useState<Result | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [scopedTime, setScopedTime] = useState(true);
  const [page, setPage] = useState(0);
  const editor = useRef<HTMLTextAreaElement>(null);
  const highlight = useRef<HTMLPreElement>(null);
  const [cursor, setCursor] = useState(0);
  const [properties, setProperties] = useState<string[]>([]);
  const prefix = sql.slice(0,cursor).match(/[a-zA-Z_][a-zA-Z_0-9.]*$/)?.[0] || "";
  const suggestions = prefix ? [...fields,...properties,"SELECT","FROM","events","WHERE","AND","OR","GROUP BY","ORDER BY","HAVING","LIMIT","COUNT","SUM","AVG","MIN","MAX","DISTINCT"].filter(w=>w.toLowerCase().startsWith(prefix.toLowerCase()) && w.toLowerCase()!==prefix.toLowerCase()).slice(0,8) : [];
  function insert(word:string) {
    const start = cursor-prefix.length;
    setSql(sql.slice(0,start)+word+sql.slice(cursor));
    setCursor(start+word.length);
    requestAnimationFrame(()=>{editor.current?.focus();editor.current?.setSelectionRange(start+word.length,start+word.length);});
  }
  const [chart, setChart] = useState("Table");
  const [xColumn, setXColumn] = useState(0);
  const [yColumn, setYColumn] = useState(1);
  const [sort, setSort] = useState<{column:number; desc:boolean} | null>(null);
  const sortedRows = result ? [...result.rows].sort((a,b)=>{
    if (!sort) return 0;
    const x=a[sort.column],y=b[sort.column];
    const cmp=typeof x==="number"&&typeof y==="number"?x-y:cell(x).localeCompare(cell(y),undefined,{numeric:true});
    return sort.desc?-cmp:cmp;
  }) : [];
  const chartData = sortedRows.slice(0,200).flatMap(row=>{
    const value=row[yColumn];
    return typeof value==="number" || (typeof value==="string" && value.trim()!=="" && Number.isFinite(Number(value))) ? [{label:cell(row[xColumn]),value:Number(value)}] : [];
  });
  const [name, setName] = useState("");
  const [history, setHistory] = useState<string[]>([]);
  const [saved, setSaved] = useState<Saved[]>([]);
  const key = "eye.query." + tenantId;
  useEffect(() => {
    try {
      const stored = JSON.parse(localStorage.getItem(key) || "{}");
      setHistory(Array.isArray(stored.history) ? stored.history : []);
      setSaved(Array.isArray(stored.saved) ? stored.saved : []);
    } catch { setHistory([]); setSaved([]); }
    setResult(null);
    setProperties([]);
    if (!tenantId) return;
    const controller = new AbortController();
    fetch("/api/proxy/query-explorer?tenant_id="+encodeURIComponent(tenantId),{signal:controller.signal})
      .then(r=>r.ok?r.json():[]).then(data=>{if(!controller.signal.aborted&&Array.isArray(data))setProperties(data);}).catch(()=>undefined);
    return ()=>controller.abort();
  }, [key, tenantId]);
  function persist(h: string[], s: Saved[]) {
    setHistory(h); setSaved(s);
    try { localStorage.setItem(key, JSON.stringify({history:h,saved:s})); }
    catch { setError("Storage unavailable"); }
  }
  async function run() {
    if (!tenantId || loading || busy || !sql.trim() || (scopedTime && !to)) return;
    setBusy(true); setError(""); setResult(null); setPage(0); setSort(null); setXColumn(0); setYColumn(1);
    try {
      const response = await fetch("/api/proxy/query-explorer", {
        method:"POST", headers:{"Content-Type":"application/json"},
        body:JSON.stringify({tenant_id:tenantId,sql,from_ts:scopedTime?from:0,to_ts:scopedTime?to:0}),
      });
      if (!response.ok) throw new Error(await response.text());
      setResult(await response.json() as Result);
      persist([sql,...history.filter(item=>item!==sql)].slice(0,30),saved);
    } catch (e) { setError(e instanceof Error ? e.message : "Query failed"); }
    finally { setBusy(false); }
  }
  return <main className="space-y-5 p-6">
    <header><h1 className="text-2xl font-semibold">{t("Query Explorer")}</h1><p className="text-sm text-fg-muted">{t("Read-only event SQL")}</p></header>
    {!tenantId && <p role="status" className="rounded-lg border border-border p-3">{t("Select a tenant before running a query.")}</p>}
    <div className="grid gap-4 lg:grid-cols-[minmax(0,1fr)_16rem]">
      <section className="space-y-3 rounded-xl border border-border bg-surface p-4">
        <div className="relative rounded-lg border border-border bg-surface-2" dir="ltr"><pre ref={highlight} aria-hidden="true" className="pointer-events-none absolute inset-0 m-0 overflow-hidden whitespace-pre-wrap break-words p-4 font-mono text-sm leading-6">{sql.split(/('(?:[^']|'')*'|\b(?:SELECT|FROM|WHERE|AND|OR|JOIN|LEFT|INNER|ON|AS|GROUP|BY|ORDER|HAVING|LIMIT|OFFSET|IN|EXISTS|NOT|NULL|DISTINCT|COUNT|SUM|AVG|MIN|MAX)\b|\b\d+(?:\.\d+)?\b)/gi).map((token,i)=><span key={i} className={token.startsWith("'")?"text-emerald-600 dark:text-emerald-400":i%2===1?"text-violet-600 dark:text-violet-400":"text-fg"}>{token}</span>)}{"\n"}</pre><textarea ref={editor} onScroll={e=>{if(highlight.current){highlight.current.scrollTop=e.currentTarget.scrollTop;highlight.current.scrollLeft=e.currentTarget.scrollLeft;}}} onSelect={e=>setCursor(e.currentTarget.selectionStart)} aria-label={t("Query Explorer")} dir="ltr" spellCheck={false} rows={10} value={sql} onChange={e=>{setSql(e.target.value);setCursor(e.target.selectionStart);}} onKeyDown={e=>{if((e.ctrlKey||e.metaKey)&&e.key==="Enter"){e.preventDefault();void run();}}} className="relative block w-full resize-y bg-transparent p-4 font-mono text-sm leading-6 text-transparent caret-current outline-none focus:ring-2 focus:ring-accent/30" style={{caretColor:"var(--color-fg, currentColor)"}} /></div>
        <div className="flex flex-wrap gap-1">{suggestions.map(word=><button key={word} className={button+" font-mono"} onClick={()=>insert(word)}>{word}</button>)}</div>
        <div className="flex flex-wrap items-center gap-2">
          <button className={button+" bg-accent text-accent-fg"} disabled={busy||loading||!tenantId||!sql.trim()||(scopedTime&&!to)} onClick={()=>void run()}>{t(busy?"Running query…":"Run query")}</button>
          <label className="text-sm"><input type="checkbox" checked={scopedTime} onChange={e=>setScopedTime(e.target.checked)} /> {t("Use the global time range")}</label>
          <input className="rounded-lg border border-border bg-surface p-2 text-sm" aria-label={t("Query name")} placeholder={t("Query name")} value={name} onChange={e=>setName(e.target.value)} />
          <button className={button} disabled={!tenantId||!name.trim()||!sql.trim()||busy} onClick={()=>{persist(history,[{name:name.trim(),sql},...saved.filter(s=>s.name!==name.trim())].slice(0,50));setName("");}}>{t("Save query")}</button>
        </div>
        {error && <p role="alert" className="whitespace-pre-wrap text-sm text-danger">{t("Query failed")}: {t(error)}</p>}
      </section>
      <aside className="space-y-4 rounded-xl border border-border bg-surface p-4">
        <h2 className="font-semibold">{t("Schema explorer")}</h2>
        <code dir="ltr">events</code>
        <div className="flex flex-wrap gap-1">{[...fields,...properties].map(field=><button key={field} className="rounded border border-border px-2 py-1 font-mono text-xs" onClick={()=>insert(field)}>{field}</button>)}</div>
        <p className="text-xs text-fg-muted">{t("Custom properties")}: <code>payload.streak, dims.level, streak</code></p>
        <h2 className="font-semibold">{t("Query examples")}</h2>
        {examples.map(e=><button key={e.name} className={button+" block w-full text-start"} onClick={()=>setSql(e.sql)}>{t(e.name)}</button>)}
        <h2 className="font-semibold">{t("Saved queries")}</h2>
        {saved.map(s=><div key={s.name} className="flex gap-1"><button className="min-w-0 flex-1 truncate text-start text-sm" onClick={()=>setSql(s.sql)}>{s.name}</button><button className={button} disabled={busy} onClick={()=>persist(history,saved.filter(x=>x.name!==s.name))}>{t("Delete")}</button></div>)}
        <h2 className="font-semibold">{t("Query history")}</h2>
        {history.map(h=><button key={h} dir="ltr" title={h} className="block w-full truncate text-left font-mono text-xs" onClick={()=>setSql(h)}>{h}</button>)}
      </aside>
    </div>
    {result && <section className="space-y-3 rounded-xl border border-border bg-surface p-4">
      <div className="flex flex-wrap items-center gap-3"><h2 className="font-semibold">{t("Results")}</h2><span className="text-sm">{t("Rows")}: {result.rows.length} · {t("Execution time (ms)")}: {result.duration_ms}</span>
        <button className={button} onClick={()=>download("query.csv",[result.columns.map(csv).join(","),...result.rows.map(row=>row.map(csv).join(","))].join("\r\n"),"text/csv")}>{t("Export CSV")}</button>
        <button className={button} onClick={()=>download("query.json",JSON.stringify(result,null,2),"application/json")}>{t("Export JSON")}</button>
      </div>
      {result.truncated && <p>{t("Results limited to 1000 rows.")}</p>}
      {!result.rows.length && <p>{t("No results")}</p>}
      <div className="flex flex-wrap gap-2">
        <select aria-label={t("View")} className={button} value={chart} onChange={e=>setChart(e.target.value)}>{["Table","Bar chart","Line chart","Pie chart"].map(v=><option key={v} value={v}>{t(v)}</option>)}</select>
        {chart!=="Table" && <><select aria-label="X" className={button} value={xColumn} onChange={e=>setXColumn(Number(e.target.value))}>{result.columns.map((c,i)=><option key={i} value={i}>{c}</option>)}</select><select aria-label="Y" className={button} value={yColumn} onChange={e=>setYColumn(Number(e.target.value))}>{result.columns.map((c,i)=><option key={i} value={i}>{c}</option>)}</select></>}
      </div>
      {chart!=="Table" && <div className="h-80" dir="ltr"><ResponsiveContainer width="100%" height="100%">{chart==="Bar chart"?<BarChart data={chartData}><XAxis dataKey="label"/><YAxis/><Tooltip/><Bar dataKey="value" fill="#6366f1"/></BarChart>:chart==="Line chart"?<LineChart data={chartData}><XAxis dataKey="label"/><YAxis/><Tooltip/><Line dataKey="value" stroke="#6366f1"/></LineChart>:<PieChart><Tooltip/><Pie data={chartData.filter(d=>d.value>0)} dataKey="value" nameKey="label" fill="#6366f1" label/></PieChart>}</ResponsiveContainer></div>}
      <div className="overflow-auto"><table className="w-full text-sm"><thead><tr>{result.columns.map((col,i)=><th key={i} className="border-b border-border p-2 text-start"><button onClick={()=>{setSort(s=>({column:i,desc:s?.column===i?!s.desc:false}));setPage(0);}}>{col}{sort?.column===i?(sort.desc?" ↓":" ↑"):""}</button></th>)}</tr></thead><tbody>{sortedRows.slice(page*50,page*50+50).map((row,i)=><tr key={i}>{row.map((value,j)=><td key={j} className="max-w-md whitespace-pre-wrap break-all border-b border-border p-2 font-mono text-xs">{cell(value)}</td>)}</tr>)}</tbody></table></div>
      <div className="flex items-center gap-3"><button className={button} disabled={page===0} onClick={()=>setPage(p=>p-1)}>{t("Previous")}</button><span>{page+1} / {Math.max(1,Math.ceil(result.rows.length/50))}</span><button className={button} disabled={(page+1)*50>=result.rows.length} onClick={()=>setPage(p=>p+1)}>{t("Next")}</button></div>
    </section>}
  </main>;
}
