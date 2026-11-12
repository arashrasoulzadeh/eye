"use client";

import { createContext, useContext, useEffect, useMemo, useState } from "react";

export const TIME_RANGES = [
  { value: "0.0166667", label: "Last minute" },
  { value: "0.0666667", label: "Last 4 minutes" },
  { value: "1", label: "Last hour" },
  { value: "24", label: "Last 24 hours" },
  { value: "168", label: "Last 7 days" },
  { value: "720", label: "Last 30 days" },
];
const UNIT_HOURS: Record<string, number> = { m: 1 / 60, min: 1 / 60, mins: 1 / 60, minute: 1 / 60, minutes: 1 / 60, h: 1, hr: 1, hrs: 1, hour: 1, hours: 1, d: 24, day: 24, days: 24, w: 168, wk: 168, week: 168, weeks: 168, mo: 720, mon: 720, month: 720, months: 720 };

/** Converts Kibana-style relative expressions into a duration in hours. */
export function parseTimeRange(input: string): number | null {
  const value = input.trim().toLowerCase().replace(/\s+/g, " ");
  if (!value) return null;
  const compact = value.match(/^last\s*([0-9]+(?:\.[0-9]+)?)\s*([a-z]+)$/) || value.match(/^([0-9]+(?:\.[0-9]+)?)\s*([a-z]+)$/);
  if (compact) {
    const amount = Number(compact[1]);
    const unit = UNIT_HOURS[compact[2]];
    return Number.isFinite(amount) && unit ? amount * unit : null;
  }
  const nowMinus = value.match(/^now\s*-\s*([0-9]+(?:\.[0-9]+)?)\s*([a-z]+)$/);
  if (nowMinus) {
    const unit = UNIT_HOURS[nowMinus[2]];
    return unit ? Number(nowMinus[1]) * unit : null;
  }
  const named = value.match(/^last\s+(minute|hour|day|week|month)$/);
  return named ? UNIT_HOURS[named[1]] : null;
}

export function formatTimeRange(hours: string): string {
  const numeric = Number(hours);
  const preset = TIME_RANGES.find((item) => Math.abs(Number(item.value) - numeric) < 0.00001);
  if (preset) return preset.label;
  if (numeric < 1) return `Last ${Math.round(numeric * 60)} minutes`;
  if (numeric % 720 === 0) return `Last ${numeric / 720} months`;
  if (numeric % 168 === 0) return `Last ${numeric / 168} weeks`;
  if (numeric % 24 === 0) return `Last ${numeric / 24} days`;
  return `Last ${numeric} hours`;
}
function isValidHours(value: string | null): value is string {
  const numeric = Number(value);
  return Boolean(value && Number.isFinite(numeric) && numeric >= 1 / 60 && numeric <= 2160 * 24);
}
type TimeRangeValue = { hours: string; from: number; to: number };
type TimeRangeContextValue = TimeRangeValue & { setHours: (hours: string) => void };
const TimeRangeContext = createContext<TimeRangeContextValue | null>(null);
const STORAGE_KEY = "eye.admin.time-hours";

export function TimeRangeProvider({ children }: { children: React.ReactNode }) {
  const [hours, setHoursState] = useState("24");
  const [now, setNow] = useState(0);
  useEffect(() => { setNow(Date.now()); const query = new URLSearchParams(window.location.search).get("time_hours"); const stored = window.localStorage.getItem(STORAGE_KEY); const initial = isValidHours(query) ? query : stored; if (isValidHours(initial)) setHoursState(initial); const timer = window.setInterval(() => setNow(Date.now()), 60_000); return () => window.clearInterval(timer); }, []);
  function setHours(value: string) { setHoursState(value); window.localStorage.setItem(STORAGE_KEY, value); const url = new URL(window.location.href); url.searchParams.set("time_hours", value); window.history.replaceState(null, "", url); window.dispatchEvent(new CustomEvent("eye-time-range-change", { detail: value })); window.location.reload(); }
  const value = useMemo(() => ({ hours, from: now - Number(hours) * 3600000, to: now, setHours }), [hours, now]);
  return <TimeRangeContext.Provider value={value}>{children}</TimeRangeContext.Provider>;
}

export function useTimeRange() { return useContext(TimeRangeContext) || { hours: "24", from: 0, to: 0, setHours: () => undefined }; }

export function TimeRangeSelector() {
  const { hours, setHours } = useTimeRange();
  const [open, setOpen] = useState(false);
  const [input, setInput] = useState("");
  const [error, setError] = useState("");
  function apply(value: string) {
    const parsed = /^\d+(?:\.\d+)?$/.test(value.trim()) ? Number(value) : parseTimeRange(value);
    if (!parsed || parsed < 1 / 60 || parsed > 2160 * 24) {
      setError("Use a range like: last 4 minutes, last 2 hours, last 7 days, or last month");
      return;
    }
    setError("");
    setOpen(false);
    setInput("");
    setHours(String(parsed));
  }
  return <div className="relative">
    <button type="button" onClick={() => { setOpen((value) => !value); setError(""); }} className="flex w-full items-center justify-between rounded-lg border border-border bg-surface px-3 py-2 text-left text-sm text-fg hover:border-accent">
      <span className="truncate">{formatTimeRange(hours)}</span><span className="ml-2 text-xs text-fg-muted">▾</span>
    </button>
    {open && <div className="absolute left-0 top-0 z-40 w-80 rounded-xl border border-border bg-surface p-3 shadow-xl">
      <div className="mb-2 flex items-center justify-between"><span className="text-xs font-semibold text-fg">Global time range</span><button type="button" onClick={() => setOpen(false)} className="text-xs text-fg-muted">Esc</button></div>
      <input autoFocus value={input} onChange={(event) => { setInput(event.target.value); setError(""); }} onKeyDown={(event) => { if (event.key === "Enter") apply(input); if (event.key === "Escape") setOpen(false); }} placeholder="last 4 minutes" className="w-full rounded-lg border border-border bg-surface px-3 py-2 text-sm text-fg outline-none focus:border-accent focus:ring-2 focus:ring-accent/20" />
      <p className="mt-2 text-[11px] text-fg-muted">Examples: last 1m · last 4 minutes · last 2h · last month · now-30m</p>
      {error && <p className="mt-2 text-[11px] text-danger">{error}</p>}
      <div className="mt-3 grid grid-cols-2 gap-1">{TIME_RANGES.map((range) => <button key={range.value} type="button" onClick={() => apply(range.value)} className="rounded-md px-2 py-1.5 text-left text-xs text-fg-muted hover:bg-surface-2 hover:text-accent">{range.label}</button>)}</div>
      <button type="button" onClick={() => apply(input)} disabled={!input.trim()} className="mt-3 w-full rounded-lg bg-accent px-3 py-2 text-xs font-medium text-accent-fg disabled:opacity-50">Apply range</button>
    </div>}
  </div>;
}
