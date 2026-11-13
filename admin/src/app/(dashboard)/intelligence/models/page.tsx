"use client";

import { useEffect, useState } from "react";
import { api, Tenant } from "@/lib/api";
import { useTenant } from "@/components/TenantContext";
import { Card, CardHeader } from "@/components/Card";
import { Button } from "@/components/Button";
import { Combobox } from "@/components/Combobox";

export default function ModelRegistryPage() {
  const { tenantId: globalTenant } = useTenant();
  const [tenant, setTenant] = useState(globalTenant),
    [tenants, setTenants] = useState<Tenant[]>([]),
    [model, setModel] = useState<{
      scope: string;
      manifest?: {
        name: string;
        version: string;
        task: string;
        algorithm: string;
        source: string;
        created_at: string;
      };
    } | null>(null),
    [history, setHistory] = useState<
      Awaited<ReturnType<typeof api.intelligence.models>>
    >([]),
    [evaluations, setEvaluations] = useState<
      Awaited<ReturnType<typeof api.intelligence.evaluations>>
    >([]),
    [alerts, setAlerts] = useState<
      Awaited<ReturnType<typeof api.intelligence.alerts>>
    >([]),
    [message, setMessage] = useState(""),
    [busy, setBusy] = useState(false);
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
    if (tenant) {
      api.intelligence
        .alerts(tenant)
        .then(setAlerts)
        .catch(() => setAlerts([]));
      api.intelligence
        .modelLayers(tenant, "behavior")
        .then(setHistory)
        .catch(() => setHistory([]));
      api.intelligence
        .evaluations(tenant, "behavior")
        .then(setEvaluations)
        .catch(() => setEvaluations([]));
    } else {
      setAlerts([]);
      setHistory([]);
      setEvaluations([]);
    }
  }, [tenant]);
  async function inspect() {
    if (!tenant) return;
    setBusy(true);
    setMessage("");
    try {
      setModel(await api.intelligence.model(tenant, "behavior"));
    } catch {
      setModel(null);
      setMessage("No behavior model is installed for this tenant or globally.");
    } finally {
      setBusy(false);
    }
  }
  async function evaluate() {
    if (!tenant) return;
    setBusy(true);
    try {
      const result = await api.intelligence.evaluate(tenant, "behavior");
      setMessage(
        `Evaluation: ${result.status}; drift ${(result.drift_score * 100).toFixed(1)}%; calibration error ${(result.calibration_error * 100).toFixed(1)}%; model v${result.model_version}.`,
      );
    } catch {
      setMessage("Train a behavior model before evaluating it.");
    } finally {
      setBusy(false);
    }
  }
  async function promote() {
    if (!tenant) return;
    setBusy(true);
    try {
      await api.intelligence.promote(tenant, "behavior");
      setMessage("Evaluated model promoted safely.");
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "Promotion failed");
    } finally {
      setBusy(false);
    }
  }
  async function rollback() {
    if (!tenant) return;
    setBusy(true);
    try {
      await api.intelligence.rollback(tenant, "behavior");
      setMessage("Model rolled back to the previous promoted version.");
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "Rollback failed");
    } finally {
      setBusy(false);
    }
  }
  async function train() {
    if (!tenant) return;
    setBusy(true);
    setMessage("");
    try {
      await api.intelligence.refresh(tenant);
      await api.intelligence.train(tenant, "auto");
      setMessage(
        "Global baseline and tenant behavior layer trained and applied.",
      );
      await inspect();
    } catch {
      setMessage(
        "Collect tenant activity first, then run automatic training again.",
      );
    } finally {
      setBusy(false);
    }
  }
  async function exportModel() {
    if (!tenant) return;
    setBusy(true);
    setMessage("");
    try {
      const result = await api.intelligence.exportModel(
        tenant,
        "behavior",
        "global",
      );
      setMessage(
        `Exported behavior version ${result.version} with checksum ${result.sha256.slice(0, 16)}…`,
      );
    } catch {
      setMessage("Export failed. Train the behavior model first.");
    } finally {
      setBusy(false);
    }
  }
  return (
    <div>
      <div className="flex flex-wrap items-end justify-between gap-4">
        <div>
          <h1 className="text-xl font-semibold text-fg">Model registry</h1>
          <p className="mt-1 text-sm text-fg-muted">
            Inspect the global behavioral baseline and tenant adaptation layer.
          </p>
        </div>
        <Combobox
          value={tenant}
          onChange={setTenant}
          options={tenants.map((item) => ({
            value: item.ID,
            label: item.Name,
          }))}
          placeholder="Select tenant"
        />
      </div>
      <Card className="mt-6">
        <CardHeader title="Layered behavior model" />
        <p className="px-5 text-sm text-fg-muted">
          The global model learns across all tenants. The tenant layer refines
          it when local activity is available. Train, evaluate, promote, and
          roll back safely.
        </p>
        <div className="flex flex-wrap gap-2 p-5">
          <Button
            onClick={() => void inspect()}
            loading={busy}
            disabled={!tenant}
          >
            Inspect behavior model
          </Button>
          <Button
            variant="secondary"
            onClick={() => void train()}
            loading={busy}
            disabled={!tenant}
          >
            Train automatically
          </Button>
          <Button
            variant="secondary"
            onClick={() => void evaluate()}
            loading={busy}
            disabled={!tenant}
          >
            Evaluate and check drift
          </Button>
          <Button
            variant="secondary"
            onClick={() => void promote()}
            loading={busy}
            disabled={!tenant}
          >
            Promote evaluated model
          </Button>
          <Button
            variant="secondary"
            onClick={() => void rollback()}
            loading={busy}
            disabled={!tenant}
          >
            Rollback
          </Button>
          <Button
            variant="secondary"
            onClick={() => void exportModel()}
            loading={busy}
            disabled={!tenant}
          >
            Export behavior model
          </Button>
        </div>
        {message && (
          <p className="border-t border-border px-5 py-3 text-sm text-fg-muted">
            {message}
          </p>
        )}
        {model?.manifest && (
          <div className="grid gap-4 border-t border-border p-5 text-sm sm:grid-cols-2 lg:grid-cols-4">
            <div>
              <p className="text-xs text-fg-muted">Scope</p>
              <p className="mt-1 font-medium">{model.scope}</p>
            </div>
            <div>
              <p className="text-xs text-fg-muted">Version</p>
              <p className="mt-1 font-medium">{model.manifest.version}</p>
            </div>
            <div>
              <p className="text-xs text-fg-muted">Algorithm</p>
              <p className="mt-1 font-medium">{model.manifest.algorithm}</p>
            </div>
            <div>
              <p className="text-xs text-fg-muted">Source</p>
              <p className="mt-1 font-medium">{model.manifest.source}</p>
            </div>
          </div>
        )}
      </Card>
      {tenant && (
        <>
          <Card className="mt-6">
            <CardHeader
              title="Model history"
              action={
                <span className="text-xs text-fg-muted">
                  {history.length} versions
                </span>
              }
            />
            <div className="overflow-x-auto">
              <table className="w-full text-left text-sm">
                <thead>
                  <tr className="border-b border-border text-xs uppercase text-fg-muted">
                    <th className="px-5 py-3">Scope</th>
                    <th className="px-5 py-3">Version</th>
                    <th className="px-5 py-3">Algorithm</th>
                    <th className="px-5 py-3">Status</th>
                    <th className="px-5 py-3">Created</th>
                    <th className="px-5 py-3">Promoted</th>
                  </tr>
                </thead>
                <tbody>
                  {history.map((item) => (
                    <tr
                      key={item.ID}
                      className="border-b border-border last:border-0"
                    >
                      <td className="px-5 py-3">{item.TenantID ? "Tenant" : "Global"}</td>
                      <td className="px-5 py-3 font-medium">v{item.Version}</td>
                      <td className="px-5 py-3">{item.Algorithm}</td>
                      <td className="px-5 py-3">{item.Status}</td>
                      <td className="px-5 py-3 text-xs text-fg-muted">
                        {new Date(item.CreatedAt).toLocaleString()}
                      </td>
                      <td className="px-5 py-3 text-xs text-fg-muted">
                        {item.PromotedAt
                          ? new Date(item.PromotedAt).toLocaleString()
                          : "—"}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
              {!history.length && (
                <p className="p-5 text-sm text-fg-muted">
                  No behavior model versions have been trained yet.
                </p>
              )}
            </div>
          </Card>
          <Card className="mt-6">
            <CardHeader
              title="Evaluation history"
              action={
                <span className="text-xs text-fg-muted">
                  {evaluations.length} evaluations
                </span>
              }
            />
            <div className="overflow-x-auto">
              <table className="w-full text-left text-sm">
                <thead>
                  <tr className="border-b border-border text-xs uppercase text-fg-muted">
                    <th className="px-5 py-3">Model</th>
                    <th className="px-5 py-3">Status</th>
                    <th className="px-5 py-3">Drift</th>
                    <th className="px-5 py-3">Calibration</th>
                    <th className="px-5 py-3">F1</th>
                    <th className="px-5 py-3">Evaluated</th>
                  </tr>
                </thead>
                <tbody>
                  {evaluations.map((item) => (
                    <tr
                      key={item.ID}
                      className="border-b border-border last:border-0"
                    >
                      <td className="px-5 py-3 font-medium">
                        v{item.ModelVersion}
                      </td>
                      <td className="px-5 py-3">{item.Status}</td>
                      <td className="px-5 py-3">
                        {(item.DriftScore * 100).toFixed(1)}%
                      </td>
                      <td className="px-5 py-3">
                        {(item.CalibrationError * 100).toFixed(1)}%
                      </td>
                      <td className="px-5 py-3">
                        {(item.F1 * 100).toFixed(1)}%
                      </td>
                      <td className="px-5 py-3 text-xs text-fg-muted">
                        {new Date(item.CreatedAt).toLocaleString()}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
              {!evaluations.length && (
                <p className="p-5 text-sm text-fg-muted">
                  No evaluations recorded yet.
                </p>
              )}
            </div>
          </Card>
          <Card className="mt-6">
            <CardHeader
              title="Open quality alerts"
              action={
                <span className="text-xs text-fg-muted">
                  {alerts.length} open
                </span>
              }
            />
            <div className="divide-y divide-border">
              {alerts.map((alert) => (
                <div
                  key={alert.ID}
                  className="flex flex-wrap items-center justify-between gap-3 px-5 py-4 text-sm"
                >
                  <div>
                    <div className="flex items-center gap-2 font-medium">
                      <span className="rounded bg-danger/10 px-2 py-0.5 text-xs uppercase text-danger">
                        {alert.Severity}
                      </span>
                      {alert.Kind}
                    </div>
                    <p className="mt-1 text-fg-muted">{alert.Message}</p>
                    <p className="mt-1 text-xs text-fg-muted">
                      {alert.Label} ·{" "}
                      {new Date(alert.CreatedAt).toLocaleString()}
                    </p>
                  </div>
                  <Button
                    variant="secondary"
                    onClick={() =>
                      void api.intelligence
                        .resolveAlert(tenant, alert.ID)
                        .then(() =>
                          setAlerts((current) =>
                            current.filter((item) => item.ID !== alert.ID),
                          ),
                        )
                    }
                  >
                    Resolve
                  </Button>
                </div>
              ))}
              {!alerts.length && (
                <p className="p-5 text-sm text-fg-muted">
                  No open model-quality alerts.
                </p>
              )}
            </div>
          </Card>
        </>
      )}
    </div>
  );
}
