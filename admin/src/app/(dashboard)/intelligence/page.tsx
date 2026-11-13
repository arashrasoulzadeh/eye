"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { api, ApiError, Tenant } from "@/lib/api";
import { useTenant } from "@/components/TenantContext";
import { Card, CardHeader, StatCard } from "@/components/Card";
import { Badge } from "@/components/Badge";
import { Button } from "@/components/Button";
import { Combobox } from "@/components/Combobox";
import { translate, useLanguage } from "@/components/LanguageContext";

type Snapshot = Awaited<ReturnType<typeof api.intelligence.list>>[number];

export default function IntelligencePage() {
  const { tenantId: globalTenant } = useTenant();
  const { language } = useLanguage();
  const [tenant, setTenant] = useState(globalTenant),
    [tenants, setTenants] = useState<Tenant[]>([]),
    [rows, setRows] = useState<Snapshot[]>([]),
    [pipelineRuns, setPipelineRuns] = useState<
      Awaited<ReturnType<typeof api.intelligence.pipelineRuns>>
    >([]),
    [loading, setLoading] = useState(false),
    [error, setError] = useState("");
  const load = useCallback(async () => {
    if (!tenant) return;
    setLoading(true);
    setError("");
    try {
      setRows(await api.intelligence.list(tenant));
    } catch (e) {
      setError(
        e instanceof ApiError ? e.message : "Failed to load intelligence",
      );
    } finally {
      setLoading(false);
    }
  }, [tenant]);
  useEffect(() => {
    api.tenants
      .list()
      .then((items) => setTenants(Array.isArray(items) ? items : []))
      .catch(() => undefined);
  }, []);
  useEffect(() => {
    if (globalTenant) setTenant(globalTenant);
  }, [globalTenant]);
  useEffect(() => {
    void load();
    const timer = window.setInterval(() => void load(), 60000);
    return () => window.clearInterval(timer);
  }, [load]);
  useEffect(() => {
    if (!tenant) {
      setPipelineRuns([]);
      return;
    }
    const poll = () =>
      api.intelligence
        .pipelineRuns(tenant)
        .then(setPipelineRuns)
        .catch(() => undefined);
    void poll();
    const timer = window.setInterval(poll, 2500);
    return () => window.clearInterval(timer);
  }, [tenant]);
  const tagCounts = useMemo(
    () =>
      rows
        .flatMap((row) => row.tags)
        .reduce(
          (counts, tag) => counts.set(tag, (counts.get(tag) || 0) + 1),
          new Map<string, number>(),
        ),
    [rows],
  );
  const anomalies = rows.filter((row) => row.anomaly_score > 0).length;
  async function refresh() {
    if (!tenant) return;
    await api.intelligence.refresh(tenant);
    await load();
  }
  async function runPipeline() {
    if (!tenant) return;
    setLoading(true);
    setError("");
    try {
      await api.intelligence.runPipeline(tenant);
      await load();
    } catch (e) {
      setError(
        e instanceof ApiError ? e.message : "Intelligence pipeline failed",
      );
    } finally {
      setLoading(false);
    }
  }
  async function retryPipeline() {
    if (!tenant || !pipelineRuns[0] || pipelineRuns[0].Status !== "failed")
      return;
    setLoading(true);
    setError("");
    try {
      await api.intelligence.retryPipeline(tenant, pipelineRuns[0].ID);
    } catch (e) {
      setError(
        e instanceof ApiError
          ? e.message
          : "Could not retry intelligence pipeline",
      );
    } finally {
      setLoading(false);
    }
  }
  return (
    <div>
      <div className="flex flex-wrap items-end justify-between gap-4">
        <div>
          <h1 className="text-xl font-semibold text-fg">
            {translate("Intelligence", language)}
          </h1>
          <p className="mt-1 text-sm text-fg-muted">
            Transparent offline user tags, anomaly scores, and behavior
            predictions.
          </p>
        </div>
        <div className="flex gap-2">
          <Combobox
            value={tenant}
            onChange={setTenant}
            options={tenants.map((item) => ({
              value: item.ID,
              label: item.Name,
            }))}
            placeholder="Select tenant"
          />
          <Button
            onClick={() => void runPipeline()}
            loading={loading}
            disabled={!tenant}
          >
            Run intelligence
          </Button>
          <Button
            onClick={() => void refresh()}
            loading={loading}
            disabled={!tenant}
          >
            Refresh model
          </Button>
        </div>
      </div>
      {pipelineRuns[0] && (
        <Card className="mt-4">
          <div className="flex items-center justify-between gap-4 p-4 text-sm">
            <div>
              <p className="font-medium text-fg">Intelligence pipeline</p>
              <p className="text-xs text-fg-muted">
                {pipelineRuns[0].Stage || pipelineRuns[0].Status}
                {pipelineRuns[0].Error ? ` · ${pipelineRuns[0].Error}` : ""}
              </p>
            </div>
            <div className="flex items-center gap-3">
              <span
                className={
                  pipelineRuns[0].Status === "failed"
                    ? "text-danger"
                    : pipelineRuns[0].Status === "completed"
                      ? "text-success"
                      : "text-accent"
                }
              >
                {pipelineRuns[0].Status} · {pipelineRuns[0].Progress}%
              </span>
              {pipelineRuns[0].Status === "failed" && (
                <Button
                  variant="secondary"
                  onClick={() => void retryPipeline()}
                  disabled={loading}
                >
                  Retry
                </Button>
              )}
            </div>
          </div>
        </Card>
      )}
      {error && (
        <p className="mt-4 rounded border border-danger/30 bg-danger/5 p-3 text-sm text-danger">
          {error}
        </p>
      )}
      {!tenant ? (
        <Card className="mt-6">
          <p className="p-5 text-sm text-fg-muted">
            Select a tenant to inspect intelligence snapshots.
          </p>
        </Card>
      ) : (
        <>
          <div className="mt-6 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
            <StatCard
              label="Profiles"
              value={rows.length.toLocaleString()}
              hint="Latest tenant snapshots"
            />
            <StatCard
              label="Anomalies"
              value={anomalies.toLocaleString()}
              hint="Needs investigation"
            />
            <StatCard
              label="Buyers"
              value={(tagCounts.get("buyer") || 0).toLocaleString()}
              hint="Purchase signals"
            />
            <StatCard
              label="High intent"
              value={(tagCounts.get("high_intent") || 0).toLocaleString()}
              hint="Strong activity signals"
            />
          </div>
          <Card className="mt-6">
            <CardHeader
              title="User intelligence"
              action={
                <span className="text-xs text-fg-muted">
                  {loading ? "Refreshing…" : "Auto-refresh: 1 minute"}
                </span>
              }
            />
            <div className="overflow-x-auto">
              <table className="w-full text-left text-sm">
                <thead>
                  <tr className="border-b border-border text-xs uppercase text-fg-muted">
                    <th className="px-5 py-3">Tags</th>
                    <th className="px-5 py-3">IP / agent</th>
                    <th className="px-5 py-3">Next event</th>
                    <th className="px-5 py-3">Today forecast</th>
                    <th className="px-5 py-3">Anomaly</th>
                    <th className="px-5 py-3">Confidence</th>
                  </tr>
                </thead>
                <tbody>
                  {rows.map((row) => (
                    <tr
                      key={row.fingerprint}
                      className="border-b border-border last:border-0"
                    >
                      <td className="px-5 py-3">
                        <div className="flex max-w-xs flex-wrap gap-1">
                          {row.tags.map((tag) => (
                            <Badge
                              key={tag}
                              tone={
                                tag === "anomalous"
                                  ? "warning"
                                  : tag === "buyer"
                                    ? "success"
                                    : "neutral"
                              }
                            >
                              {tag}
                            </Badge>
                          ))}
                        </div>
                      </td>
                      <td className="px-5 py-3">
                        <div className="font-mono text-xs">
                          {row.ip || "Unknown"}
                        </div>
                        <div className="max-w-xs truncate text-xs text-fg-muted">
                          {row.user_agent || "Unknown"}
                        </div>
                      </td>
                      <td className="px-5 py-3">
                        {row.predicted_next_event || "—"}
                      </td>
                      <td className="px-5 py-3">
                        {row.predicted_events_today.toLocaleString()}
                      </td>
                      <td className="px-5 py-3">
                        {(row.anomaly_score * 100).toFixed(0)}%
                      </td>
                      <td className="px-5 py-3">
                        {(row.confidence * 100).toFixed(0)}%
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
              {!rows.length && !loading && (
                <p className="p-5 text-sm text-fg-muted">
                  No snapshots yet. Refresh the model after this tenant receives
                  sessions.
                </p>
              )}
            </div>
          </Card>
        </>
      )}
    </div>
  );
}
