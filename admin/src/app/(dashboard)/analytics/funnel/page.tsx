"use client";

import { Suspense, useEffect, useState } from "react";
import { useSearchParams } from "next/navigation";
import { api } from "@/lib/api";
import { Card, CardHeader, StatCard } from "@/components/Card";
import { Button } from "@/components/Button";
import { Combobox } from "@/components/Combobox";
import { translate, useLanguage } from "@/components/LanguageContext";
import { formatEventLabel, funnelSummary, type PinnedFunnel } from "@/lib/funnel";
import { useTenant } from "@/components/TenantContext";
import { formatTimeRange, useTimeRange } from "@/components/TimeRangeContext";

type Step = { event_name: string; sessions: number; conversion_rate: number };

export default function FunnelPage() {
  return <Suspense><ScopedFunnel /></Suspense>;
}

function ScopedFunnel() {
  const { tenantId, loading } = useTenant();
  const { to } = useTimeRange();
  const openPinned = useSearchParams().get("pinned") === "1";
  return !loading && !!to && <FunnelEditor key={`${tenantId}:${openPinned}`} openPinned={openPinned} />;
}

function FunnelEditor({ openPinned }: { openPinned: boolean }) {
  const { language } = useLanguage();
  const t = (text: string) => translate(text, language);
  const percent = new Intl.NumberFormat(language, { style: "percent", maximumFractionDigits: 1 });
  const { tenantId: tenant, setTenantId: setGlobalTenant, tenants } = useTenant();
  const { from: globalFrom, to: globalTo, hours } = useTimeRange();
  const timeRangeLabel = formatTimeRange(hours);
  const [steps, setSteps] = useState<string[]>([]);
  const [submitted, setSubmitted] = useState<string[] | null>(null);
  const [titles, setTitles] = useState<Record<string, string>>({});
  const [input, setInput] = useState("");
  const [eventOptions, setEventOptions] = useState<string[]>([]);
  const [result, setResult] = useState<Step[]>([]);
  const [loading, setLoading] = useState(false);
  const [suggesting, setSuggesting] = useState(false);
  const [error, setError] = useState("");

  const [pin, setPin] = useState<PinnedFunnel | null>();
  const [pinName, setPinName] = useState("");
  const [pinBusy, setPinBusy] = useState(false);
  const [pinError, setPinError] = useState("");
  const [pinStatus, setPinStatus] = useState("");

  useEffect(() => {
    if (!tenant) return;
    let active = true;
    api.analytics.pinnedFunnel(tenant).then(
      (definition) => {
        if (!active) return;
        setPin(definition);
        if (!definition) return;
        setPinName((name) => name || definition.name);
        if (openPinned) {
          setSteps(definition.steps);
          setSubmitted(definition.steps);
        }
      },
      () => { if (active) setPinError("Could not load pinned funnel."); },
    );
    return () => { active = false; };
  }, [tenant, openPinned]);

  useEffect(() => {
    let active = true;
    api.eventOptions(tenant, input, false, { from_ts: globalFrom, to_ts: globalTo }).then(
      (options) => { if (active) setEventOptions(options.map((option) => option.value)); },
      () => { if (active) setEventOptions([]); },
    );
    return () => { active = false; };
  }, [tenant, input, globalFrom, globalTo]);

  useEffect(() => {
    if (!submitted) return;
    let active = true;
    setLoading(true);
    setError("");
    api.analytics.funnel(tenant, submitted, globalFrom, globalTo).then(
      (rows) => { if (active) setResult(rows); },
      (e: Error) => { if (active) { setError(e.message); setResult([]); } },
    ).finally(() => { if (active) setLoading(false); });
    return () => { active = false; };
  }, [tenant, submitted, globalFrom, globalTo]);

  function editSteps(next: string[]) {
    setSteps(next);
    setSubmitted(null);
    setResult([]);
    setLoading(false);
    setError("");
  }

  async function loadSuggested() {
    setSuggesting(true);
    setError("");
    try {
      const suggestion = await api.analytics.suggestedFunnel(tenant, globalFrom, globalTo);
      const valid = suggestion.filter((item) => typeof item.event_name === "string" && item.event_name.trim());
      const names = [...new Set(valid.map((item) => item.event_name))];
      if (names.length < 2) {
        setError(t("No funnel suggestion for this range."));
        return;
      }
      editSteps(names);
      setTitles(Object.fromEntries(valid.map((item) => [item.event_name, item.title])));
      setSubmitted(names);
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setSuggesting(false);
    }
  }

  function addStep(value: string) {
    const name = value.trim();
    if (name && !steps.includes(name)) editSteps([...steps, name]);
    setInput("");
  }

  function moveStep(index: number, destination: number) {
    const reordered = [...steps];
    const [step] = reordered.splice(index, 1);
    reordered.splice(destination, 0, step);
    editSteps(reordered);
  }

  async function savePin() {
    setPinBusy(true);
    setPinError("");
    setPinStatus("");
    try {
      const definition = await api.analytics.pinFunnel(tenant, { name: pinName.trim(), steps });
      setPin(definition);
      setPinName(definition.name);
      setPinStatus("Funnel pinned to dashboard.");
    } catch {
      setPinError("Could not save pinned funnel.");
    } finally {
      setPinBusy(false);
    }
  }

  async function removePin() {
    setPinBusy(true);
    setPinError("");
    setPinStatus("");
    try {
      await api.analytics.unpinFunnel(tenant);
      setPin(null);
      setPinStatus("Funnel removed from dashboard.");
    } catch {
      setPinError("Could not remove pinned funnel.");
    } finally {
      setPinBusy(false);
    }
  }

  const label = (name: unknown) => formatEventLabel(name, titles);
  const { entered: firstSessions, completed: completedSessions, rate: overallConversion, transitions, biggestDrop } = funnelSummary(result);

  if (tenant && openPinned && pin === undefined && !pinError) {
    return <p className="text-sm text-fg-muted">{t("Loading pinned funnel…")}</p>;
  }

  return (
    <div>
      <h1 className="text-xl font-semibold text-fg">Funnel Analysis</h1>
      <p className="mt-1 text-sm text-fg-muted">
        {t("Add ordered events or use a smart suggestion to analyze a session journey.")}
      </p>
      <details className="mt-4 rounded-lg border border-border bg-surface-2 px-4 py-3 text-sm">
        <summary className="cursor-pointer font-medium text-fg">
          How funnels work
        </summary>
        <p className="mt-2 text-fg-muted">
          Suggestions start with the most common first event in a session and
          follow the most common next events. A session only advances when
          events occur in order.
        </p>
      </details>
      <Card className="mt-6">
        <CardHeader
          title="Funnel definition"
          action={
            <Button
              variant="secondary"
              onClick={() => void loadSuggested()}
              loading={suggesting}
              disabled={loading}
            >
              {t("Use smart suggestion")}
            </Button>
          }
        />
        <div className="space-y-4 p-5">
          <Combobox
            value={tenant}
            onChange={setGlobalTenant}
            options={tenants.map((item) => ({
              value: item.ID,
              label: item.Name,
            }))}
            placeholder="Select tenant"
          />
          <div className="flex gap-2">
            <Combobox
              value={input}
              onChange={setInput}
              options={eventOptions.map((value) => ({ value }))}
              placeholder="Search event and select…"
            />
            <Button variant="secondary" onClick={() => addStep(input)} disabled={suggesting}>
              Add step
            </Button>
          </div>
          <ol className="space-y-2">
            {steps.map((step, index) => (
              <li key={step} className="flex items-center justify-between gap-3 rounded-lg border border-border px-3 py-2 text-sm">
                <span>{index + 1}. <bdi>{label(step)}</bdi></span>
                <div className="flex gap-1">
                  <Button variant="secondary" aria-label={`${t("Move stage earlier")} ${label(step)}`}
                    disabled={suggesting || index === 0} onClick={() => moveStep(index, index - 1)}>↑</Button>
                  <Button variant="secondary" aria-label={`${t("Move stage later")} ${label(step)}`}
                    disabled={suggesting || index === steps.length - 1} onClick={() => moveStep(index, index + 1)}>↓</Button>
                  <Button variant="secondary" aria-label={`${t("Remove")} ${label(step)}`}
                    disabled={suggesting} onClick={() => editSteps(steps.filter((item) => item !== step))}>×</Button>
                </div>
              </li>
            ))}
          </ol>
          <p className="text-xs text-fg-muted">Global time range: <span className="font-medium text-fg">{timeRangeLabel}</span></p>
          <Button
            onClick={() => setSubmitted([...steps])}
            loading={loading}
            disabled={suggesting || steps.length < 2}
          >
            Run funnel
          </Button>
        </div>
      </Card>
      {tenant && (
        <Card className="mt-6">
          <CardHeader title={t("Pinned funnel")} />
          <div className="space-y-3 p-5">
            <label className="block text-sm font-medium">
              {t("Funnel name")}
              <input
                className="mt-1 block w-full max-w-sm rounded-lg border border-border bg-surface px-3 py-2 text-fg"
                value={pinName}
                onChange={(event) => setPinName(event.target.value)}
              />
            </label>
            <div className="flex flex-wrap gap-2">
              <Button loading={pinBusy} disabled={!pinName.trim() || steps.length < 2 || suggesting || (pin === undefined && !pinError)}
                onClick={() => void savePin()}>{t(pin ? "Update pinned funnel" : "Pin to dashboard")}</Button>
              {pin && (
                <>
                  <Button variant="secondary" disabled={pinBusy || suggesting} onClick={() => {
                    editSteps(pin.steps);
                    setPinName(pin.name);
                    setSubmitted(pin.steps);
                  }}>{t("Load pinned funnel")}</Button>
                  <Button variant="secondary" disabled={pinBusy} onClick={() => void removePin()}>{t("Unpin funnel")}</Button>
                </>
              )}
            </div>
            {pinError && <p role="alert" className="text-sm text-danger">{t(pinError)}</p>}
            {pinStatus && <p role="status" className="text-sm text-fg-muted">{t(pinStatus)}</p>}
          </div>
        </Card>
      )}
      {error && (
        <p className="mt-4 rounded border border-danger/30 bg-danger/5 p-3 text-sm text-danger">
          {error}
        </p>
      )}
      {(loading || suggesting) && !result.length && (
        <Card className="mt-6">
          <p className="p-5 text-sm text-fg-muted">
            Discovering the tenant’s session journey…
          </p>
        </Card>
      )}
      {submitted && !loading && !suggesting && !result.length && !error && (
        <Card className="mt-6">
          <p className="p-5 text-sm text-fg-muted">
            No ordered session path was found for this tenant.
          </p>
        </Card>
      )}
      {result.length > 0 && (
        <>
          <div className="mt-6 grid grid-cols-1 gap-4 sm:grid-cols-3">
            <StatCard label={t("Sessions entering")} value={firstSessions.toLocaleString(language)} hint={t("Sessions reaching stage 1")} />
            <StatCard label={t("Sessions completing")} value={completedSessions.toLocaleString(language)} hint={t("Sessions reaching the final stage")} />
            <StatCard label={t("Overall conversion")} value={overallConversion === null ? "—" : percent.format(overallConversion)} hint={t("Final stage ÷ first stage")} />
          </div>
          <Card className="mt-6">
            <CardHeader
              title={t("Funnel stages")}
              action={<span className="text-xs text-fg-muted">{t("Share of first-stage sessions")}</span>}
            />
            <ol className="divide-y divide-border">
              {result.map((stage, index) => {
                const name = label(stage.event_name);
                const stageLabel = `${index + 1}. ${name}`;
                const share = firstSessions ? stage.sessions / firstSessions : 0;
                const transition = transitions[index];
                const highlight = biggestDrop?.index === index;
                return (
                  <li key={`${stage.event_name}-${index}`} className="space-y-3 p-5">
                    <div className="flex flex-wrap items-baseline justify-between gap-x-4 gap-y-1">
                      <p className="min-w-0 break-words font-medium text-fg">
                        {index + 1}. <bdi>{name}</bdi>
                      </p>
                      <p className="flex gap-3 text-sm tabular-nums text-fg-muted">
                        <span>{t("Sessions")} <bdi>{stage.sessions.toLocaleString(language)}</bdi></span>
                        <bdi>{firstSessions ? percent.format(share) : "—"}</bdi>
                      </p>
                    </div>
                    <div
                      role={firstSessions ? "meter" : undefined}
                      aria-hidden={firstSessions ? undefined : true}
                      aria-label={stageLabel}
                      aria-valuemin={0}
                      aria-valuemax={firstSessions}
                      aria-valuenow={stage.sessions}
                      className="h-3 overflow-hidden rounded-full bg-surface-2"
                    >
                      <div className="h-full rounded-full bg-accent" style={{ width: `${share * 100}%` }} />
                    </div>
                    {transition && (
                      <div className={`flex flex-wrap items-baseline gap-x-4 gap-y-1 rounded-md px-3 py-2 text-xs ${highlight ? "bg-warning/10 font-semibold text-fg" : "bg-surface-2 text-fg-muted"}`}>
                        {highlight && (
                          <span className="rounded bg-warning/20 px-2 py-0.5">
                            {t("Largest drop-off")} <bdi>{percent.format(1 - biggestDrop.rate)}</bdi>
                          </span>
                        )}
                        <span>{t("Dropped sessions")} <bdi>{transition.dropped.toLocaleString(language)}</bdi></span>
                        <span>{t("Conversion to next stage")} <bdi>{transition.rate === null ? "—" : percent.format(transition.rate)}</bdi></span>
                      </div>
                    )}
                  </li>
                );
              })}
            </ol>
          </Card>
        </>
      )}
    </div>
  );
}
