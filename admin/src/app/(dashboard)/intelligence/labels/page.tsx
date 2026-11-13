"use client";

import { FormEvent, useEffect, useState } from "react";
import { api, Tenant } from "@/lib/api";
import { useTenant } from "@/components/TenantContext";
import { Card, CardHeader } from "@/components/Card";
import { Button } from "@/components/Button";
import { Combobox } from "@/components/Combobox";

const LABELS = ["buyer", "potential_buyer", "returning_user", "new_user", "bot", "automation", "ai_crawler", "high_intent", "inactive"];

export default function IntelligenceLabelsPage() {
  const { tenantId: globalTenant } = useTenant();
  const [tenant, setTenant] = useState(globalTenant), [tenants, setTenants] = useState<Tenant[]>([]), [fingerprint, setFingerprint] = useState(""), [label, setLabel] = useState("buyer"), [message, setMessage] = useState(""), [busy, setBusy] = useState(false);
  useEffect(() => { api.tenants.list().then((items) => setTenants(Array.isArray(items) ? items : [])).catch(() => undefined); }, []);
  useEffect(() => { if (globalTenant) setTenant(globalTenant); }, [globalTenant]);
  async function submit(event: FormEvent) { event.preventDefault(); if (!tenant || !fingerprint.trim()) return; setBusy(true); setMessage(""); try { await api.intelligence.label(tenant, fingerprint.trim(), label); setFingerprint(""); setMessage("Label saved for future training."); } catch (error) { setMessage(error instanceof Error ? error.message : "Could not save label"); } finally { setBusy(false); } }
  return <div><div className="flex flex-wrap items-end justify-between gap-4"><div><h1 className="text-xl font-semibold text-fg">User labels</h1><p className="mt-1 text-sm text-fg-muted">Add verified labels to improve supervised offline models.</p></div><Combobox value={tenant} onChange={setTenant} options={tenants.map((item) => ({ value: item.ID, label: item.Name }))} placeholder="Select tenant"/></div><Card className="mt-6"><CardHeader title="Add verified label"/><form onSubmit={submit} className="grid gap-4 p-5 sm:grid-cols-[1fr_220px_auto]"><input value={fingerprint} onChange={(event) => setFingerprint(event.target.value)} placeholder="User fingerprint from Intelligence" className="rounded-lg border border-border bg-surface px-3 py-2 text-sm text-fg"/><select value={label} onChange={(event) => setLabel(event.target.value)} className="rounded-lg border border-border bg-surface px-3 py-2 text-sm text-fg">{LABELS.map((item) => <option key={item}>{item}</option>)}</select><Button type="submit" loading={busy} disabled={!tenant || !fingerprint.trim()}>Save label</Button></form>{message && <p className="border-t border-border px-5 py-3 text-sm text-fg-muted">{message}</p>}</Card><Card className="mt-6"><CardHeader title="How labels are used"/><p className="p-5 text-sm text-fg-muted">Labels are tenant-scoped and become supervised examples. The trainer treats the selected label as positive and other known labels as negative examples. Training requires balanced data before a model can be created.</p></Card></div>;
}
