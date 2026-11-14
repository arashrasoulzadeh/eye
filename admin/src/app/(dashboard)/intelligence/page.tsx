"use client";

import { useEffect, useState } from "react";
import { useTenant } from "@/components/TenantContext";
import { Card, CardHeader } from "@/components/Card";
import { translate, useLanguage } from "@/components/LanguageContext";

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
  const t = (key: string) => translate(key, language);
  const url = "/api/proxy/intelligence/users";
  const query = new URLSearchParams({ tenant_id: scope, action: "training-csv" });
  useEffect(() => {
    if (tenantLoading) return;
    const controller = new AbortController();
    const params = new URLSearchParams({ tenant_id: scope, action: "inference-log" });
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
  }, [scope, tenantLoading]);
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
        {!tenantLoading && <a href={`${url}?${query}`} download="eye-training.csv" className="rounded bg-accent px-4 py-2 text-sm font-medium text-white">
          {t("Download training CSV")}
        </a>}
      </header>
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
