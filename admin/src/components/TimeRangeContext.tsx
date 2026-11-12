"use client";

import { createContext, useContext, useEffect, useMemo, useState } from "react";
import { Combobox } from "@/components/Combobox";

export const TIME_RANGES = [
  { value: "1", label: "Last hour" },
  { value: "24", label: "Last 24 hours" },
  { value: "168", label: "Last 7 days" },
  { value: "720", label: "Last 30 days" },
  { value: "2160", label: "Last 90 days" },
];
type TimeRangeValue = { hours: string; from: number; to: number };
type TimeRangeContextValue = TimeRangeValue & { setHours: (hours: string) => void };
const TimeRangeContext = createContext<TimeRangeContextValue | null>(null);
const STORAGE_KEY = "eye.admin.time-hours";

export function TimeRangeProvider({ children }: { children: React.ReactNode }) {
  const [hours, setHoursState] = useState("24");
  const [now, setNow] = useState(0);
  useEffect(() => { setNow(Date.now()); const stored = window.localStorage.getItem(STORAGE_KEY); if (stored && TIME_RANGES.some((item) => item.value === stored)) setHoursState(stored); const timer = window.setInterval(() => setNow(Date.now()), 60_000); return () => window.clearInterval(timer); }, []);
  function setHours(value: string) { setHoursState(value); window.localStorage.setItem(STORAGE_KEY, value); window.dispatchEvent(new CustomEvent("eye-time-range-change", { detail: value })); window.location.reload(); }
  const value = useMemo(() => ({ hours, from: now - Number(hours) * 3600000, to: now, setHours }), [hours, now]);
  return <TimeRangeContext.Provider value={value}>{children}</TimeRangeContext.Provider>;
}

export function useTimeRange() { return useContext(TimeRangeContext) || { hours: "24", from: 0, to: 0, setHours: () => undefined }; }

export function TimeRangeSelector() { const { hours, setHours } = useTimeRange(); return <Combobox value={hours} onChange={setHours} options={TIME_RANGES} placeholder="Time range" maxVisible={5} />; }
