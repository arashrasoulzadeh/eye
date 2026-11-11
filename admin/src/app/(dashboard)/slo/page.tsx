"use client";

import { useCallback, useEffect, useState } from "react";
import { api, ApiError } from "@/lib/api";
import { Card, CardHeader, StatCard } from "@/components/Card";
import { Button } from "@/components/Button";
import { Badge } from "@/components/Badge";

export default function SLOPage() {
  const [data, setData] = useState<Awaited<ReturnType<typeof api.slo>> | null>(null), [loading, setLoading] = useState(true), [error, setError] = useState("");
  const load = useCallback(async () => { setLoading(true); try { setData(await api.slo()); setError(""); } catch (e) { setError(e instanceof ApiError ? e.message : "Failed to load SLO data"); } finally { setLoading(false); } }, []);
  useEffect(() => { void load(); }, [load]);
  return <div><div className="flex items-end justify-between gap-4"><div><h1 className="text-xl font-semibold text-fg">SLO &amp; Error Budget</h1><p className="mt-1 text-sm text-fg-muted">Process availability observed by the API since its current start. Use Prometheus for historical windows.</p></div><Button variant="secondary" onClick={() => void load()} loading={loading}>Refresh</Button></div><details className="mt-4 rounded-lg border border-border bg-surface-2 px-4 py-3 text-sm"><summary className="cursor-pointer font-medium text-fg">How this is calculated</summary><p className="mt-2 text-fg-muted">5xx responses consume the error budget. The default target is 99.9%; this view is a fast operational snapshot, while long-term SLO reporting belongs in Prometheus/Grafana.</p></details>{error && <p className="mt-4 rounded border border-danger/30 bg-danger/5 p-3 text-sm text-danger">{error}</p>}{data && <><div className="mt-6 grid gap-4 md:grid-cols-4"><StatCard label="Availability" value={`${(data.availability * 100).toFixed(3)}%`} hint={`Target ${(data.target * 100).toFixed(2)}%`}/><StatCard label="Requests" value={data.requests.toLocaleString()}/><StatCard label="5xx errors" value={data.errors.toLocaleString()}/><StatCard label="Budget remaining" value={data.error_budget_remaining.toFixed(2)} hint={`of ${data.error_budget.toFixed(2)} allowed`}/></div><Card className="mt-6"><CardHeader title="Current status"/><div className="flex items-center justify-between p-5"><div><p className="font-medium text-fg">Availability objective</p><p className="text-sm text-fg-muted">{data.errors} server errors across {data.requests} observed requests</p></div><Badge tone={data.within_slo ? "success" : "danger"}>{data.within_slo ? "Within SLO" : "Burned through budget"}</Badge></div></Card></>}</div>;
}
