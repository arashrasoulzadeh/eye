"use client";
import { useEffect, useState } from "react";
import { api, Tenant, ApiError } from "@/lib/api";
import { Card, CardHeader } from "@/components/Card";
import { Button } from "@/components/Button";
import { Combobox } from "@/components/Combobox";

export default function ReplayPage() {
  const [tenants, setTenants] = useState<Tenant[]>([]), [tenant, setTenant] = useState(""), [from, setFrom] = useState(""), [to, setTo] = useState(""), [result, setResult] = useState(""), [loading, setLoading] = useState(false), [error, setError] = useState("");
  useEffect(() => { api.tenants.list().then((x) => { setTenants(x); if (x[0]) setTenant(x[0].ID); }).catch((e) => setError(e.message)); }, []);
  async function replay() { if (!tenant || !from || !to) return; setLoading(true); setError(""); try { const r = await api.operations.replay(tenant, new Date(from).getTime(), new Date(to).getTime()); setResult(JSON.stringify(r)); } catch (e) { setError(e instanceof ApiError ? e.message : "Replay failed"); } finally { setLoading(false); } }
  return <div><h1 className="text-xl font-semibold text-fg">Event Replay</h1><p className="mt-1 text-sm text-fg-muted">Re-ingest a tenant’s historical events into the configured pipeline.</p><Card className="mt-6"><CardHeader title="Replay range"/><div className="flex flex-wrap items-end gap-3 p-5"><label className="text-sm text-fg-muted">Tenant<Combobox value={tenant} onChange={setTenant} options={tenants.map((t) => ({ value: t.ID, label: t.Name }))} placeholder="Select tenant"/></label><label className="text-sm text-fg-muted">From<input type="datetime-local" value={from} onChange={(e) => setFrom(e.target.value)} className="mt-1 block rounded border border-border bg-surface px-3 py-2 text-fg"/></label><label className="text-sm text-fg-muted">To<input type="datetime-local" value={to} onChange={(e) => setTo(e.target.value)} className="mt-1 block rounded border border-border bg-surface px-3 py-2 text-fg"/></label><Button onClick={() => void replay()} loading={loading} disabled={!tenant || !from || !to}>Replay events</Button></div></Card>{error && <p className="mt-4 text-sm text-danger">{error}</p>}{result && <Card className="mt-6"><CardHeader title="Replay result"/><pre className="p-5 text-sm text-fg-muted">{result}</pre></Card>}</div>;
}
