"use client";
import { useCallback, useEffect, useState } from "react";
import { api } from "@/lib/api";
import { Card, CardHeader } from "@/components/Card";
import { DataTable, Column, TableQuery } from "@/components/DataTable";
type Anomaly = { tenant_id: string; event_name: string; count: number; mean: number; stddev: number; z_score: number };
export default function AnomaliesPage() {
  const [items, setItems] = useState<Anomaly[]>([]), [total, setTotal] = useState(0), [loading, setLoading] = useState(false), [error, setError] = useState("");
  const load = useCallback(async (query?: TableQuery) => { setLoading(true); setError(""); try { const page = await api.operations.anomaliesPage({ filter: query?.filter, dir: query?.sortDir, limit: 10, offset: (query?.page ?? 0) * 10 }); setItems(page.items as Anomaly[]); setTotal(page.total); } catch (e) { setError(e instanceof Error ? e.message : "Failed to load anomalies"); } finally { setLoading(false); } }, []);
  useEffect(() => { void load(); }, [load]);
  const columns: Column<Anomaly>[] = [{ key: "tenant", header: "Tenant", render: (item) => item.tenant_id, sortValue: (item) => item.tenant_id }, { key: "event", header: "Event", render: (item) => <span className="font-medium">{item.event_name}</span>, sortValue: (item) => item.event_name }, { key: "count", header: "Count", render: (item) => item.count.toLocaleString(), sortValue: (item) => item.count }, { key: "z_score", header: "Z-score", render: (item) => item.z_score.toFixed(2), sortValue: (item) => item.z_score }];
  return <div><h1 className="text-xl font-semibold text-fg">Anomalies</h1><p className="mt-1 text-sm text-fg-muted">Review statistically unusual event volume detected from recent tenant history.</p><details className="mt-4 rounded-lg border border-border bg-surface-2 px-4 py-3 text-sm"><summary className="cursor-pointer font-medium text-fg">How anomalies work</summary><p className="mt-2 text-fg-muted">Anomalies use a rolling z-score. Filter and sort requests are sent to the API.</p></details>{error && <p className="mt-4 rounded border border-danger/30 bg-danger/5 p-3 text-sm text-danger">{error}</p>}<Card className="mt-6"><CardHeader title={`Current anomalies (${total})`}/><DataTable columns={columns} rows={items} rowKey={(item) => `${item.tenant_id}:${item.event_name}`} emptyMessage="No current anomalies." loading={loading} serverSide totalRows={total} defaultSortKey="z_score" onQueryChange={(query) => void load(query)} /></Card></div>;
}
