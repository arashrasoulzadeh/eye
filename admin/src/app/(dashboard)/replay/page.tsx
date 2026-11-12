"use client";
import { useEffect, useState } from "react";
import { api, Tenant, ApiError } from "@/lib/api";
import { Card, CardHeader } from "@/components/Card";
import { Button } from "@/components/Button";
import { Combobox } from "@/components/Combobox";
import { useTenant } from "@/components/TenantContext";
import { formatTimeRange, useTimeRange } from "@/components/TimeRangeContext";

export default function ReplayPage() {
  const { tenantId: globalTenant } = useTenant();
  const { from: globalFrom, to: globalTo, hours } = useTimeRange();
  const timeRangeLabel = formatTimeRange(hours);
  void timeRangeLabel;
  const [tenants, setTenants] = useState<Tenant[]>([]), [tenant, setTenant] = useState(""), [result, setResult] = useState(""), [loading, setLoading] = useState(false), [error, setError] = useState("");
  useEffect(() => { api.tenants.list().then((x) => { setTenants(x); if (x[0]) setTenant(x[0].ID); }).catch((e) => setError(e.message)); }, []);
  useEffect(() => { if (globalTenant && globalTenant !== tenant) setTenant(globalTenant); }, [globalTenant, tenant]);
  async function replay() { const start = globalFrom; const finish = globalTo; if (!tenant || !Number.isFinite(start) || !Number.isFinite(finish) || start >= finish) { setError("Choose a valid replay range"); return; } if (!window.confirm("Replay all events in this range? This may generate significant downstream traffic.")) return; setLoading(true); setError(""); try { const r = await api.operations.replay(tenant, start, finish); setResult(JSON.stringify(r, null, 2)); } catch (e) { setError(e instanceof ApiError ? e.message : "Replay failed"); } finally { setLoading(false); } }
  return <div><h1 className="text-xl font-semibold text-fg">Event Replay</h1><p className="mt-1 text-sm text-fg-muted">Re-ingest a tenant’s historical events into the configured pipeline. Replay is safe to retry because event deduplication remains active.</p><details className="mt-4 rounded-lg border border-border bg-surface-2 px-4 py-3 text-sm"><summary className="cursor-pointer font-medium text-fg">How to use replay</summary><p className="mt-2 text-fg-muted">Choose a tenant and a relative or absolute range. Replay can create downstream traffic, so verify the range before confirming.</p></details><Card className="mt-6"><CardHeader title="Replay range"/><div className="flex flex-wrap items-end gap-3 p-5"><label className="text-sm text-fg-muted">Tenant<Combobox value={tenant} onChange={setTenant} options={tenants.map((t) => ({ value: t.ID, label: t.Name }))} placeholder="Select tenant"/></label><p className="text-sm text-fg-muted">Global range: <span className="font-medium text-fg">{timeRangeLabel}</span>.</p><Button onClick={() => void replay()} loading={loading} disabled={!tenant}>Replay events</Button></div></Card>{error && <p className="mt-4 rounded border border-danger/30 bg-danger/5 p-3 text-sm text-danger">{error}</p>}{result && <Card className="mt-6"><CardHeader title="Replay result"/><pre className="max-h-80 overflow-auto p-5 text-sm text-fg-muted">{result}</pre></Card>}</div>;
}
