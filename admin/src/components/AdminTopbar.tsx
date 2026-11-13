"use client";

import { usePathname } from "next/navigation";
import { HelpIcon } from "@/components/HelpIcon";
import { translate, useLanguage } from "@/components/LanguageContext";

const LABELS: Record<string, string> = {
  dashboard: "Dashboard", stream: "Live Stream", analytics: "Analytics", graphs: "Graphs", questions: "Questions",
  tenants: "Tenants", sessions: "Sessions", events: "Events", schemas: "Schemas", cohorts: "Cohorts", alerts: "Alerts",
  operations: "Operations", settings: "Settings", replay: "Replay", experiments: "Experiments", "feature-flags": "Feature Flags",
  aggregations: "Aggregations", anomalies: "Anomalies", "audit-log": "Audit Log", "dead-letters": "Dead Letters", "test-event": "Test Event", "cdn-test": "CDN Test",
  intelligence: "Intelligence",
};

const HELP: Record<string, string> = {
  dashboard: "Start here for system health, ingestion volume, and the newest activity. Use the linked cards to move directly into a resource.",
  stream: "Live Stream polls recent events continuously. Start without filters for a broad view, then narrow by tenant, event, or time range.",
  analytics: "Build shareable API-backed queries with event filters, grouping, aggregations, and relative or absolute time ranges.",
  graphs: "Use this page for a focused chart from a question and time range. Graph data is loaded from the API, not calculated only in the browser.",
  tenants: "Tenants isolate customer data and API keys. Open a tenant for usage, quotas, rotation, and related sessions.",
  sessions: "Sessions are ordered newest first. Open a session to inspect its compact journey, metadata, and event timeline.",
  users: "Users are inferred from session fingerprints because the event contract has no explicit user_id. Location comes from DeviceInfo; Online means seen within five minutes.",
  events: "Events are ordered newest first. Open an event for its complete payload, dimensions, source, and linked session.",
  alerts: "Alerts are server-evaluated rules. Test destinations carefully; failed deliveries are retried and eventually appear in Dead Letters.",
  replay: "Replay a relative or absolute event range into the ingest pipeline. Verify the tenant and range before starting because this creates downstream traffic.",
  schemas: "Schemas are versioned and immutable. Registering the same name creates a new version without rewriting historical events.",
  cohorts: "Cohorts store reusable membership rules. Materialize recomputes the current matching session snapshot.",
};

export function AdminTopbar() {
  const pathname = usePathname() || "/dashboard";
  const segment = pathname.split("/").filter(Boolean)[0] || "dashboard";
  const title = LABELS[segment] || "Admin";
  const { language } = useLanguage();
  return <header className="sticky top-0 z-20 flex min-h-14 items-center justify-between border-b border-border bg-bg/90 px-8 backdrop-blur">
    <div className="flex min-w-0 items-center gap-2 text-sm"><span className="font-medium text-fg">{translate(title, language)}</span>{pathname !== `/${segment}` && <><span className="text-fg-muted">/</span><span className="truncate text-fg-muted">{pathname.split("/").filter(Boolean).slice(1).join(" / ")}</span></>}</div>
    <div className="flex items-center gap-3 text-xs text-fg-muted"><HelpIcon title={translate(title, language)}><div className="space-y-2"><p>{language === "fa" ? "از این صفحه برای فیلتر، بررسی و تازه‌سازی داده‌های فعلی سرور استفاده کنید." : HELP[segment] || "Use this page to filter, inspect, and refresh current server data."}</p><p><span className="font-medium text-fg">{language === "fa" ? "داده:" : "Data:"}</span> {language === "fa" ? "فهرست‌ها وضعیت فعلی API را نشان می‌دهند." : "lists reflect current API state."}</p><p><span className="font-medium text-fg">{language === "fa" ? "عملیات:" : "Actions:"}</span> {language === "fa" ? "برای جزئیات، ویرایش، کپی و تازه‌سازی امن از عملیات ردیف استفاده کنید." : "use row actions for details, edits, copies, and safe refreshes."}</p></div></HelpIcon><span className="hidden rounded-full border border-border bg-surface px-2.5 py-1 sm:inline-flex">{translate("Admin console", language)}</span><span className="rounded-full border border-border bg-surface px-2.5 py-1">{translate("Protected", language)}</span></div>
  </header>;
}
