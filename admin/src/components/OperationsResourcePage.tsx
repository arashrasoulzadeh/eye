"use client";
import { useCallback, useEffect, useState } from "react";
import { api, Tenant, ApiError } from "@/lib/api";
import { Card, CardHeader } from "@/components/Card";
import { Button } from "@/components/Button";
import { Combobox } from "@/components/Combobox";

export function OperationsResourcePage({ title, description, tenantScoped, loader, action }: { title: string; description: string; tenantScoped?: boolean; loader: (tenantId: string) => Promise<unknown[]>; action?: (tenantId: string, reload: () => void) => React.ReactNode }) {
  const [tenants, setTenants] = useState<Tenant[]>([]), [tenantId, setTenantId] = useState(""), [rows, setRows] = useState<unknown[]>([]), [error, setError] = useState(""), [loading, setLoading] = useState(false);
  const load = useCallback(async () => { if (tenantScoped && !tenantId) return; setLoading(true); setError(""); try { setRows(await loader(tenantId)); } catch (e) { setError(e instanceof ApiError ? e.message : "Failed to load data"); } finally { setLoading(false); } }, [loader, tenantId, tenantScoped]);
  useEffect(() => { if (tenantScoped) api.tenants.list().then((x) => { setTenants(x); if (x[0]) setTenantId(x[0].ID); }).catch((e) => setError(e.message)); }, [tenantScoped]);
  useEffect(() => { void load(); }, [load]);
  return <div><div className="flex flex-wrap items-end justify-between gap-4"><div><h1 className="text-xl font-semibold text-fg">{title}</h1><p className="mt-1 text-sm text-fg-muted">{description}</p></div><div className="flex gap-2">{tenantScoped && <Combobox value={tenantId} onChange={setTenantId} options={tenants.map((t) => ({ value: t.ID, label: t.Name }))} placeholder="Select tenant"/>}<Button variant="secondary" onClick={() => void load()} loading={loading}>Refresh</Button></div></div>{error && <p className="mt-4 text-sm text-danger">{error}</p>}{action?.(tenantId, () => void load())}<Card className="mt-6"><CardHeader title={`${title} (${rows.length})`} /><div className="divide-y divide-border">{rows.length === 0 ? <p className="p-5 text-sm text-fg-muted">No records.</p> : rows.map((row, index) => <pre key={index} className="overflow-auto px-5 py-3 text-xs text-fg-muted">{JSON.stringify(row, null, 2)}</pre>)}</div></Card></div>;
}
