"use client";

import { useTimeRange } from "@/components/TimeRangeContext";
import { useEffect, useState } from "react";
import { useTenant } from "@/components/TenantContext";
import { Button } from "@/components/Button";
import { Card, CardHeader } from "@/components/Card";
import { translate, useLanguage } from "@/components/LanguageContext";

const url = "/api/proxy/intelligence/users";

type ModelStatus = {
  installed: string;
  latest: { version: string; created_at: string } | null;
};

async function requestModel(action: string, method: "GET" | "POST", signal?: AbortSignal): Promise<ModelStatus> {
  const response = await fetch(`${url}?action=${action}`, { method, signal });
  if (!response.ok) throw new Error();
  return response.json();
}

type Run = {
  ID: number;
  TenantID: string;
  Version: string;
  Events: number;
  Sessions: number;
  Users: number;
  Error: string;
  CreatedAt: string;
};

export default function IntelligencePage() {
  const { tenantId, tenants, loading: tenantLoading } = useTenant();
  const { language } = useLanguage();
  const [log, setLog] = useState<{ scope: string; runs: Run[]; failed: boolean }>();
  const scope = tenantId || "";
  const { from, to } = useTimeRange();
  const t = (key: string) => translate(key, language);
  const [model, setModel] = useState<ModelStatus>();
  const [modelError, setModelError] = useState("");
  const [downloading, setDownloading] = useState(false);
  useEffect(() => {
    const controller = new AbortController();
    const check = () => requestModel("model-status", "GET", controller.signal)
      .then((status) => {
        if (!controller.signal.aborted) setModel(status);
      })
      .catch(() => {
        if (!controller.signal.aborted) setModelError("Could not check model releases.");
      });
    if (downloading) return;
    void check();
    const timer = window.setInterval(check, 60000);
    return () => {
      controller.abort();
      window.clearInterval(timer);
    };
  }, [downloading]);
  async function downloadModel() {
    setDownloading(true);
    setModelError("");
    try {
      setModel(await requestModel("download-model", "POST"));
    } catch {
      setModelError("Could not download the model.");
    } finally {
      setDownloading(false);
    }
  }
  useEffect(() => {
    if (tenantLoading || !to) return;
    const controller = new AbortController();
    const params = new URLSearchParams({ tenant_id: scope, action: "inference-log", from_ts: String(from), to_ts: String(to) });
    const load = async () => {
      try {
        const response = await fetch(`${url}?${params}`, { signal: controller.signal });
        if (!response.ok) throw new Error();
        const data: unknown = await response.json();
        if (!controller.signal.aborted) setLog({ scope, runs: Array.isArray(data) ? data : [], failed: false });
      } catch {
        if (!controller.signal.aborted) setLog({ scope, runs: [], failed: true });
      }
    };
    void load();
    const timer = window.setInterval(load, 15000);
    return () => {
      controller.abort();
      window.clearInterval(timer);
    };
  }, [scope, tenantLoading, from, to]);
  const current = log?.scope === scope ? log : undefined;
  const runs = current?.runs ?? [];
  const version = runs.find((run) => !run.Error)?.Version;
  return (
    <div className="space-y-5">
      <header className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <h1 className="text-xl font-semibold text-fg">{t("Intelligence")}</h1>
          <p className="mt-1 text-sm text-fg-muted">{t("Download a training batch to share privately. Inference runs locally with the released model.")}</p>
        </div>
        <div className="space-y-1">
          <form action={url} method="get">
            <input type="hidden" name="action" value="training-csv" />
            <input type="hidden" name="tenant_id" value={scope} />
            <input type="hidden" name="from_ts" value={from} />
            <input type="hidden" name="to_ts" value={to} />
            <Button type="submit" disabled={tenantLoading || !to}>{t("Download training CSV")}</Button>
          </form>
          <p className="text-xs text-fg-muted">
            {tenantLoading ? t("Loading tenants…") : tenants.find((tenant) => tenant.ID === scope)?.Name || t("All tenants")}
          </p>
        </div>
      </header>
      <Card>
        <CardHeader title={t("Global model")} action={
          <Button type="button" loading={downloading} onClick={() => void downloadModel()}>
            {t("Download latest model")}
          </Button>
        } />
        <div className="space-y-1 px-5 pb-5 text-sm">
          <p>{t("Installed model")} · {model?.installed || "—"}</p>
          {model?.latest ? (
            <>
              <p>{t("Latest release")} · {model.latest.version}</p>
              <p>{t("Created")} · {new Date(model.latest.created_at).toLocaleString(language === "fa" ? "fa-IR" : undefined)}</p>
              <p className="font-medium">{t(model.latest.version === model.installed ? "Up to date" : "New model available")}</p>
            </>
          ) : model && <p>{t("No model release available.")}</p>}
          {modelError && <p role="alert" className="text-danger">{t(modelError)}</p>}
        </div>
      </Card>
      <Card>
        <CardHeader title={t("Last successful model")} />
        <p className="px-5 pb-5 text-sm">{version || t("No successful inference yet.")}</p>
      </Card>
      <Card>
        <CardHeader title={t("Inference log")} />
        {!runs.length && (
          <p className="px-5 pb-5 text-sm text-fg-muted">
            {t(!current ? "Loading inference log…" : current.failed ? "Could not load inference log." : "No inference runs yet.")}
          </p>
        )}
        {!!runs.length && (
          <div className="overflow-x-auto">
            <table className="w-full text-start text-sm">
              <thead><tr className="border-b border-border text-fg-muted">
                {["Time", "Tenant", "Model version", "Events", "Sessions", "Users", "Status"].map((key) => (
                  <th key={key} scope="col" className="px-4 py-3 text-start">{t(key)}</th>
                ))}
              </tr></thead>
              <tbody>
                {runs.map((run) => (
                  <tr key={run.ID} className="border-b border-border last:border-0">
                    <td className="px-4 py-3">{new Date(run.CreatedAt).toLocaleString(language === "fa" ? "fa-IR" : undefined)}</td>
                    <td className="px-4 py-3">{tenants.find((tenant) => tenant.ID === run.TenantID)?.Name || run.TenantID}</td>
                    <td className="px-4 py-3">{run.Version}</td>
                    <td className="px-4 py-3">{run.Events}</td>
                    <td className="px-4 py-3">{run.Sessions}</td>
                    <td className="px-4 py-3">{run.Users}</td>
                    <td className="px-4 py-3" title={run.Error || undefined}>{t(run.Error ? "Inference failed" : "Inference completed")}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </Card>
    </div>
  );
}
