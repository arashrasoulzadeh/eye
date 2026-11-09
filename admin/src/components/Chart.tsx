"use client";
import { Bar, BarChart, CartesianGrid, Line, LineChart, ResponsiveContainer, Tooltip, XAxis, YAxis } from "recharts";

export type ChartPoint = { label: string; value: number };

const AXIS_COLOR = "var(--color-fg-muted, #888)";
const LINE_COLOR = "var(--color-accent, #6366f1)";

/** Small reusable line chart over {label, value} points — no legend/series config, matches the app's single-series usage (events over time, etc). */
export function LineTrend({ data, height = 220 }: { data: ChartPoint[]; height?: number }) {
  if (data.length === 0) {
    return <div className="flex h-[220px] items-center justify-center text-sm text-fg-muted">No data yet.</div>;
  }
  return (
    <ResponsiveContainer width="100%" height={height}>
      <LineChart data={data} margin={{ top: 8, right: 16, left: 0, bottom: 0 }}>
        <CartesianGrid strokeDasharray="3 3" stroke="var(--color-border, #e5e7eb)" />
        <XAxis dataKey="label" tick={{ fontSize: 11, fill: AXIS_COLOR }} tickLine={false} axisLine={false} />
        <YAxis tick={{ fontSize: 11, fill: AXIS_COLOR }} tickLine={false} axisLine={false} allowDecimals={false} />
        <Tooltip />
        <Line type="monotone" dataKey="value" stroke={LINE_COLOR} strokeWidth={2} dot={false} />
      </LineChart>
    </ResponsiveContainer>
  );
}

/** Small reusable bar chart over {label, value} points — e.g. events per tenant. */
export function BarBreakdown({ data, height = 220 }: { data: ChartPoint[]; height?: number }) {
  if (data.length === 0) {
    return <div className="flex h-[220px] items-center justify-center text-sm text-fg-muted">No data yet.</div>;
  }
  return (
    <ResponsiveContainer width="100%" height={height}>
      <BarChart data={data} margin={{ top: 8, right: 16, left: 0, bottom: 0 }}>
        <CartesianGrid strokeDasharray="3 3" stroke="var(--color-border, #e5e7eb)" />
        <XAxis dataKey="label" tick={{ fontSize: 11, fill: AXIS_COLOR }} tickLine={false} axisLine={false} />
        <YAxis tick={{ fontSize: 11, fill: AXIS_COLOR }} tickLine={false} axisLine={false} allowDecimals={false} />
        <Tooltip />
        <Bar dataKey="value" fill={LINE_COLOR} radius={[4, 4, 0, 0]} />
      </BarChart>
    </ResponsiveContainer>
  );
}
