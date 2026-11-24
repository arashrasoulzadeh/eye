"use client";
import { useState } from "react";
import { useTenant } from "@/components/TenantContext";
import { api } from "@/lib/api";

type Storage = { database_bytes: number; reusable_bytes: number; size_scope: string; records: number; payload_bytes: number; pruned_payload_bytes: number; scan_ms: number; count_median_ms: number; count_p95_ms: number; benchmark_samples: number };
type Result = { tenant: string; cutoff: number; storage: Storage; count: number; sample: number; bytes: number; pruned: number; groups: number; median: number; p95: number };
const bytes = (n: number) => new Intl.NumberFormat("en", { maximumFractionDigits: 2 }).format(n / 1024 / 1024) + " MiB";
export default function CompressionPage() {
  const { tenantId } = useTenant();
  const [event, setEvent] = useState("");
  const [days, setDays] = useState(30);
  const [resolution, setResolution] = useState(3600000);
  const [confirmation, setConfirmation] = useState("");
  const [operation, setOperation] = useState("");
  const [comparison, setComparison] = useState<{before: Storage; after: Storage} | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [result, setResult] = useState<Result | null>(null);
  async function assess(fixedCutoff?: number) {
    if (!tenantId || busy || days < 1) return;
    setBusy(true); setError(""); setResult(null);
    try {
      const filters = { tenant_id: tenantId, event_name: event || undefined, to_ts: fixedCutoff ?? Date.now() - days * 86400000, limit: 100 };
      const response = await fetch('/api/proxy/compression/assessment?' + new URLSearchParams({tenant_id: tenantId, event_name: event, to_ts: String(filters.to_ts)}), {cache: 'no-store'});
      if (!response.ok) throw new Error(await response.text());
      const storage: Storage = await response.json();
      const times: number[] = [];
      let page = await api.events.listPage(filters);
      for (let i = 0; i < 5; i++) {
        const start = performance.now();
        page = await api.events.listPage(filters);
        times.push(performance.now() - start);
      }
      times.sort((a,b) => a-b);
      const encoder = new TextEncoder();
      const rows = page.items;
      const size = rows.reduce((n,r) => n + encoder.encode(JSON.stringify(r)).length, 0);
      const pruned = rows.reduce((n,r) => n + encoder.encode(JSON.stringify({...r, Payload: "{}"})).length, 0);
      const groups = new Set(rows.map(r => JSON.stringify([r.EventName, Math.floor(r.ClientTS / resolution)]))).size;
      setResult({tenant: tenantId, cutoff: filters.to_ts, storage, count: page.total, sample: rows.length, bytes: size, pruned, groups, median: times[2], p95: times[4]});
    } catch(e) { setError(e instanceof Error ? e.message : String(e)); }
    finally { setBusy(false); }
  }
  async function prune(action: "prune" | "compact" = "prune") {
    const expected = action.toUpperCase()+" "+tenantId+" "+event;
    if (!result || result.tenant !== tenantId || !event || busy || confirmation !== expected) return;
    setBusy(true); setError(""); setOperation(""); setComparison(null);
    try {
      const res = await fetch("/api/proxy/compression/prune",{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({tenant_id:tenantId,event_name:event,to_ts:result.cutoff,confirm:confirmation,action})});
      if (!res.ok) throw new Error(await res.text());
      const data = await res.json();
      setOperation(action === "compact" ? "Compacted "+data.updated+" records into historical counts. Individual event details are permanently removed." : "Updated "+data.updated+" records; actual logical payload bytes "+data.before_payload_bytes+" → "+data.after_payload_bytes+"; operation "+data.duration_ms.toFixed(1)+" ms. Run assessment again to measure database allocation.");
      setResult(null); setConfirmation("");
      const measured = await fetch("/api/proxy/compression/assessment?" + new URLSearchParams({tenant_id:tenantId,event_name:event,to_ts:String(result.cutoff)}),{cache:"no-store"});
      if (!measured.ok) throw new Error("Operation committed; post-operation assessment failed: "+await measured.text());
      const after: Storage = await measured.json();
      setComparison({before:result.storage,after});
    } catch(e) {setError(e instanceof Error?e.message:String(e));}
    finally {setBusy(false);}
  }
  return <div className="space-y-6">
    <h1 className="text-2xl font-semibold">Compression &amp; Storage</h1>
    <p className="text-fg-muted">Assessment is read-only. Owner-authorized payload pruning permanently removes event payloads.</p>
    <div className="grid gap-4 rounded-lg border border-border p-4 md:grid-cols-3">
      <label>Event name<input className="block w-full rounded border border-border bg-surface p-2" value={event} disabled={busy} onChange={e=>{setEvent(e.target.value);setResult(null);}} placeholder="All event types"/></label>
      <label>Older than (days)<input className="block w-full rounded border border-border bg-surface p-2" type="number" min={1} max={36500} value={days} disabled={busy} onChange={e=>{setDays(Number(e.target.value));setResult(null);}}/></label>
      <label>Aggregation resolution<select className="block w-full rounded border border-border bg-surface p-2" value={resolution} disabled={busy} onChange={e=>{setResolution(Number(e.target.value));setResult(null);}}><option value={60000}>Minute</option><option value={3600000}>Hour</option><option value={86400000}>Day</option></select></label>
      <button className="rounded bg-accent p-2 text-accent-fg disabled:opacity-50" disabled={!tenantId || busy || !Number.isFinite(days) || days < 1 || days > 36500} onClick={()=>assess()}>{busy ? "Measuring…" : "Run assessment"}</button>
    </div>
    {operation && <p role="status">{operation}</p>}
    {comparison && <div className="rounded border border-border p-4 space-y-2">
      <h2 className="font-semibold">Measured before / after</h2>
      <p>Database allocation: {bytes(comparison.before.database_bytes)} → {bytes(comparison.after.database_bytes)}</p>
      <p>Reusable space: {bytes(comparison.before.reusable_bytes)} → {bytes(comparison.after.reusable_bytes)}</p>
      <p>Matching payload: {bytes(comparison.before.payload_bytes)} → {bytes(comparison.after.payload_bytes)}</p>
      <p>Raw records: {comparison.before.records} → {comparison.after.records}</p>
      <p>Count median: {comparison.before.count_median_ms.toFixed(2)} → {comparison.after.count_median_ms.toFixed(2)} ms</p>
      <p>Count p95: {comparison.before.count_p95_ms.toFixed(2)} → {comparison.after.count_p95_ms.toFixed(2)} ms</p>
      <p>Database allocation is global; concurrent activity can affect these measurements. Raw count latency does not measure historical rollup accuracy.</p>
    </div>}
    {error && <p role="alert">{error}</p>}
    {result && result.tenant === tenantId && <div className="space-y-4 rounded-lg border border-border p-4">
      <h2 className="font-semibold">Assessment results</h2>
      <p>Fixed cutoff: {new Date(result.cutoff).toISOString()}</p>
      <div className="grid gap-3 md:grid-cols-2">
      <p>Database allocation: {bytes(result.storage.database_bytes)} ({result.storage.size_scope})</p>
      <p>Reported reusable space: {bytes(result.storage.reusable_bytes)} (SQLite only)</p>
      <p>Matching records at assessment: {result.storage.records.toLocaleString()}</p>
      <p>Full matching payload size: {bytes(result.storage.payload_bytes)}</p>
      <p>Projected payload size after pruning: {bytes(result.storage.pruned_payload_bytes)}</p>
      <p>Projected logical savings: {bytes(result.storage.payload_bytes-result.storage.pruned_payload_bytes)}</p>
      <p>Server assessment scan: {result.storage.scan_ms.toFixed(2)} ms</p>
      <p>Database count median: {result.storage.count_median_ms.toFixed(2)} ms</p>
      <p>Database count p95: {result.storage.count_p95_ms.toFixed(2)} ms ({result.storage.benchmark_samples} samples)</p>
      </div>
      <p>{result.count.toLocaleString()} matching records; {result.sample} sampled.</p>
      <p>Sample serialized size: {bytes(result.bytes)}. Estimated matching serialized size: {bytes(result.sample ? result.bytes / result.sample * result.count : 0)}.</p>
      <p>Payload removal: sample before {bytes(result.bytes)}, after {bytes(result.pruned)}; savings {result.bytes ? ((1-result.pruned/result.bytes)*100).toFixed(1) : "0"}%.</p>
      <p>Aggregation: {result.sample} sampled records into {result.groups} event/time groups. Full-range size and speed are not yet measured.</p>
      <p>Deletion: {result.count.toLocaleString()} matching records would be removed; historical analytics may be lost.</p>
      <p>Baseline API latency (5 requests): median {result.median.toFixed(1)} ms; p95 {result.p95.toFixed(1)} ms.</p>
      {event && <div className="space-y-2 border border-border rounded p-3">
      <h3 className="font-semibold">Prune payloads (owner only)</h3>
      <p>Permanently replace payloads with empty JSON for up to 500 matching records. Payload-based analytics will change. Back up first.</p>
      <label>Type PRUNE {tenantId} {event}<input className="block w-full border rounded p-2" value={confirmation} disabled={busy} onChange={e=>setConfirmation(e.target.value)}/></label>
      <button disabled={busy || confirmation !== "PRUNE "+tenantId+" "+event} onClick={()=>prune()} className="border rounded p-2 disabled:opacity-50">Prune up to 500 payloads</button>
      <p>Historical compaction permanently removes raw records, preserving full bucket counts only. Partial windows, funnels and session analytics may change.</p>
      <label>For compaction type COMPACT {tenantId} {event}</label>
      <button disabled={busy || confirmation !== "COMPACT "+tenantId+" "+event} onClick={()=>prune("compact")} className="border rounded p-2 disabled:opacity-50">Compact up to 500 events</button>
      </div>}
      <p className="text-fg-muted">Serialized JSON is not physical database size. Sample is newest-first, not representative by design. Cached API timings include network overhead. Post-operation latency and disk savings require server-side execution and measurement.</p>
    </div>}
  </div>;
}
