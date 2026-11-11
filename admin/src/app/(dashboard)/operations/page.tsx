"use client";
import { useCallback, useEffect, useState } from "react";
import { api, Tenant, ApiError } from "@/lib/api";
import { Card, CardHeader } from "@/components/Card";
import { Button } from "@/components/Button";
import { Combobox } from "@/components/Combobox";

function JsonRows({ values }: { values: unknown[] }) {
  return <div className="divide-y divide-border">{values.length === 0 ? <p className="p-4 text-sm text-fg-muted">No records.</p> : values.slice(0, 50).map((value, index) => <pre key={index} className="overflow-auto px-4 py-3 text-xs text-fg-muted">{JSON.stringify(value, null, 2)}</pre>)}</div>;
}

export default function OperationsPage() {
  const [tenants, setTenants] = useState<Tenant[]>([]);
  const [tenantId, setTenantId] = useState("");
  const [flags, setFlags] = useState<unknown[]>([]), [aggregations, setAggregations] = useState<unknown[]>([]), [anomalies, setAnomalies] = useState<unknown[]>([]), [audit, setAudit] = useState<unknown[]>([]), [deadLetters, setDeadLetters] = useState<unknown[]>([]);
  const [flagKey, setFlagKey] = useState(""), [loading, setLoading] = useState(false), [error, setError] = useState("");
  const load = useCallback(async () => {
    if (!tenantId) return;
    setLoading(true); setError("");
    try {
      const [f, a, n, l, d] = await Promise.all([api.operations.flags(tenantId), api.operations.aggregations(tenantId), api.operations.anomalies(), api.operations.auditLog(), api.operations.deadLetters()]);
      setFlags(f); setAggregations(a); setAnomalies(n); setAudit(l); setDeadLetters(d);
    } catch (e) { setError(e instanceof ApiError ? e.message : "Failed to load operations"); } finally { setLoading(false); }
  }, [tenantId]);
  useEffect(() => { api.tenants.list().then((list) => { setTenants(list); if (list[0]) setTenantId(list[0].ID); }).catch((e) => setError(e.message)); }, []);
  useEffect(() => { void load(); }, [load]);
  async function setFlag() { if (!tenantId || !flagKey.trim()) return; await api.operations.setFlag(tenantId, flagKey.trim(), true); setFlagKey(""); await load(); }
  async function refreshAggregations() { await api.operations.refreshAggregations(tenantId); await load(); }
  return <div><div className="flex flex-wrap items-end justify-between gap-4"><div><h1 className="text-xl font-semibold text-fg">Operations</h1><p className="mt-1 text-sm text-fg-muted">Feature flags, aggregations, anomalies, audit history, dead letters, and replay controls.</p></div><div className="flex gap-2"><Combobox value={tenantId} onChange={setTenantId} options={tenants.map((t) => ({ value: t.ID, label: t.Name }))} placeholder="Select tenant"/><Button variant="secondary" onClick={() => void load()} loading={loading}>Refresh</Button></div></div>{error && <p className="mt-4 text-sm text-danger">{error}</p>}<div className="mt-6 grid gap-6 lg:grid-cols-2"><Card><CardHeader title={`Feature flags (${flags.length})`} /><div className="flex gap-2 p-4"><input value={flagKey} onChange={(e) => setFlagKey(e.target.value)} placeholder="flag.key" className="min-w-0 flex-1 rounded border border-border bg-surface px-3 py-2 text-sm"/><Button onClick={() => void setFlag()}>Enable</Button></div><JsonRows values={flags}/></Card><Card><CardHeader title={`Aggregations (${aggregations.length})`} action={<Button variant="secondary" onClick={() => void refreshAggregations()} disabled={!tenantId}>Refresh materialized</Button>}/><JsonRows values={aggregations}/></Card><Card><CardHeader title={`Anomalies (${anomalies.length})`}/><JsonRows values={anomalies}/></Card><Card><CardHeader title={`Webhook dead letters (${deadLetters.length})`}/><JsonRows values={deadLetters}/></Card><Card className="lg:col-span-2"><CardHeader title={`Audit log (${audit.length})`}/><JsonRows values={audit}/></Card></div></div>;
}
