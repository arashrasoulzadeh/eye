"use client";
import { useCallback, useEffect, useState } from "react";
import { useTimeRange } from "@/components/TimeRangeContext";
import { api } from "@/lib/api";
import { Card, CardHeader } from "@/components/Card";
import { DataTable, Column, TableQuery } from "@/components/DataTable";
type Audit = { ID: string; Action: string; Resource: string; ActorKey: string; StatusCode: number; BodySnippet: string; CreatedAt: number };
export default function AuditLogPage() {
  const [items, setItems] = useState<Audit[]>([]), [total, setTotal] = useState(0), [loading, setLoading] = useState(false), [error, setError] = useState("");
  const { from, to } = useTimeRange();
  const load = useCallback(async (query?: TableQuery) => { if (!to) return; setLoading(true); setError(""); try { const page = await api.operations.auditLogPage({ from_ts: from, to_ts: to, filter: query?.filter, dir: query?.sortDir, limit: 10, offset: (query?.page ?? 0) * 10 }); setItems(page.items as Audit[]); setTotal(page.total); } catch (e) { setError(e instanceof Error ? e.message : "Failed to load audit log"); } finally { setLoading(false); } }, [from, to]);
  useEffect(() => { void load(); }, [load]);
  const columns: Column<Audit>[] = [{ key: "action", header: "Action", render: (row) => <span className="font-medium">{row.Action}</span>, sortValue: (row) => row.Action }, { key: "resource", header: "Resource", render: (row) => <span className="break-all">{row.Resource}</span>, sortValue: (row) => row.Resource }, { key: "status", header: "Status", render: (row) => row.StatusCode, sortValue: (row) => row.StatusCode }, { key: "created", header: "Created", render: (row) => new Date(row.CreatedAt * 1000).toLocaleString(), sortValue: (row) => row.CreatedAt }];
  return <div><h1 className="text-xl font-semibold text-fg">Audit Log</h1><p className="mt-1 text-sm text-fg-muted">Review administrative actions newest first. Filtering, sorting, and pagination are API-backed.</p><details className="mt-4 rounded-lg border border-border bg-surface-2 px-4 py-3 text-sm"><summary className="cursor-pointer font-medium text-fg">Audit log help</summary><p className="mt-2 text-fg-muted">Search by action, resource, actor, or request body. Inspect a row for the complete stored record.</p></details>{error && <p className="mt-4 rounded border border-danger/30 bg-danger/5 p-3 text-sm text-danger">{error}</p>}<Card className="mt-6"><CardHeader title={`Audit entries (${total})`}/><DataTable columns={columns} rows={items} rowKey={(row) => row.ID} emptyMessage="No audit entries yet." loading={loading} serverSide totalRows={total} defaultSortKey="created" onQueryChange={(query) => void load(query)} /></Card></div>;
}
