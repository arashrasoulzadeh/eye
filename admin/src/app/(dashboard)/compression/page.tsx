"use client";
import { useState } from "react";
import { useTenant } from "@/components/TenantContext";
import { api } from "@/lib/api";

type Storage = { database_bytes: number; reusable_bytes: number; size_scope: string; records: number; payload_bytes: number; pruned_payload_bytes: number; scan_ms: number };
type Result = { storage: Storage; count: number; sample: number; bytes: number; pruned: number; groups: number; median: number; p95: number };
const bytes = (n: number) => new Intl.NumberFormat("en", { maximumFractionDigits: 2 }).format(n / 1024 / 1024) + " MiB";
export default function CompressionPage() {
  const { tenantId } = useTenant();
  const [event, setEvent] = useState("");
  const [days, setDays] = useState(30);
  const [resolution, setResolution] = useState(3600000);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [result, setResult] = useState<Result | null>(null);
  async function assess() {
    if (!tenantId || busy || days < 1) return;
    setBusy(true); setError(""); setResult(null);
    try {
      const filters = { tenant_id: tenantId, event_name: event || undefined, to_ts: Date.now() - days * 86400000, limit: 100 };
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
      setResult({storage, count: page.total, sample: rows.length, bytes: size, pruned, groups, median: times[2], p95: times[4]});
    } catch(e) { setError(e instanceof Error ? e.message : String(e)); }
    finally { setBusy(false); }
  }
  return <div className="space-y-6">
    <h1 className="text-2xl font-semibold">Compression &amp; Storage</h1>
    <p className="text-fg-muted">Read-only assessment. Select a tenant before measuring. No events are modified or deleted.</p>
    <div className="grid gap-4 rounded-lg border border-border p-4 md:grid-cols-3">
      <label>Event name<input className="block w-full rounded border border-border bg-surface p-2" value={event} disabled={busy} onChange={e=>{setEvent(e.target.value);setResult(null);}} placeholder="All event types"/></label>
      <label>Older than (days)<input className="block w-full rounded border border-border bg-surface p-2" type="number" min={1} max={36500} value={days} disabled={busy} onChange={e=>{setDays(Number(e.target.value));setResult(null);}}/></label>
      <label>Aggregation resolution<select className="block w-full rounded border border-border bg-surface p-2" value={resolution} disabled={busy} onChange={e=>{setResolution(Number(e.target.value));setResult(null);}}><option value={60000}>Minute</option><option value={3600000}>Hour</option><option value={86400000}>Day</option></select></label>
      <button className="rounded bg-accent p-2 text-accent-fg disabled:opacity-50" disabled={!tenantId || busy || !Number.isFinite(days) || days < 1 || days > 36500} onClick={assess}>{busy ? "Measuring…" : "Run assessment"}</button>
    </div>
    {error && <p role="alert">{error}</p>}
    {result && <div className="space-y-4 rounded-lg border border-border p-4">
      <h2 className="font-semibold">Assessment results</h2>
      <div className="grid gap-3 md:grid-cols-2">
      <p>Database allocation: {bytes(result.storage.database_bytes)} ({result.storage.size_scope})</p>
      <p>Reported reusable space: {bytes(result.storage.reusable_bytes)} (SQLite only)</p>
      <p>Matching records at assessment: {result.storage.records.toLocaleString()}</p>
      <p>Full matching payload size: {bytes(result.storage.payload_bytes)}</p>
      <p>Projected payload size after pruning: {bytes(result.storage.pruned_payload_bytes)}</p>
      <p>Projected logical savings: {bytes(result.storage.payload_bytes-result.storage.pruned_payload_bytes)}</p>
      <p>Server assessment scan: {result.storage.scan_ms.toFixed(2)} ms</p>
      </div>
      <p>{result.count.toLocaleString()} matching records; {result.sample} sampled.</p>
      <p>Sample serialized size: {bytes(result.bytes)}. Estimated matching serialized size: {bytes(result.sample ? result.bytes / result.sample * result.count : 0)}.</p>
      <p>Payload removal: sample before {bytes(result.bytes)}, after {bytes(result.pruned)}; savings {result.bytes ? ((1-result.pruned/result.bytes)*100).toFixed(1) : "0"}%.</p>
      <p>Aggregation: {result.sample} sampled records into {result.groups} event/time groups. Full-range size and speed are not yet measured.</p>
      <p>Deletion: {result.count.toLocaleString()} matching records would be removed; historical analytics may be lost.</p>
      <p>Baseline API latency (5 requests): median {result.median.toFixed(1)} ms; p95 {result.p95.toFixed(1)} ms.</p>
      <p className="text-fg-muted">Serialized JSON is not physical database size. Sample is newest-first, not representative by design. Cached API timings include network overhead. Post-operation latency and disk savings require server-side execution and measurement.</p>
    </div>}
  </div>;
}
