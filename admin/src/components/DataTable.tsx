"use client";
import { ReactNode, useMemo, useState } from "react";

export type Column<T> = {
  key: string;
  header: string;
  render: (row: T) => ReactNode;
  sortValue?: (row: T) => string | number;
  filterValue?: (row: T) => string | number;
};

export type TableQuery = { filter: string; sortKey: string | null; sortDir: "asc" | "desc"; page: number };

export function DataTable<T>({
  columns,
  rows,
  rowKey,
  emptyMessage = "No data yet.",
  pageSize = 10,
  defaultSortKey,
  serverSide = false,
  onQueryChange,
  totalRows,
  showActions = true,
  loading = false,
}: {
  columns: Column<T>[];
  rows: T[];
  rowKey: (row: T) => string;
  emptyMessage?: string;
  pageSize?: number;
  defaultSortKey?: string;
  serverSide?: boolean;
  onQueryChange?: (query: TableQuery) => void;
  totalRows?: number;
  showActions?: boolean;
  loading?: boolean;
}) {
  const [filter, setFilter] = useState("");
  const [sortKey, setSortKey] = useState<string | null>(() =>
    defaultSortKey ?? columns.find((column) => ["last_seen", "last_fired", "fired", "time", "created", "started", "version"].includes(column.key))?.key ?? null,
  );
  const [sortDir, setSortDir] = useState<"asc" | "desc">(() =>
    defaultSortKey || columns.some((column) => ["last_seen", "last_fired", "fired", "time", "created", "started", "version"].includes(column.key)) ? "desc" : "asc",
  );
  const [page, setPage] = useState(0);
  const [inspectRow, setInspectRow] = useState<T | null>(null);
  const [copied, setCopied] = useState(false);

  function notify(next: Partial<TableQuery>) {
    const query = { filter, sortKey, sortDir, page, ...next };
    setPage(query.page);
    onQueryChange?.(query);
  }

  const filtered = useMemo(() => {
    const query = filter.trim().toLowerCase();
    if (serverSide || !query) return rows;
    return rows.filter((row) => {
      const values = columns.map((column) => column.filterValue?.(row) ?? column.sortValue?.(row));
      return values.some((value) => value !== undefined && String(value).toLowerCase().includes(query)) ||
        JSON.stringify(row).toLowerCase().includes(query);
    });
  }, [rows, columns, filter, serverSide]);

  const sorted = useMemo(() => {
    if (serverSide || !sortKey) return filtered;
    const col = columns.find((c) => c.key === sortKey);
    const copy = [...filtered];
    if (!col) return copy;
    copy.sort((a, b) => {
      const av = col.sortValue?.(a) ?? JSON.stringify(a);
      const bv = col.sortValue?.(b) ?? JSON.stringify(b);
      const cmp = av < bv ? -1 : av > bv ? 1 : 0;
      return sortDir === "asc" ? cmp : -cmp;
    });
    return copy;
  }, [filtered, sortKey, sortDir, columns, serverSide]);

  const pageCount = Math.max(1, Math.ceil((serverSide ? totalRows ?? rows.length : sorted.length) / pageSize));
  const clampedPage = Math.min(page, pageCount - 1);
  const pageRows = serverSide ? rows : sorted.slice(clampedPage * pageSize, clampedPage * pageSize + pageSize);

  function toggleSort(col: Column<T>) {
    if (sortKey === col.key) {
      const nextDir = sortDir === "asc" ? "desc" : "asc";
      setSortDir(nextDir);
      notify({ sortKey: col.key, sortDir: nextDir, page: 0 });
    } else {
      setSortKey(col.key);
      setSortDir("asc");
      notify({ sortKey: col.key, sortDir: "asc", page: 0 });
    }
  }

  if (loading && rows.length === 0) {
    return <div className="space-y-3 px-5 py-10" aria-label="Loading table"><div className="h-4 animate-pulse rounded bg-surface-2"/><div className="h-4 animate-pulse rounded bg-surface-2"/><div className="h-4 animate-pulse rounded bg-surface-2"/></div>;
  }
  if (rows.length === 0) {
    return <div className="px-5 py-10 text-center text-sm text-fg-muted">{emptyMessage}</div>;
  }

  return (
    <div>
      <div className="flex items-center justify-between border-b border-border px-5 py-3">
        <input
          value={filter}
          onChange={(event) => {
            setFilter(event.target.value);
            setPage(0);
            notify({ filter: event.target.value, page: 0 });
          }}
          placeholder="Filter all fields…"
          aria-label="Filter table"
          className="w-full max-w-xs rounded-md border border-border bg-surface px-3 py-1.5 text-sm text-fg outline-none placeholder:text-fg-muted focus:border-accent"
        />
        <span className="text-xs text-fg-muted">{serverSide ? totalRows ?? rows.length : filtered.length} rows</span>
      </div>
      {loading && <div className="h-1 animate-pulse bg-accent" aria-label="Refreshing table" />}
      <div className="overflow-x-auto">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-border text-left text-xs font-medium uppercase tracking-wide text-fg-muted">
              {columns.map((col) => (
                <th
                  key={col.key}
                  onClick={() => toggleSort(col)}
                  className="cursor-pointer select-none px-5 py-2.5 hover:text-fg"
                >
                  {col.header}
                  {sortKey === col.key && (sortDir === "asc" ? " ▲" : " ▼")}
                </th>
              ))}
              {showActions && <th className="px-5 py-2.5 text-right">Actions</th>}
            </tr>
          </thead>
          <tbody>
            {pageRows.map((row) => (
              <tr key={rowKey(row)} className="border-b border-border last:border-0 hover:bg-surface-2/60">
                {columns.map((col) => (
                  <td key={col.key} className="px-5 py-3 align-middle">
                    {col.render(row)}
                  </td>
                ))}
                {showActions && (
                  <td className="px-5 py-3 text-right">
                    <button
                      type="button"
                      aria-label={`Inspect row ${rowKey(row)}`}
                      title="Inspect row"
                      onClick={() => { setInspectRow(row); setCopied(false); }}
                      className="rounded-md p-1.5 text-fg-muted hover:bg-surface-2 hover:text-fg"
                    >
                      ◉
                    </button>
                  </td>
                )}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      {pageCount > 1 && (
        <div className="flex items-center justify-between border-t border-border px-5 py-3 text-xs text-fg-muted">
          <span>
            Page {clampedPage + 1} of {pageCount} · {serverSide ? totalRows ?? rows.length : rows.length} rows
          </span>
          <div className="flex gap-1">
            <button
              onClick={() => setPage((p) => Math.max(0, p - 1))}
              disabled={clampedPage === 0}
              className="rounded-md px-2 py-1 hover:bg-surface-2 disabled:opacity-40"
            >
              Prev
            </button>
            <button
              onClick={() => setPage((p) => Math.min(pageCount - 1, p + 1))}
              disabled={clampedPage >= pageCount - 1}
              className="rounded-md px-2 py-1 hover:bg-surface-2 disabled:opacity-40"
            >
              Next
            </button>
          </div>
        </div>
      )}
      {inspectRow && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4" role="dialog" aria-modal="true" aria-label="Row details">
          <div className="max-h-[80vh] w-full max-w-2xl overflow-hidden rounded-xl border border-border bg-surface shadow-xl">
            <div className="flex items-center justify-between border-b border-border px-5 py-4">
              <h2 className="font-semibold text-fg">Row details</h2>
              <div className="flex gap-2">
                <button
                  type="button"
                  onClick={() => {
                    void navigator.clipboard?.writeText(JSON.stringify(inspectRow, null, 2));
                    setCopied(true);
                  }}
                  className="rounded-md border border-border px-3 py-1.5 text-xs text-fg-muted hover:bg-surface-2"
                >
                  {copied ? "Copied" : "Copy values"}
                </button>
                <button type="button" aria-label="Close row details" onClick={() => setInspectRow(null)} className="rounded-md px-2 py-1 text-fg-muted hover:bg-surface-2">×</button>
              </div>
            </div>
            <pre className="max-h-[65vh] overflow-auto p-5 text-xs text-fg-muted">{JSON.stringify(inspectRow, null, 2)}</pre>
          </div>
        </div>
      )}
    </div>
  );
}
