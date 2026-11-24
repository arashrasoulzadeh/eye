"use client";

import { useState, useEffect } from "react";
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
  }, [key]);
  function persist(h: string[], s: Saved[]) {
    setHistory(h); setSaved(s);
    try { localStorage.setItem(key, JSON.stringify({history:h,saved:s})); }
    catch { setError("Storage unavailable"); }
  }
  async function run() {
    if (!tenantId || busy || !sql.trim()) return;
    setBusy(true); setError(""); setResult(null); setPage(0);
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
        <textarea aria-label={t("Query Explorer")} dir="ltr" spellCheck={false} rows={10} value={sql} onChange={e=>setSql(e.target.value)} onKeyDown={e=>{if((e.ctrlKey||e.metaKey)&&e.key==="Enter"){e.preventDefault();void run();}}} className="w-full rounded-lg border border-border bg-surface-2 p-4 font-mono text-sm" />
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
        <div className="flex flex-wrap gap-1">{fields.map(field=><button key={field} className="rounded border border-border px-2 py-1 font-mono text-xs" onClick={()=>setSql(s=>s+" "+field)}>{field}</button>)}</div>
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
      <div className="overflow-auto"><table className="w-full text-sm"><thead><tr>{result.columns.map((col,i)=><th key={i} className="border-b border-border p-2 text-start">{col}</th>)}</tr></thead><tbody>{result.rows.slice(page*50,page*50+50).map((row,i)=><tr key={i}>{row.map((value,j)=><td key={j} className="max-w-md whitespace-pre-wrap break-all border-b border-border p-2 font-mono text-xs">{cell(value)}</td>)}</tr>)}</tbody></table></div>
      <div className="flex items-center gap-3"><button className={button} disabled={page===0} onClick={()=>setPage(p=>p-1)}>{t("Previous")}</button><span>{page+1} / {Math.max(1,Math.ceil(result.rows.length/50))}</span><button className={button} disabled={(page+1)*50>=result.rows.length} onClick={()=>setPage(p=>p+1)}>{t("Next")}</button></div>
    </section>}
  </main>;
}
