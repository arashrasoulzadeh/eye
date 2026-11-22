"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { api, type EventOption } from "@/lib/api";
import { Card, CardHeader } from "@/components/Card";
import { translate, useLanguage } from "@/components/LanguageContext";

export function DashboardEvents({ tenantId, from, to }: { tenantId: string; from: number; to: number }) {
  const { language } = useLanguage();
  const t = (label: string) => translate(label, language);
  const [snapshot, setSnapshot] = useState<{ from: number; to: number; tenantId: string; events: EventOption[] | null }>();
  const current = snapshot?.tenantId === tenantId && snapshot.from === from && snapshot.to === to ? snapshot : undefined;

  useEffect(() => {
    let active = true;
    api.eventOptions(tenantId, "", false, { from_ts: from, to_ts: to }).then(
      (events) => { if (active) setSnapshot({ tenantId, from, to, events }); },
      () => { if (active) setSnapshot({ tenantId, from, to, events: null }); },
    );
    return () => { active = false; };
  }, [tenantId, from, to]);

  return (
    <Card className="mt-6">
      <CardHeader title={t("Top events by volume")} action={
        <Link className="text-sm text-accent hover:underline" href="/aggregations">{t("Aggregations")}</Link>
      } />
      <div className="p-5">
        {!current && <p className="text-sm text-fg-muted">{t("Loading event counts…")}</p>}
        {current?.events === null && <p role="alert" className="text-sm text-danger">{t("Could not load event counts.")}</p>}
        {current?.events?.length === 0 && <p className="text-sm text-fg-muted">{t("No events match the current filters.")}</p>}
        {!!current?.events?.length && (
          <>
            <p className="mb-2 text-xs text-fg-muted">{t("Counts include raw and generated event records.")}</p>
            <ol className="divide-y divide-border">
              {current.events.map((event) => (
                <li key={event.value} className="flex items-center justify-between gap-4 py-2 text-sm">
                  <Link
                    href={{ pathname: "/events", query: { event_name: event.value, tenant_id: tenantId, from_ts: from, to_ts: to } }}
                    className="min-w-0 break-all text-accent hover:underline"
                  >
                    <bdi>{event.value}</bdi>
                  </Link>
                  <span className="tabular-nums text-fg-muted">{event.count.toLocaleString(language)}</span>
                </li>
              ))}
            </ol>
          </>
        )}
      </div>
    </Card>
  );
}
