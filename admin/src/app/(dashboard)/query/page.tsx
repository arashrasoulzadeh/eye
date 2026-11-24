"use client";

import { useState, useEffect, useRef } from "react";
import { ResponsiveContainer, BarChart, Bar, LineChart, Line, PieChart, Pie, XAxis, YAxis, Tooltip } from "recharts";
import { useTenant } from "@/components/TenantContext";
import { useTimeRange } from "@/components/TimeRangeContext";
import { translate, useLanguage } from "@/components/LanguageContext";

type Result = { columns: string[]; rows: unknown[][]; truncated: boolean; duration_ms: number };
const examples = [
  { name: "Answer streak", sql: "SELECT * FROM events WHERE events.type = 'answer' AND event.streak > 1 ORDER BY timestamp DESC LIMIT 100;" },
  { name: "Event counts", sql: "SELECT type, COUNT(*) AS total FROM events GROUP BY type ORDER BY total DESC;" },
];
const fields = ["event_id","tenant_id","session_id","user_id","type","timestamp","ingest_ts","payload","dims","source","is_aggregated","aggregation_id","experiment_id","variant"];
const button = "rounded-lg border border-border px-2 py-1 text-xs hover:bg-surface-2 disabled:opacity-50";
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
  const [schemaOpen, setSchemaOpen] = useState(false);
  const [allFields, setAllFields] = useState(false);
  const [inspectedCell, setInspectedCell] = useState<string | null>(null);
  const [inspected, setInspected] = useState<string | null>(null);
  const tableNames = [...new Set(Array.from(sql.replace(/'(?:[^']|'')*'/g, " ").matchAll(/\b(?:FROM|JOIN)\s+([a-z_][a-z_0-9]*)/gi), m => m[1].toLowerCase()))];
  const schemaFields = [...new Set([...fields,...properties])];
  const smartExamples = [...examples, {name:"Active users",sql:"SELECT COUNT(DISTINCT user_id) AS total FROM events;"},{name:"Busiest sessions",sql:"SELECT session_id, COUNT(*) AS total FROM events GROUP BY session_id ORDER BY total DESC LIMIT 10;"},{name:"Latest events",sql:"SELECT type, timestamp, user_id FROM events ORDER BY timestamp DESC LIMIT 20;"}, ...properties.slice(0,3).map(p=>({name:p,sql:`SELECT ${p}, COUNT(*) AS total FROM events WHERE ${p} IS NOT NULL GROUP BY ${p} ORDER BY total DESC LIMIT 10;`}))];
  const [history, setHistory] = useState<string[]>([]);
  const key = "eye.query." + tenantId;
  useEffect(() => {
    try {
      const stored = JSON.parse(localStorage.getItem(key) || "{}");
      setHistory(Array.isArray(stored.history) ? stored.history : []);
    } catch { setHistory([]); }
    setResult(null);
    setProperties([]);
    if (!tenantId) return;
    const controller = new AbortController();
    fetch("/api/proxy/query-explorer?tenant_id="+encodeURIComponent(tenantId),{signal:controller.signal})
      .then(r=>r.ok?r.json():[]).then(data=>{if(!controller.signal.aborted&&Array.isArray(data))setProperties(data);}).catch(()=>undefined);
    return ()=>controller.abort();
  }, [key, tenantId]);
  function persist(h: string[]) {
    setHistory(h);
    try { const old = JSON.parse(localStorage.getItem(key) || "{}"); localStorage.setItem(key, JSON.stringify({...old,history:h})); }
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
      persist([sql,...history.filter(item=>item!==sql)].slice(0,30));
    } catch (e) { setError(e instanceof Error ? e.message : "Query failed"); }
    finally { setBusy(false); }
  }
  return <main className="space-y-3 p-4">
    <header><h1 className="text-xl font-semibold">{t("Query Explorer")}</h1><p className="text-sm text-fg-muted">{t("Read-only event SQL")}</p></header>
    {!tenantId && <p role="status" className="rounded-lg border border-border p-3">{t("Select a tenant before running a query.")}</p>}
    <div className="grid gap-3 lg:grid-cols-[minmax(0,1fr)_14rem]">
      <section className="min-w-0 space-y-2 rounded-lg border border-border bg-surface p-3">
        <div className="relative rounded-lg border border-border bg-surface-2" dir="ltr"><pre ref={highlight} aria-hidden="true" className="pointer-events-none absolute inset-0 m-0 overflow-hidden whitespace-pre-wrap break-words p-4 font-mono text-sm leading-6">{sql.split(/('(?:[^']|'')*'|\b(?:SELECT|FROM|WHERE|AND|OR|JOIN|LEFT|INNER|ON|AS|GROUP|BY|ORDER|HAVING|LIMIT|OFFSET|IN|EXISTS|NOT|NULL|DISTINCT|COUNT|SUM|AVG|MIN|MAX)\b|\b\d+(?:\.\d+)?\b)/gi).map((token,i)=><span key={i} className={token.startsWith("'")?"text-emerald-600 dark:text-emerald-400":i%2===1?"text-violet-600 dark:text-violet-400":"text-fg"}>{token}</span>)}{"\n"}</pre><textarea ref={editor} onScroll={e=>{if(highlight.current){highlight.current.scrollTop=e.currentTarget.scrollTop;highlight.current.scrollLeft=e.currentTarget.scrollLeft;}}} onSelect={e=>setCursor(e.currentTarget.selectionStart)} aria-label={t("Query Explorer")} dir="ltr" spellCheck={false} rows={5} value={sql} onChange={e=>{setSql(e.target.value);setCursor(e.target.selectionStart);}} onKeyDown={e=>{if((e.ctrlKey||e.metaKey)&&e.key==="Enter"){e.preventDefault();void run();}}} className="relative block w-full resize-y bg-transparent p-4 font-mono text-sm leading-6 text-transparent caret-current outline-none focus:ring-2 focus:ring-accent/30" style={{caretColor:"var(--color-fg, currentColor)"}} /></div>
        <div className="flex flex-wrap gap-1">{suggestions.map(word=><button key={word} className={button+" font-mono"} onClick={()=>insert(word)}>{word}</button>)}</div>
        <div className="flex flex-wrap items-center gap-2">
          <button title={t("Run selected tenant query (Ctrl/⌘ + Enter)")} className={button+" bg-accent text-accent-fg"} disabled={busy||loading||!tenantId||!sql.trim()||(scopedTime&&!to)} onClick={()=>void run()}>{t(busy?"Running query…":"Run query")}</button>
          <label title={t("Restrict events to the dashboard time range")} className="text-xs"><input type="checkbox" checked={scopedTime} onChange={e=>setScopedTime(e.target.checked)} /> {t("Use the global time range")}</label>
        </div>
        {tableNames.length>0 && <div className="flex flex-wrap items-center gap-1 text-xs"><span>{t("Referenced tables")}:</span>{tableNames.map(table=><button key={table} title={t("Inspect table schema")} className={button+" font-mono"} onClick={()=>{setInspected(table);setSchemaOpen(true);setAllFields(false);}}>{table}</button>)}</div>}
        {error && <p role="alert" className="whitespace-pre-wrap text-sm text-danger">{t("Query failed")}: {t(error)}</p>}
      </section>
      <aside className="min-w-0 space-y-2 rounded-lg border border-border bg-surface p-3">
        <button className="flex w-full justify-between text-sm font-medium" title={t("Inspect table schema")} aria-expanded={schemaOpen} onClick={()=>{setInspected("events");setSchemaOpen(!schemaOpen);}}>{t("Schema explorer")} <span>{schemaOpen?"−":"+"}</span></button>
        {schemaOpen && <div className="space-y-1 text-xs"><code>{inspected || "events"}</code>{inspected && inspected!=="events"?<p>{t("Only events is available")}</p>:<><div className="max-h-48 overflow-auto">{schemaFields.slice(0,allFields?undefined:10).map(field=><button key={field} title={t("Insert field into query")} className="block w-full truncate text-start font-mono hover:bg-surface-2" onClick={()=>insert(field)}>{field}</button>)}</div>{schemaFields.length>10&&<button className={button} onClick={()=>setAllFields(!allFields)}>{t(allFields?"Show less":"Show all fields")}</button>}</>}</div>}
        <details open><summary className="cursor-pointer text-sm font-medium">{t("Smart query examples")}</summary><div className="mt-1 space-y-1">{smartExamples.map(e=><button key={e.name} title={e.sql} className="block w-full truncate rounded px-1 py-1 text-start text-xs hover:bg-surface-2" onClick={()=>{setSql(e.sql);setCursor(e.sql.length);}}>{t(e.name)}</button>)}</div></details>
        <details><summary className="cursor-pointer text-sm font-medium">{t("Query history")}</summary>{history.slice(0,10).map(h=><button key={h} dir="ltr" title={h} className="block w-full truncate text-left font-mono text-xs" onClick={()=>{setSql(h);setCursor(h.length);}}>{h}</button>)}</details>
      </aside>
    </div>
    {result && <section className="space-y-3 rounded-xl border border-border bg-surface p-4">
      <div className="flex flex-wrap items-center gap-3"><h2 className="font-semibold">{t("Results")}</h2><span className="text-sm">{t("Rows")}: {result.rows.length} · {t("Execution time (ms)")}: {result.duration_ms}</span>
        <button title={t("Download all returned rows as CSV")} className={button} onClick={()=>download("query.csv",[result.columns.map(csv).join(","),...result.rows.map(row=>row.map(csv).join(","))].join("\r\n"),"text/csv")}>{t("Export CSV")}</button>
        <button title={t("Download all returned rows as JSON")} className={button} onClick={()=>download("query.json",JSON.stringify(result,null,2),"application/json")}>{t("Export JSON")}</button>
      </div>
      {result.truncated && <p>{t("Results limited to 1000 rows.")}</p>}
      {!result.rows.length && <p>{t("No results")}</p>}
      <div className="flex flex-wrap gap-2">
        <select aria-label={t("View")} className={button} value={chart} onChange={e=>setChart(e.target.value)}>{["Table","Bar chart","Line chart","Pie chart"].map(v=><option key={v} value={v}>{t(v)}</option>)}</select>
        {chart!=="Table" && <><select aria-label="X" className={button} value={xColumn} onChange={e=>setXColumn(Number(e.target.value))}>{result.columns.map((c,i)=><option key={i} value={i}>{c}</option>)}</select><select aria-label="Y" className={button} value={yColumn} onChange={e=>setYColumn(Number(e.target.value))}>{result.columns.map((c,i)=><option key={i} value={i}>{c}</option>)}</select></>}
      </div>
      {chart!=="Table" && <div className="h-80" dir="ltr"><ResponsiveContainer width="100%" height="100%">{chart==="Bar chart"?<BarChart data={chartData}><XAxis dataKey="label"/><YAxis/><Tooltip/><Bar dataKey="value" fill="#6366f1"/></BarChart>:chart==="Line chart"?<LineChart data={chartData}><XAxis dataKey="label"/><YAxis/><Tooltip/><Line dataKey="value" stroke="#6366f1"/></LineChart>:<PieChart><Tooltip/><Pie data={chartData.filter(d=>d.value>0)} dataKey="value" nameKey="label" fill="#6366f1" label/></PieChart>}</ResponsiveContainer></div>}
      {chart==="Table" && <div className="max-h-[55vh] overflow-auto"><table className="w-full text-xs"><thead className="sticky top-0 bg-surface"><tr>{result.columns.map((col,i)=><th key={i} className="border-b border-border px-2 py-1 text-start whitespace-nowrap"><button title={t("Sort by column")} onClick={()=>{setSort(s=>({column:i,desc:s?.column===i?!s.desc:false}));setPage(0);}}>{col}{sort?.column===i?(sort.desc?" ↓":" ↑"):""}</button></th>)}</tr></thead><tbody>{sortedRows.slice(page*50,page*50+50).map((row,i)=><tr key={i}>{row.map((value,j)=><td key={j} className="max-w-48 truncate border-b border-border px-2 py-1 font-mono text-xs"><button className="block max-w-full truncate text-start" title={cell(value)} onClick={()=>setInspectedCell(cell(value))}>{cell(value)}</button></td>)}</tr>)}</tbody></table></div>}
      <div className="flex items-center gap-3"><button className={button} disabled={page===0} onClick={()=>setPage(p=>p-1)}>{t("Previous")}</button><span>{page+1} / {Math.max(1,Math.ceil(result.rows.length/50))}</span><button className={button} disabled={(page+1)*50>=result.rows.length} onClick={()=>setPage(p=>p+1)}>{t("Next")}</button></div>
    </section>}
    {inspectedCell!==null && <section className="rounded-lg border border-border p-3"><button className={button} onClick={()=>setInspectedCell(null)}>{t("Close")}</button><pre className="max-h-64 overflow-auto whitespace-pre-wrap break-all text-xs" dir="ltr">{inspectedCell}</pre></section>}
  </main>;
}
