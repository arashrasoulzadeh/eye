"use client";
import { useEffect, useState } from "react";
import { api, Tenant } from "@/lib/api";
import { Card, CardHeader } from "@/components/Card";
import { Button } from "@/components/Button";
import { Combobox } from "@/components/Combobox";
import { useTenant } from "@/components/TenantContext";
import { useTimeRange } from "@/components/TimeRangeContext";
type Row = { cohort_period: string; cohort_size: number; active_by_n: number[]; retained_pct: number[] };
export default function RetentionPage() {
  const { tenantId: globalTenant } = useTenant();
  const { from: globalFrom, to: globalTo } = useTimeRange();
  const [tenants, setTenants] = useState<Tenant[]>([]), [tenant, setTenant] = useState(""), [granularity, setGranularity] = useState<"daily" | "weekly">("daily"), [rows, setRows] = useState<Row[]>([]), [loading, setLoading] = useState(false), [error, setError] = useState("");
  useEffect(() => { api.tenants.list().then((items) => { setTenants(items); if (items[0]) setTenant(items[0].ID); }).catch((e) => setError(e.message)); }, []);
  useEffect(() => { if (globalTenant && globalTenant !== tenant) setTenant(globalTenant); }, [globalTenant, tenant]);
  useEffect(() => { if (!tenant || !globalFrom || !globalTo) return; setLoading(true); api.analytics.retention(tenant, granularity, { from_ts: globalFrom, to_ts: globalTo }).then(setRows).catch((e) => setError(e.message)).finally(() => setLoading(false)); }, [tenant, granularity, globalFrom, globalTo]);
  const refresh = () => { if (!tenant || !globalFrom || !globalTo) return; setLoading(true); api.analytics.retention(tenant, granularity, { from_ts: globalFrom, to_ts: globalTo }).then(setRows).catch((e) => setError(e.message)).finally(() => setLoading(false)); };
  return <div><h1 className="text-xl font-semibold text-fg">Retention Analysis</h1><p className="mt-1 text-sm text-fg-muted">Compare how cohorts return in later daily or weekly periods.</p><Card className="mt-6"><CardHeader title="Retention controls" /><div className="flex flex-wrap gap-3 p-5"><Combobox value={tenant} onChange={setTenant} options={tenants.map((item) => ({ value: item.ID, label: item.Name }))} placeholder="Select tenant" /><Combobox value={granularity} onChange={(value) => setGranularity(value as "daily" | "weekly")} options={[{ value: "daily", label: "Daily cohorts" }, { value: "weekly", label: "Weekly cohorts" }]} placeholder="Granularity" /><Button variant="secondary" onClick={refresh} loading={loading}>Refresh</Button></div></Card>{error && <p className="mt-4 rounded border border-danger/30 bg-danger/5 p-3 text-sm text-danger">{error}</p>}<Card className="mt-6"><CardHeader title="Retention matrix" /><div className="overflow-x-auto">{loading && !rows.length ? <p className="p-5 text-sm text-fg-muted">Loading retention data…</p> : <table className="w-full text-left text-sm"><thead><tr className="border-b border-border text-xs uppercase text-fg-muted"><th className="px-5 py-3">Cohort</th><th className="px-5 py-3">Size</th>{rows[0]?.retained_pct.map((_, index) => <th key={index} className="px-5 py-3">{index === 0 ? "Initial" : `+${index}`}</th>)}</tr></thead><tbody>{rows.map((row) => <tr key={row.cohort_period} className="border-b border-border"><td className="px-5 py-3 font-medium">{row.cohort_period}</td><td className="px-5 py-3">{row.cohort_size.toLocaleString()}</td>{row.retained_pct.map((pct, index) => <td key={index} className="px-5 py-3" title={`${row.active_by_n[index] || 0} active sessions`}>{pct.toFixed(1)}%</td>)}</tr>)}</tbody></table>}{!loading && !rows.length && <p className="p-5 text-sm text-fg-muted">No retention data for this tenant.</p>}</div></Card></div>;
}
