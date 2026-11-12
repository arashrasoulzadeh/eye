"use client";

import { useEffect, useMemo, useState } from "react";
import { useRouter } from "next/navigation";

const COMMANDS = [
  ["Dashboard", "/dashboard"], ["Analytics", "/analytics"], ["Live Stream", "/stream"], ["Events", "/events"],
  ["Sessions", "/sessions"], ["Users", "/users"], ["Funnels", "/analytics/funnel"], ["Retention", "/analytics/retention"],
  ["Health Center", "/health"], ["Alerts", "/alerts"], ["Tenants", "/tenants"], ["Settings", "/settings"],
] as const;

export function CommandPalette() {
  const router = useRouter();
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState("");
  useEffect(() => { const onKey = (event: KeyboardEvent) => { if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === "k") { event.preventDefault(); setOpen(true); } if (event.key === "Escape") setOpen(false); }; window.addEventListener("keydown", onKey); return () => window.removeEventListener("keydown", onKey); }, []);
  const filtered = useMemo(() => COMMANDS.filter(([label, path]) => `${label} ${path}`.toLowerCase().includes(query.toLowerCase())), [query]);
  if (!open) return <button type="button" onClick={() => setOpen(true)} className="hidden rounded-md border border-border bg-surface px-2.5 py-1 text-xs text-fg-muted hover:border-accent sm:inline-flex">Search <kbd className="ml-2 rounded bg-surface-2 px-1.5">⌘K</kbd></button>;
  return <div className="fixed inset-0 z-50 flex items-start justify-center bg-black/30 px-4 pt-[15vh]" onMouseDown={() => setOpen(false)}><div className="w-full max-w-lg overflow-hidden rounded-xl border border-border bg-surface shadow-2xl" onMouseDown={(event) => event.stopPropagation()}><input autoFocus value={query} onChange={(event) => setQuery(event.target.value)} placeholder="Go to a page…" className="w-full border-b border-border bg-surface px-4 py-3 text-sm text-fg outline-none" /><div className="max-h-80 overflow-auto p-2">{filtered.map(([label, path]) => <button key={path} type="button" onClick={() => { setOpen(false); router.push(path); }} className="flex w-full items-center justify-between rounded-lg px-3 py-2 text-left text-sm text-fg hover:bg-accent/10 hover:text-accent"><span>{label}</span><span className="text-xs text-fg-muted">{path}</span></button>)}{!filtered.length && <p className="p-3 text-sm text-fg-muted">No matching page.</p>}</div></div></div>;
}
