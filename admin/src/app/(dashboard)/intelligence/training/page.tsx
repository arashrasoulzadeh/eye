"use client";

import { useEffect, useMemo, useState } from "react";
import { api, Tenant } from "@/lib/api";
import { useTenant } from "@/components/TenantContext";
import { Card, CardHeader } from "@/components/Card";
import { Combobox } from "@/components/Combobox";
import { BarBreakdown } from "@/components/Chart";

type Run = Awaited<ReturnType<typeof api.intelligence.trainingRuns>>[number];
function metrics(run: Run): Record<string, unknown> { try { return JSON.parse(run.metrics) as Record<string, unknown>; } catch { return {}; } }
function count(value: unknown) { return typeof value === "number" ? value.toLocaleString() : "—"; }

export default function TrainingHistoryPage() {
  const { tenantId: globalTenant } = useTenant();
  const [tenant, setTenant] = useState(globalTenant), [tenants, setTenants] = useState<Tenant[]>([]), [runs, setRuns] = useState<Run[]>([]), [error, setError] = useState("");
  useEffect(() => { api.tenants.list().then((items) => setTenants(Array.isArray(items) ? items : [])).catch((e) => setError(e.message)); }, []);
  useEffect(() => { if (globalTenant) setTenant(globalTenant); }, [globalTenant]);
  useEffect(() => { if (!tenant) { setRuns([]); return; } api.intelligence.trainingRuns(tenant).then(setRuns).catch((e) => setError(e.message)); }, [tenant]);
  const latest = runs[0];
  const latestMetrics = latest ? metrics(latest) : {};
  const eventTypes = useMemo(() => Object.entries((latestMetrics.event_types || {}) as Record<string, number>).map(([label, value]) => ({ label, value: Number(value) })).sort((a, b) => b.value - a.value).slice(0, 10), [latestMetrics]);
  return <div><div className="flex flex-wrap items-end justify-between gap-4"><div><h1 className="text-xl font-semibold text-fg">Model training</h1><p className="mt-1 text-sm text-fg-muted">Automatic behavioral training uses all events plus user profiles. Unsupervised runs do not have class balance, accuracy, precision, recall, F1, or epochs.</p></div><Combobox value={tenant} onChange={setTenant} options={tenants.map((item) => ({ value: item.ID, label: item.Name }))} placeholder="Select tenant" /></div>{error && <p className="mt-4 rounded border border-danger/30 bg-danger/5 p-3 text-sm text-danger">{error}</p>}{latest && <div className="mt-6 grid gap-4 sm:grid-cols-2 lg:grid-cols-4"><Card><p className="p-4 text-xs text-fg-muted">Latest model</p><p className="px-4 pb-4 text-lg font-semibold text-fg">{latest.label || "behavior"}</p></Card><Card><p className="p-4 text-xs text-fg-muted">Events used</p><p className="px-4 pb-4 text-lg font-semibold text-fg">{count(latestMetrics.events ?? latest.examples)}</p></Card><Card><p className="p-4 text-xs text-fg-muted">Profiles used</p><p className="px-4 pb-4 text-lg font-semibold text-fg">{count(latestMetrics.profiles)}</p></Card><Card><p className="p-4 text-xs text-fg-muted">Anomalies detected</p><p className="px-4 pb-4 text-lg font-semibold text-fg">{count(latestMetrics.anomalies)}</p></Card></div>}{latest && <Card className="mt-6"><CardHeader title="Latest event distribution" /><div className="p-5"><BarBreakdown data={eventTypes} legend="Events" height={280} /></div></Card>}<Card className="mt-6"><CardHeader title={`Training runs (${runs.length})`} /><div className="overflow-x-auto"><table className="w-full text-left text-sm"><thead><tr className="border-b border-border text-xs uppercase text-fg-muted"><th className="px-5 py-3">Model</th><th className="px-5 py-3">Events</th><th className="px-5 py-3">Profiles</th><th className="px-5 py-3">Sessions</th><th className="px-5 py-3">Anomalies</th><th className="px-5 py-3">Method</th><th className="px-5 py-3">Created</th></tr></thead><tbody>{runs.map((run) => { const data = metrics(run); return <tr key={run.id} className="border-b border-border last:border-0"><td className="px-5 py-3 font-medium">{run.label || "behavior"}</td><td className="px-5 py-3">{count(data.events ?? run.examples)}</td><td className="px-5 py-3">{count(data.profiles)}</td><td className="px-5 py-3">{count(data.sessions)}</td><td className="px-5 py-3">{count(data.anomalies)}</td><td className="px-5 py-3">{String(data.method || "behavioral_baseline")}</td><td className="px-5 py-3 text-xs text-fg-muted">{run.created_at ? new Date(run.created_at).toLocaleString() : "—"}</td></tr>; })}</tbody></table>{!runs.length && <p className="p-5 text-sm text-fg-muted">No training runs for this tenant.</p>}</div></Card></div>;
}
