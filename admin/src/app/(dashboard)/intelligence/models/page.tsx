"use client";

import { useEffect, useState } from "react";
import { api, Tenant } from "@/lib/api";
import { useTenant } from "@/components/TenantContext";
import { Card, CardHeader } from "@/components/Card";
import { Button } from "@/components/Button";
import { Combobox } from "@/components/Combobox";

export default function ModelRegistryPage() {
  const { tenantId: globalTenant } = useTenant();
  const [tenant, setTenant] = useState(globalTenant), [tenants, setTenants] = useState<Tenant[]>([]), [model, setModel] = useState<{ scope: string; manifest?: { name: string; version: string; task: string; algorithm: string; source: string; created_at: string } } | null>(null), [message, setMessage] = useState(""), [busy, setBusy] = useState(false);
  useEffect(() => { api.tenants.list().then((items) => setTenants(Array.isArray(items) ? items : [])).catch(() => undefined); }, []);
  useEffect(() => { if (globalTenant) setTenant(globalTenant); }, [globalTenant]);
  async function inspect() { if (!tenant) return; setBusy(true); setMessage(""); try { setModel(await api.intelligence.model(tenant, "buyer")); } catch { setModel(null); setMessage("No valid buyer model is installed for this tenant or globally."); } finally { setBusy(false); } }
  async function train() { if (!tenant) return; setBusy(true); setMessage(""); try { await api.intelligence.refresh(tenant); await api.intelligence.train(tenant, "auto"); setMessage("Unsupervised behavior baseline trained from tenant activity."); } catch { setMessage("Collect tenant activity first, then run automatic training again."); } finally { setBusy(false); } }
  async function exportModel() { if (!tenant) return; setBusy(true); setMessage(""); try { const result = await api.intelligence.exportModel(tenant, "buyer", "global"); setMessage(`Exported version ${result.version} with checksum ${result.sha256.slice(0, 16)}…`); } catch { setMessage("Export failed. Train a model with sufficient labels first."); } finally { setBusy(false); } }
  return <div><div className="flex flex-wrap items-end justify-between gap-4"><div><h1 className="text-xl font-semibold text-fg">Model registry</h1><p className="mt-1 text-sm text-fg-muted">Manage offline global and tenant model artifacts.</p></div><Combobox value={tenant} onChange={setTenant} options={tenants.map((item) => ({ value: item.ID, label: item.Name }))} placeholder="Select tenant"/></div><Card className="mt-6"><CardHeader title="Automatic behavior model"/><p className="px-5 text-sm text-fg-muted">First-run training is unsupervised: it uses the tenant’s activity snapshots, behavioral segments, and anomaly scores. Verified labels are optional.</p><div className="flex flex-wrap gap-2 p-5"><Button onClick={() => void inspect()} loading={busy} disabled={!tenant}>Check active model</Button><Button variant="secondary" onClick={() => void train()} loading={busy} disabled={!tenant}>Train automatically</Button><Button variant="secondary" onClick={() => void exportModel()} loading={busy} disabled={!tenant}>Export to registry</Button></div>{message && <p className="border-t border-border px-5 py-3 text-sm text-fg-muted">{message}</p>}{model?.manifest && <div className="grid gap-4 border-t border-border p-5 text-sm sm:grid-cols-2 lg:grid-cols-4"><div><p className="text-xs text-fg-muted">Scope</p><p className="mt-1 font-medium">{model.scope}</p></div><div><p className="text-xs text-fg-muted">Version</p><p className="mt-1 font-medium">{model.manifest.version}</p></div><div><p className="text-xs text-fg-muted">Algorithm</p><p className="mt-1 font-medium">{model.manifest.algorithm}</p></div><div><p className="text-xs text-fg-muted">Source</p><p className="mt-1 font-medium">{model.manifest.source}</p></div></div>}</Card></div>;
}
