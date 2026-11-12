"use client";
import { Area, AreaChart, Bar, BarChart, CartesianGrid, Line, LineChart, Pie, PieChart, Cell, Legend, ResponsiveContainer, Tooltip, XAxis, YAxis } from "recharts";

export type ChartPoint = { label: string; value: number };

const AXIS_COLOR = "var(--color-fg-muted, #888)";
const LINE_COLOR = "var(--color-accent, #6366f1)";

/** Small reusable line chart over {label, value} points — no legend/series config, matches the app's single-series usage (events over time, etc). */
export function LineTrend({ data, height = 220, legend = "Events" }: { data: ChartPoint[]; height?: number; legend?: string }) {
  if (!data || data.length === 0) {
    return <div className="flex h-[220px] items-center justify-center text-sm text-fg-muted">No data yet.</div>;
  }
  return (
    <ResponsiveContainer width="100%" height={height}>
      <LineChart data={data} margin={{ top: 8, right: 16, left: 0, bottom: 0 }}>
        <CartesianGrid strokeDasharray="3 3" stroke="var(--color-border, #e5e7eb)" />
        <XAxis dataKey="label" tick={{ fontSize: 11, fill: AXIS_COLOR }} tickLine={false} axisLine={false} />
        <YAxis tick={{ fontSize: 11, fill: AXIS_COLOR }} tickLine={false} axisLine={false} allowDecimals={false} />
        <Legend wrapperStyle={{ color: AXIS_COLOR, fontSize: 12 }} />
        <Tooltip contentStyle={{ backgroundColor: "var(--surface)", border: "1px solid var(--border)", borderRadius: 8, color: "var(--fg)" }} labelStyle={{ color: "var(--fg-muted)" }} itemStyle={{ color: "var(--accent)" }} />
        <Line name={legend} type="monotone" dataKey="value" stroke={LINE_COLOR} strokeWidth={2} dot={false} />
      </LineChart>
    </ResponsiveContainer>
  );
}

/** Small reusable bar chart over {label, value} points — e.g. events per tenant. */
export function BarBreakdown({ data, height = 220, legend = "Events" }: { data: ChartPoint[]; height?: number; legend?: string }) {
  if (!data || data.length === 0) {
    return <div className="flex h-[220px] items-center justify-center text-sm text-fg-muted">No data yet.</div>;
  }
  return (
    <ResponsiveContainer width="100%" height={height}>
      <BarChart data={data} margin={{ top: 8, right: 16, left: 0, bottom: 0 }}>
        <CartesianGrid strokeDasharray="3 3" stroke="var(--color-border, #e5e7eb)" />
        <XAxis dataKey="label" tick={{ fontSize: 11, fill: AXIS_COLOR }} tickLine={false} axisLine={false} />
        <YAxis tick={{ fontSize: 11, fill: AXIS_COLOR }} tickLine={false} axisLine={false} allowDecimals={false} />
        <Legend wrapperStyle={{ color: AXIS_COLOR, fontSize: 12 }} />
        <Tooltip contentStyle={{ backgroundColor: "var(--surface)", border: "1px solid var(--border)", borderRadius: 8, color: "var(--fg)" }} labelStyle={{ color: "var(--fg-muted)" }} itemStyle={{ color: "var(--accent)" }} />
        <Bar name={legend} dataKey="value" fill={LINE_COLOR} radius={[4, 4, 0, 0]} />
      </BarChart>
    </ResponsiveContainer>
  );
}

export function AreaTrend({ data, height = 320, legend = "Events" }: { data: ChartPoint[]; height?: number; legend?: string }) {
  return <div className="h-full w-full"><ResponsiveContainer width="100%" height={height}><AreaChart data={data}><CartesianGrid strokeDasharray="3 3" stroke="var(--border)"/><XAxis dataKey="label"/><YAxis allowDecimals={false}/><Legend wrapperStyle={{ color: "var(--color-fg-muted, #888)", fontSize: 12 }}/><Tooltip contentStyle={{ backgroundColor: "var(--surface)", border: "1px solid var(--border)", borderRadius: 8, color: "var(--fg)" }} labelStyle={{ color: "var(--fg-muted)" }} itemStyle={{ color: "var(--accent)" }}/><Area name={legend} type="monotone" dataKey="value" stroke="var(--accent)" fill="var(--accent)" fillOpacity={0.2}/></AreaChart></ResponsiveContainer></div>;
}

export function PieBreakdown({ data, height = 320, legend = "Events" }: { data: ChartPoint[]; height?: number; legend?: string }) {
  const colors = ["#6366f1", "#22c55e", "#f59e0b", "#ef4444", "#06b6d4", "#a855f7"];
  return <div className="h-full w-full"><ResponsiveContainer width="100%" height={height}><PieChart><Legend formatter={() => legend} wrapperStyle={{ color: "var(--color-fg-muted, #888)", fontSize: 12 }}/><Tooltip contentStyle={{ backgroundColor: "var(--surface)", border: "1px solid var(--border)", borderRadius: 8, color: "var(--fg)" }} labelStyle={{ color: "var(--fg-muted)" }} itemStyle={{ color: "var(--fg)" }}/><Pie data={data} dataKey="value" nameKey="label" cx="50%" cy="50%" outerRadius="70%" label>{data.map((_, i) => <Cell key={i} fill={colors[i % colors.length]}/>)}</Pie></PieChart></ResponsiveContainer></div>;
}
