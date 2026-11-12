"use client";
import { useCallback, useEffect, useState } from "react";
import { api, Tenant, ApiError } from "@/lib/api";
import { Card, CardHeader } from "@/components/Card";
import { Button } from "@/components/Button";
import { Combobox } from "@/components/Combobox";
import { useTenant } from "@/components/TenantContext";

function JsonRows({ values }: { values: unknown[] }) {
  return <div className="divide-y divide-border">{values.length === 0 ? <p className="p-4 text-sm text-fg-muted">No recent records match the current scope.</p> : values.map((value, index) => { const record = (value && typeof value === "object" ? value : { value }) as Record<string, unknown>; const serialized = JSON.stringify(record, null, 2); return <div key={recordKey(record, index)} className="flex items-start gap-3 px-4 py-3"><div className="grid min-w-0 flex-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">{Object.entries(record).map(([key, item]) => <div key={key} className="min-w-0"><p className="text-[11px] font-medium uppercase tracking-wide text-fg-muted">{key.replaceAll("_", " ")}</p><p className="truncate text-sm text-fg" title={String(item)}>{typeof item === "object" ? JSON.stringify(item) : String(item ?? "—")}</p></div>)}</div><Button variant="secondary" onClick={() => void navigator.clipboard?.writeText(serialized)}>Copy</Button></div>; })}</div>;
}
function recordKey(record: Record<string, unknown>, index: number) { const id = record.ID ?? record.id ?? record.Key ?? record.key ?? record.Timestamp ?? record.timestamp; return id == null ? `record-${index}-${JSON.stringify(record)}` : String(id); }

export default function OperationsPage() {
  const { tenantId: globalTenant } = useTenant();
  const [tenants, setTenants] = useState<Tenant[]>([]);
  const [tenantId, setTenantId] = useState("");
  useEffect(() => { if (globalTenant && globalTenant !== tenantId) setTenantId(globalTenant); }, [globalTenant, tenantId]);
  const [flags, setFlags] = useState<unknown[]>([]), [aggregations, setAggregations] = useState<unknown[]>([]), [anomalies, setAnomalies] = useState<unknown[]>([]), [audit, setAudit] = useState<unknown[]>([]), [deadLetters, setDeadLetters] = useState<unknown[]>([]);
  const [flagKey, setFlagKey] = useState(""), [loading, setLoading] = useState(false), [busy, setBusy] = useState(""), [error, setError] = useState("");
  const load = useCallback(async () => {
    if (!tenantId) return;
    setLoading(true); setError("");
    try {
      const [f, a, n, l, d] = await Promise.all([api.operations.flagsPage(tenantId, { limit: 5 }), api.operations.aggregationsPage(tenantId, { limit: 5 }), api.operations.anomaliesPage({ limit: 5 }), api.operations.auditLogPage({ limit: 5 }), api.operations.deadLetters({ limit: 5 })]);
      setFlags(f.items); setAggregations(a.items); setAnomalies(n.items); setAudit(l.items); setDeadLetters(d.items);
    } catch (e) { setError(e instanceof ApiError ? e.message : "Failed to load operations"); } finally { setLoading(false); }
  }, [tenantId]);
  useEffect(() => { api.tenants.list().then((list) => { setTenants(list); if (list[0]) setTenantId(list[0].ID); }).catch((e) => setError(e instanceof ApiError ? e.message : "Failed to load tenants")); }, []);
  useEffect(() => { void load(); }, [load]);
  async function setFlag() { if (!tenantId || !flagKey.trim()) return; setBusy("flag"); setError(""); try { await api.operations.setFlag(tenantId, flagKey.trim(), true); setFlagKey(""); await load(); } catch (e) { setError(e instanceof ApiError ? e.message : "Failed to enable feature flag"); } finally { setBusy(""); } }
  async function refreshAggregations() { setBusy("aggregations"); setError(""); try { await api.operations.refreshAggregations(tenantId); await load(); } catch (e) { setError(e instanceof ApiError ? e.message : "Failed to refresh aggregations"); } finally { setBusy(""); } }
  return <div><div className="flex flex-wrap items-end justify-between gap-4"><div><h1 className="text-xl font-semibold text-fg">Operations</h1><p className="mt-1 text-sm text-fg-muted">Quick operational snapshots. Each panel loads only the five newest API records; open the resource page for full CRUD and pagination.</p></div><div className="flex gap-2"><Combobox value={tenantId} onChange={setTenantId} options={tenants.map((t) => ({ value: t.ID, label: t.Name }))} placeholder="Select tenant"/><Button variant="secondary" onClick={() => void load()} loading={loading}>Refresh</Button></div></div>{error && <p className="mt-4 rounded border border-danger/30 bg-danger/5 p-3 text-sm text-danger">{error}</p>}<div className="mt-6 grid gap-6 lg:grid-cols-2"><Card><CardHeader title={`Feature flags (latest ${flags.length})`} /><div className="flex gap-2 p-4"><input value={flagKey} onChange={(e) => setFlagKey(e.target.value)} placeholder="flag.key" className="min-w-0 flex-1 rounded border border-border bg-surface px-3 py-2 text-sm"/><Button onClick={() => void setFlag()} loading={busy === "flag"}>Enable</Button></div><JsonRows values={flags}/></Card><Card><CardHeader title={`Aggregations (latest ${aggregations.length})`} action={<Button variant="secondary" onClick={() => void refreshAggregations()} loading={busy === "aggregations"} disabled={!tenantId}>Refresh materialized</Button>}/><JsonRows values={aggregations}/></Card><Card><CardHeader title={`Anomalies (latest ${anomalies.length})`}/><JsonRows values={anomalies}/></Card><Card><CardHeader title={`Webhook dead letters (latest ${deadLetters.length})`}/><JsonRows values={deadLetters}/></Card><Card className="lg:col-span-2"><CardHeader title={`Audit log (latest ${audit.length})`}/><JsonRows values={audit}/></Card></div></div>;
}
