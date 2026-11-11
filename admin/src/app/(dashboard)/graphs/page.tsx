"use client";

import { useEffect, useState } from "react";
import { api, GraphResult, Question, Tenant } from "@/lib/api";
import { Card, CardHeader } from "@/components/Card";
import { Combobox } from "@/components/Combobox";
import { LineTrend, BarBreakdown } from "@/components/Chart";
import { Button } from "@/components/Button";

export default function GraphsPage() {
  const [tenants, setTenants] = useState<Tenant[]>([]), [tenant, setTenant] = useState("");
  const [questions, setQuestions] = useState<Question[]>([]), [results, setResults] = useState<Record<string, GraphResult>>({});
  const [loadingQuestions, setLoadingQuestions] = useState(false), [loadingGraph, setLoadingGraph] = useState("");
  const [chart, setChart] = useState("line"), [mode, setMode] = useState("relative"), [days, setDays] = useState(7);
  const [from, setFrom] = useState(""), [to, setTo] = useState(""), [error, setError] = useState("");

  useEffect(() => { api.tenants.list().then((items) => { setTenants(items); if (items[0]) setTenant(items[0].ID); }).catch((e) => setError(e.message)); }, []);
  useEffect(() => { if (!tenant) return; setLoadingQuestions(true); setError(""); setResults({}); api.questions.list(tenant).then(setQuestions).catch((e) => setError(e.message)).finally(() => setLoadingQuestions(false)); }, [tenant]);

  async function run(question: Question) {
    const now = Date.now();
    const fromTs = mode === "relative" ? now - Math.max(1, days) * 86400000 : new Date(from).getTime();
    const toTs = mode === "relative" ? now : new Date(to).getTime();
    if (!Number.isFinite(fromTs) || !Number.isFinite(toTs) || fromTs >= toTs) { setError("Choose a valid graph time range."); return; }
    setLoadingGraph(question.ID); setError("");
    try { setResults((current) => ({ ...current, [question.ID]: await api.questions.results(tenant, question.ID, { from_ts: fromTs, to_ts: toTs }) })); }
    catch (e) { setError((e as Error).message); } finally { setLoadingGraph(""); }
  }

  return <div><h1 className="text-xl font-semibold text-fg">Graphs</h1><p className="mt-1 text-sm text-fg-muted">Load saved questions with a shared time range and visualization. Query data comes from the API.</p>
    <Card className="mt-5"><CardHeader title="Graph controls" /><div className="flex flex-wrap gap-3 p-5"><Combobox value={tenant} onChange={setTenant} options={tenants.map((item) => ({ value: item.ID, label: item.Name }))} placeholder="Select tenant" /><Combobox value={chart} onChange={setChart} options={[{ value: "line", label: "Line chart" }, { value: "bar", label: "Bar chart" }]} placeholder="Chart type" /><Combobox value={mode} onChange={setMode} options={[{ value: "relative", label: "Relative range" }, { value: "absolute", label: "Absolute range" }]} placeholder="Time range" />{mode === "relative" ? <label className="flex items-center gap-2 text-sm text-fg-muted">Last days<input type="number" min="1" value={days} onChange={(e) => setDays(Math.max(1, Number(e.target.value) || 1))} className="w-20 rounded border border-border bg-surface px-2 py-1 text-fg" /></label> : <><input aria-label="From" type="datetime-local" value={from} onChange={(e) => setFrom(e.target.value)} className="rounded border border-border bg-surface px-2 py-1" /><input aria-label="To" type="datetime-local" value={to} onChange={(e) => setTo(e.target.value)} className="rounded border border-border bg-surface px-2 py-1" /></>}</div></Card>
    {error && <p className="mt-4 rounded border border-danger/30 bg-danger/5 p-3 text-sm text-danger">{error}</p>}<div className="mt-5 grid gap-5 lg:grid-cols-2">{loadingQuestions ? <Card><p className="p-5 text-sm text-fg-muted">Loading questions…</p></Card> : questions.length === 0 ? <Card><p className="p-5 text-sm text-fg-muted">No questions exist for this tenant. Create one on the Questions page first.</p></Card> : questions.map((question) => { const result = results[question.ID]; return <Card key={question.ID}><CardHeader title={question.Name} action={<Button onClick={() => void run(question)} loading={loadingGraph === question.ID}>Load graph</Button>} /><div className="p-5"><p className="mb-3 text-xs text-fg-muted">{(question.EventNames || question.EventName).split(",").join(" · ")}</p>{result ? (chart === "bar" ? <BarBreakdown data={result.points} /> : <LineTrend data={result.points} />) : <p className="text-sm text-fg-muted">Click Load graph to fetch this question’s data.</p>}</div></Card>; })}</div></div>;
}
