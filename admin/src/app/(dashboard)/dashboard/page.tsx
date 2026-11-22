"use client";
import { useTenant } from "@/components/TenantContext";
import { useCallback, useEffect, useRef, useState } from "react";
import { api, ApiError, DashboardCache } from "@/lib/api";
import { StatCard, Card, CardHeader } from "@/components/Card";
import { LineTrend, BarBreakdown } from "@/components/Chart";
import { PinnedFunnelCard } from "@/components/PinnedFunnelCard";
import { previousPeriod } from "@/lib/funnel";
import { DashboardEvents } from "@/components/DashboardEvents";
import { translate, useLanguage } from "@/components/LanguageContext";
import { Button } from "@/components/Button";
import Link from "next/link";
import { formatTimeRange, useTimeRange } from "@/components/TimeRangeContext";

export default function Dashboard() {
  const { language } = useLanguage();
  const t = (text: string) => translate(text, language);
  const [snapshot, setSnapshot] = useState<{ tenantId: string; hours: string; cache: DashboardCache; previous: DashboardCache | null }>();
  const request = useRef(0);
  const { from: globalFrom, to: globalTo, hours } = useTimeRange();
  const [error, setError] = useState(""), [loading, setLoading] = useState(true);
  const timeRangeLabel = formatTimeRange(hours);
  const localizedRange = formatTimeRange(hours, language);
  const { tenantId, tenants, loading: scopeLoading } = useTenant();
  const tenantLabel = tenantId ? tenants.find((tenant) => tenant.ID === tenantId)?.Name || tenantId : t("All tenants");
  const current = snapshot?.tenantId === tenantId && snapshot.hours === hours ? snapshot : undefined;
  const cache = current?.cache;
  const previous = current?.previous;
  const changeFormat = new Intl.NumberFormat(language, { style: "percent", signDisplay: "exceptZero", maximumFractionDigits: 1 });
  function change(value: number | undefined, prior: number | undefined) {
    if (value === undefined || !prior) return "—";
    return changeFormat.format((value - prior) / prior);
  }

  const load = useCallback(async () => {
    if (scopeLoading || !globalFrom || !globalTo) return;
    const id = ++request.current;
    setLoading(true);
    setError("");
    try {
      const [data, prior] = await Promise.all([
        api.dashboardCache({ tenant_id: tenantId, from_ts: globalFrom, to_ts: globalTo }),
        api.dashboardCache({ tenant_id: tenantId, ...previousPeriod(globalFrom, globalTo) }).catch(() => null),
      ]);
      if (request.current === id) setSnapshot({ tenantId, hours, cache: data, previous: prior });
    } catch (e) {
      if (request.current === id) setError(e instanceof ApiError ? e.message : "Failed to load dashboard metrics");
    } finally {
      if (request.current === id) setLoading(false);
    }
  }, [globalFrom, globalTo, tenantId, scopeLoading, hours]);

  useEffect(() => {
    void load();
    return () => { request.current++; };
  }, [load]);
  const hourlyAverage = cache?.events_by_hour?.length ? cache.events_by_hour.reduce((sum, point) => sum + point.value, 0) / cache.events_by_hour.length : 0;
  const cachedAtMs = cache ? (cache.cached_at < 100000000000 ? cache.cached_at * 1000 : cache.cached_at) : 0;
  const eventsPerMinute = hourlyAverage / 60;
  const eventsPerSecond = eventsPerMinute / 60;
  function downloadReport() {
    if (!cache) return;
    const rows = [["Metric", "Value"], ["Events", String(cache.events)], ["Sessions", String(cache.sessions)], ["Users", String(cache.users ?? cache.sessions)], ["Online users", String(cache.online_users ?? 0)], ["Retention", `${((cache.retention_rate ?? 0) * 100).toFixed(1)}%`], ["Tenants", String(cache.tenants)], ["Events per hour", String(Math.round(hourlyAverage))], ["Time range hours", hours]];
    const csv = rows.map((row) => row.map((value) => `"${value.replaceAll('"', '""')}"`).join(",")).join("\n");
    const url = URL.createObjectURL(new Blob([csv], { type: "text/csv;charset=utf-8" }));
    const link = document.createElement("a"); link.href = url; link.download = `eye-dashboard-${timeRangeLabel.toLowerCase().replaceAll(" ", "-")}.csv`; link.click(); URL.revokeObjectURL(url);
  }

  return (
    <div>
      <header className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
        <div className="min-w-0">
          <h1 className="text-xl font-semibold text-fg">{t("Dashboard")}</h1>
          <ul className="mt-2 flex flex-wrap gap-2 text-sm">
            <li className="max-w-full rounded-md border border-border bg-surface px-2.5 py-1">
              <span className="text-fg-muted">{t("Tenant scope")}</span>{" "}
              <bdi className="break-words font-medium text-fg">{scopeLoading ? "…" : tenantLabel}</bdi>
            </li>
            <li className="rounded-md border border-border bg-surface px-2.5 py-1">
              <span className="text-fg-muted">{t("Global time range")}</span>{" "}
              <bdi className="font-medium text-fg">{localizedRange}</bdi>
            </li>
          </ul>
        </div>
        <div className="flex shrink-0 flex-wrap gap-2">
          <Button variant="secondary" onClick={() => window.print()}>{t("Print report")}</Button>
          <Button variant="secondary" onClick={downloadReport} disabled={!cache}>{t("Download CSV")}</Button>
          <Button variant="secondary" onClick={() => void load()} loading={loading}>{t("Refresh")}</Button>
        </div>
      </header>

      {error && <p role="alert" className="mt-4 rounded border border-danger/30 bg-danger/5 p-3 text-sm text-danger">{t(error)}</p>}

      <section aria-label={t("Activity overview")} className="mt-6 grid grid-cols-1 gap-4 sm:grid-cols-2">
        <StatCard label={t("Events")} value={cache ? cache.events.toLocaleString(language) : "…"} hint={cache && `${t("Change vs previous period")} ${change(cache.events, previous?.events)}`} />
        <StatCard label={t("Sessions")} value={cache ? cache.sessions.toLocaleString(language) : "…"} hint={cache && `${t("Change vs previous period")} ${change(cache.sessions, previous?.sessions)}`} />
      </section>

      {!scopeLoading && !!tenantId && !!globalFrom && !!globalTo && (
        <PinnedFunnelCard key={tenantId} tenantId={tenantId} from={globalFrom} to={globalTo} />
      )}

      {!scopeLoading && !!globalFrom && !!globalTo && (
        <DashboardEvents tenantId={tenantId} from={globalFrom} to={globalTo} />
      )}

      <div className="mt-6 grid grid-cols-1 gap-6 lg:grid-cols-2">
        <Card>
          <CardHeader title="Events by hour (cached)" />
          <div className="p-5">
            <LineTrend data={cache ? cache.events_by_hour : []} />
          </div>
        </Card>
        <Card>
          <CardHeader title="Events by tenant (cached, top 8)" />
          <div className="p-5">
            <BarBreakdown data={cache ? cache.events_by_tenant : []} />
          </div>
        </Card>
      </div>

      <details className="mt-6 rounded-lg border border-border bg-surface px-4 py-3">
        <summary className="cursor-pointer text-sm font-semibold text-fg">{t("Technical metrics")}</summary>
        <div className="mt-4 grid grid-cols-2 gap-3 sm:grid-cols-3">
          <StatCard label={t("Users")} value={cache ? (cache.users ?? cache.sessions).toLocaleString(language) : "…"} hint={t("Estimated from distinct sessions")} />
          <StatCard label={t("Tenants")} value={cache ? cache.tenants.toLocaleString(language) : "…"} />
          <StatCard label={t("Online users")} value={cache ? (cache.online_users ?? 0).toLocaleString(language) : "…"} hint={t("All tenants · Last 5 minutes")} />
          <StatCard label={t("Retention")} value={cache ? (cache.retention_rate ?? 0).toLocaleString(language, { style: "percent", maximumFractionDigits: 1 }) : "…"} hint={t("All tenants · 7-day return rate")} />
          <StatCard label={t("Events / hour")} value={cache ? Math.round(hourlyAverage).toLocaleString(language) : "…"} />
          <StatCard label={t("Events / minute")} value={cache ? eventsPerMinute.toLocaleString(language, { maximumFractionDigits: 1 }) : "…"} />
          <StatCard label={t("Events / second")} value={cache ? eventsPerSecond.toLocaleString(language, { maximumFractionDigits: 2 }) : "…"} />
          <StatCard label={t("Hourly buckets")} value={cache ? cache.events_by_hour.length.toLocaleString(language) : "…"} />
          <StatCard label={t("Cache age")} value={cache ? Math.max(0, Math.round((Date.now() - cachedAtMs) / 1000)).toLocaleString(language, { style: "unit", unit: "second" }) : "…"} />
        </div>
      </details>

      <Card className="mt-6">
        <CardHeader title="Quick links" />
        <div className="grid grid-cols-2 gap-3 p-5 sm:grid-cols-3">
          {[
            ["Manage tenants", "/tenants"],
            ["Browse events", "/events"],
            ["Funnels", "/analytics/funnel"],
            ["Send a test event", "/test-event"],
            ["Register a schema", "/schemas"],
            ["View sessions", "/sessions"],
          ].map(([label, href]) => (
            <Link
              key={href}
              href={href}
              className="rounded-lg border border-border px-3 py-2.5 text-sm text-fg hover:border-accent hover:text-accent"
            >
              {translate(label, language)}
            </Link>
          ))}
        </div>
      </Card>
      <Card className="mt-6"><CardHeader title="Recommended next steps"/><div className="grid gap-3 p-5 sm:grid-cols-3"><Link href="/health" className="rounded-lg border border-border p-3 text-sm hover:border-accent"><p className="font-medium text-fg">Review system health</p><p className="mt-1 text-xs text-fg-muted">SLO, anomalies, alerts, and dead letters.</p></Link><Link href="/analytics" className="rounded-lg border border-border p-3 text-sm hover:border-accent"><p className="font-medium text-fg">Investigate trends</p><p className="mt-1 text-xs text-fg-muted">Open a shareable query for the current range.</p></Link><Link href="/stream" className="rounded-lg border border-border p-3 text-sm hover:border-accent"><p className="font-medium text-fg">Watch live traffic</p><p className="mt-1 text-xs text-fg-muted">Inspect the latest events and sessions.</p></Link></div></Card>
    </div>
  );
}
