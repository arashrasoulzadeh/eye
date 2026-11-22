"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { api } from "@/lib/api";
import { formatEventLabel, funnelSummary, previousPeriod, type PinnedFunnel } from "@/lib/funnel";
import { Card, CardHeader } from "@/components/Card";
import { translate, useLanguage } from "@/components/LanguageContext";

export function PinnedFunnelCard({ tenantId, from, to }: { tenantId: string; from: number; to: number }) {
  const { language } = useLanguage();
  const t = (key: string) => translate(key, language);
  const percent = new Intl.NumberFormat(language, { style: "percent", maximumFractionDigits: 1 });
  const points = new Intl.NumberFormat(language, { signDisplay: "exceptZero", maximumFractionDigits: 1 });
  const [pin, setPin] = useState<PinnedFunnel | null>();
  const [snapshot, setSnapshot] = useState<{
    from: number;
    to: number;
    current: ReturnType<typeof funnelSummary>;
    previousRate: number | null;
  }>();
  const [error, setError] = useState(false);

  useEffect(() => {
    let active = true;
    api.analytics.pinnedFunnel(tenantId).then(
      (definition) => { if (active) setPin(definition); },
      () => { if (active) setError(true); },
    );
    return () => { active = false; };
  }, [tenantId]);

  useEffect(() => {
    if (!pin) return;
    let active = true;
    const previous = previousPeriod(from, to);
    Promise.all([
      api.analytics.funnel(tenantId, pin.steps, from, to),
      api.analytics.funnel(tenantId, pin.steps, previous.from_ts, previous.to_ts).catch(() => []),
    ]).then(([current, prior]) => {
      if (!active) return;
      setSnapshot({ from, to, current: funnelSummary(current), previousRate: funnelSummary(prior).rate });
      setError(false);
    }, () => { if (active) setError(true); });
    return () => { active = false; };
  }, [tenantId, pin, from, to]);

  const data = snapshot?.from === from && snapshot.to === to ? snapshot : undefined;
  const drop = data?.current.biggestDrop;
  const delta = data?.current.rate != null && data.previousRate !== null
    ? (data.current.rate - data.previousRate) * 100
    : null;

  return (
    <Card className="mt-6">
      <CardHeader title={t("Pinned funnel")} action={
        <Link href={pin ? "/analytics/funnel?pinned=1" : "/analytics/funnel"} className="text-sm text-accent hover:underline">
          {t(pin ? "Open pinned funnel" : "Configure funnel")}
        </Link>
      } />
      <div className="space-y-3 p-5 text-sm">
        {pin && <p className="break-words font-semibold text-fg"><bdi>{pin.name}</bdi></p>}
        {error && <p role="alert" className="text-danger">{t("Could not load pinned funnel.")}</p>}
        {!error && pin === null && <p className="text-fg-muted">{t("No funnel pinned for this tenant.")}</p>}
        {!error && pin !== null && !data && <p className="text-fg-muted">{t("Loading pinned funnel…")}</p>}
        {data && pin && (
          <>
            <dl className="grid grid-cols-1 gap-4 sm:grid-cols-3">
              <div><dt className="text-fg-muted">{t("Sessions entering")}</dt><dd className="text-lg font-semibold">{data.current.entered.toLocaleString(language)}</dd></div>
              <div><dt className="text-fg-muted">{t("Overall conversion")}</dt><dd className="text-lg font-semibold">{data.current.rate === null ? "—" : percent.format(data.current.rate)}</dd></div>
              <div><dt className="text-fg-muted">{t("Change vs previous period")}</dt><dd className="text-lg font-semibold">{delta === null ? "—" : <><bdi>{points.format(delta)}</bdi>{" "}{t("percentage points")}</>}</dd></div>
            </dl>
            {drop && (
              <div className="flex flex-wrap gap-x-3 rounded-md bg-warning/10 px-3 py-2">
                <span className="font-medium">{t("Largest drop-off")}</span>
                <span><bdi>{formatEventLabel(pin.steps[drop.index])}</bdi>{" → "}<bdi>{formatEventLabel(pin.steps[drop.index + 1])}</bdi></span>
                <bdi>{percent.format(1 - drop.rate)}</bdi>
                <span>{t("Dropped sessions")} <bdi>{drop.dropped.toLocaleString(language)}</bdi></span>
              </div>
            )}
          </>
        )}
      </div>
    </Card>
  );
}
