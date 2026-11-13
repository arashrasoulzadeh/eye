"use client";

import { useEffect, useState } from "react";
import { api, Tenant } from "@/lib/api";
import { useTenant } from "@/components/TenantContext";
import { Card, CardHeader } from "@/components/Card";
import { Combobox } from "@/components/Combobox";

type Run = Awaited<ReturnType<typeof api.intelligence.trainingRuns>>[number];

function metric(metrics: string, key: string) { try { const value = (JSON.parse(metrics) as Record<string, unknown>)[key]; return typeof value === "number" ? `${(value * 100).toFixed(1)}%` : "—"; } catch { return "—"; } }

export default function TrainingHistoryPage() {
  const { tenantId: globalTenant } = useTenant();
  const [tenant, setTenant] = useState(globalTenant), [tenants, setTenants] = useState<Tenant[]>([]), [runs, setRuns] = useState<Run[]>([]), [error, setError] = useState("");
  useEffect(() => { api.tenants.list().then((items) => setTenants(Array.isArray(items) ? items : [])).catch((e) => setError(e.message)); }, []);
  useEffect(() => { if (globalTenant) setTenant(globalTenant); }, [globalTenant]);
  useEffect(() => { if (!tenant) { setRuns([]); return; } api.intelligence.trainingRuns(tenant).then(setRuns).catch((e) => setError(e.message)); }, [tenant]);
  return <div><div className="flex flex-wrap items-end justify-between gap-4"><div><h1 className="text-xl font-semibold text-fg">Model training</h1><p className="mt-1 text-sm text-fg-muted">Versioned training history and evaluation metrics for offline intelligence.</p></div><Combobox value={tenant} onChange={setTenant} options={tenants.map((item) => ({ value: item.ID, label: item.Name }))} placeholder="Select tenant"/></div>{error && <p className="mt-4 rounded border border-danger/30 bg-danger/5 p-3 text-sm text-danger">{error}</p>}<Card className="mt-6"><CardHeader title={`Training runs (${runs.length})`}/><div className="overflow-x-auto"><table className="w-full text-left text-sm"><thead><tr className="border-b border-border text-xs uppercase text-fg-muted"><th className="px-5 py-3">Label</th><th className="px-5 py-3">Examples</th><th className="px-5 py-3">Balance</th><th className="px-5 py-3">Accuracy</th><th className="px-5 py-3">Precision</th><th className="px-5 py-3">Recall</th><th className="px-5 py-3">F1</th><th className="px-5 py-3">Created</th></tr></thead><tbody>{runs.map((run) => <tr key={run.id} className="border-b border-border last:border-0"><td className="px-5 py-3 font-medium">{run.label || "behavior"}</td><td className="px-5 py-3">{(Number.isFinite(run.examples) ? run.examples : 0).toLocaleString()}</td><td className="px-5 py-3">{run.positive || 0} / {run.negative || 0}</td><td className="px-5 py-3">{metric(run.metrics, "accuracy")}</td><td className="px-5 py-3">{metric(run.metrics, "precision")}</td><td className="px-5 py-3">{metric(run.metrics, "recall")}</td><td className="px-5 py-3">{metric(run.metrics, "f1")}</td><td className="px-5 py-3 text-xs text-fg-muted">{run.created_at ? new Date(run.created_at).toLocaleString() : "—"}</td></tr>)}</tbody></table>{!runs.length && <p className="p-5 text-sm text-fg-muted">No training runs for this tenant.</p>}</div></Card></div>;
}
