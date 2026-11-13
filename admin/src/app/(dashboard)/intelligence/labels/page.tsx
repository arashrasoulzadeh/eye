"use client";

import { useCallback, useEffect, useState } from "react";
import { api, Tenant } from "@/lib/api";
import { useTenant } from "@/components/TenantContext";
import { Card, CardHeader } from "@/components/Card";
import { Button } from "@/components/Button";
import { Combobox } from "@/components/Combobox";

type Suggestion = Awaited<ReturnType<typeof api.intelligence.suggestions>>[number];

export default function IntelligenceLabelsPage() {
  const { tenantId: globalTenant } = useTenant();
  const [tenant, setTenant] = useState(globalTenant), [tenants, setTenants] = useState<Tenant[]>([]), [rows, setRows] = useState<Suggestion[]>([]), [busy, setBusy] = useState(false), [message, setMessage] = useState("");
  const load = useCallback(async () => { if (!tenant) return; setBusy(true); try { setRows(await api.intelligence.suggestions(tenant)); } catch (error) { setMessage(error instanceof Error ? error.message : "Could not load suggestions"); } finally { setBusy(false); } }, [tenant]);
  useEffect(() => { api.tenants.list().then((items) => setTenants(Array.isArray(items) ? items : [])).catch(() => undefined); }, []);
  useEffect(() => { if (globalTenant) setTenant(globalTenant); }, [globalTenant]);
  useEffect(() => { void load(); }, [load]);
  async function review(row: Suggestion, decision: "accepted" | "denied") { if (!tenant) return; setBusy(true); setMessage(""); try { await api.intelligence.reviewSuggestion(tenant, row.fingerprint, row.label, decision); setRows((current) => current.filter((item) => !(item.fingerprint === row.fingerprint && item.label === row.label))); } catch (error) { setMessage(error instanceof Error ? error.message : "Could not save review"); } finally { setBusy(false); } }
  return <div><div className="flex flex-wrap items-end justify-between gap-4"><div><h1 className="text-xl font-semibold text-fg">Suggested user labels</h1><p className="mt-1 text-sm text-fg-muted">Review explainable suggestions. Accept or deny each one; no manual fingerprint entry is required.</p></div><div className="flex gap-2"><Combobox value={tenant} onChange={setTenant} options={tenants.map((item) => ({ value: item.ID, label: item.Name }))} placeholder="Select tenant"/><Button onClick={() => void load()} loading={busy} disabled={!tenant}>Refresh</Button></div></div>{message && <p className="mt-4 rounded border border-danger/30 p-3 text-sm text-danger">{message}</p>}<Card className="mt-6"><CardHeader title="Review queue" action={<span className="text-xs text-fg-muted">{rows.length} suggestions</span>}/><div className="overflow-x-auto"><table className="w-full text-left text-sm"><thead><tr className="border-b border-border text-xs uppercase text-fg-muted"><th className="px-5 py-3">User</th><th className="px-5 py-3">Suggested label</th><th className="px-5 py-3">Confidence</th><th className="px-5 py-3">Reasons</th><th className="px-5 py-3">Review</th></tr></thead><tbody>{rows.map((row) => <tr key={`${row.fingerprint}:${row.label}`} className="border-b border-border last:border-0"><td className="px-5 py-3"><div className="font-mono text-xs">{row.snapshot.ip || row.fingerprint.slice(0, 12)}</div><div className="max-w-sm truncate text-xs text-fg-muted">{row.snapshot.user_agent || "Unknown user agent"}</div></td><td className="px-5 py-3 font-medium">{row.label}</td><td className="px-5 py-3">{(row.confidence * 100).toFixed(0)}%</td><td className="px-5 py-3 text-fg-muted">{row.reasons.join("; ") || "Behavioral model signal"}</td><td className="px-5 py-3"><div className="flex gap-2"><Button onClick={() => void review(row, "accepted")} disabled={busy}>Accept</Button><Button variant="secondary" onClick={() => void review(row, "denied")} disabled={busy}>Deny</Button></div></td></tr>)}</tbody></table>{!rows.length && !busy && <p className="p-5 text-sm text-fg-muted">No pending suggestions. Refresh intelligence after more user activity arrives.</p>}</div></Card><Card className="mt-6"><p className="p-5 text-sm text-fg-muted">Accepted reviews become tenant-supervised examples. The global behavioral baseline supplies shared patterns; tenant-specific accepted reviews provide the local adaptation layer.</p></Card></div>;
}
