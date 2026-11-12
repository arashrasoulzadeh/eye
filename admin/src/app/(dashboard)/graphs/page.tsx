"use client";
import { useEffect, useState } from "react";
import { api, GraphResult, Question, Tenant } from "@/lib/api";
import { Card, CardHeader } from "@/components/Card";
import { Combobox } from "@/components/Combobox";
import { LineTrend, BarBreakdown } from "@/components/Chart";
import { Button } from "@/components/Button";
import { useTenant } from "@/components/TenantContext";
import { formatTimeRange, useTimeRange } from "@/components/TimeRangeContext";

export default function GraphsPage() {
  const { tenantId: globalTenant } = useTenant();
  const { from: globalFrom, to: globalTo, hours } = useTimeRange();
  const timeRangeLabel = formatTimeRange(hours);
  void timeRangeLabel;
  const [tenants, setTenants] = useState<Tenant[]>([]), [tenant, setTenant] = useState("");
  const [questions, setQuestions] = useState<Question[]>([]), [questionTotal, setQuestionTotal] = useState(0), [questionPage, setQuestionPage] = useState(0), [results, setResults] = useState<Record<string, GraphResult>>({});
  const [loadingQuestions, setLoadingQuestions] = useState(false), [loadingGraph, setLoadingGraph] = useState("");
  const [chart, setChart] = useState("line");
  const [error, setError] = useState("");
  useEffect(() => { if (globalTenant && globalTenant !== tenant) setTenant(globalTenant); }, [globalTenant, tenant]);

  useEffect(() => { api.tenants.list().then((items) => { setTenants(items); if (items[0]) setTenant(items[0].ID); }).catch((e) => setError(e.message)); }, []);
  useEffect(() => { if (!tenant) return; setLoadingQuestions(true); setError(""); setResults({}); api.questions.listPage(tenant, { limit: 10, offset: questionPage * 10 }).then((page) => { setQuestions(page.items); setQuestionTotal(page.total); }).catch((e) => setError(e.message)).finally(() => setLoadingQuestions(false)); }, [tenant, questionPage]);

  async function run(question: Question) {
    const fromTs = globalFrom;
    const toTs = globalTo;
    if (!Number.isFinite(fromTs) || !Number.isFinite(toTs) || fromTs >= toTs) { setError("Choose a valid graph time range."); return; }
    setLoadingGraph(question.ID); setError("");
    try { const result = await api.questions.results(tenant, question.ID, { from_ts: fromTs, to_ts: toTs }); setResults((current) => ({ ...current, [question.ID]: result })); }
    catch (e) { setError((e as Error).message); } finally { setLoadingGraph(""); }
  }

  return <div><h1 className="text-xl font-semibold text-fg">Graphs</h1><p className="mt-1 text-sm text-fg-muted">Load saved questions with a shared time range and visualization. Query data comes from the API.</p>
    <Card className="mt-5"><CardHeader title="Graph controls" /><div className="flex flex-wrap gap-3 p-5"><Combobox value={tenant} onChange={(value) => { setTenant(value); setQuestionPage(0); }} options={tenants.map((item) => ({ value: item.ID, label: item.Name }))} placeholder="Select tenant" /><Combobox value={chart} onChange={setChart} options={[{ value: "line", label: "Line chart" }, { value: "bar", label: "Bar chart" }]} placeholder="Chart type" /><span className="text-sm text-fg-muted">Global: <span className="font-medium text-fg">{timeRangeLabel}</span></span></div></Card>
    {error && <p className="mt-4 rounded border border-danger/30 bg-danger/5 p-3 text-sm text-danger">{error}</p>}<div className="mt-5 grid gap-5 lg:grid-cols-2">{loadingQuestions ? <Card><p className="p-5 text-sm text-fg-muted">Loading questions…</p></Card> : questions.length === 0 ? <Card><p className="p-5 text-sm text-fg-muted">No questions exist for this tenant. Create one on the Questions page first.</p></Card> : questions.map((question) => { const result = results[question.ID]; return <Card key={question.ID}><CardHeader title={question.Name} action={<Button onClick={() => void run(question)} loading={loadingGraph === question.ID}>Load graph</Button>} /><div className="p-5"><p className="mb-3 text-xs text-fg-muted">{(question.EventNames || question.EventName).split(",").join(" · ")}</p>{result ? (chart === "bar" ? <BarBreakdown data={result.points} /> : <LineTrend data={result.points} />) : <p className="text-sm text-fg-muted">Click Load graph to fetch this question’s data.</p>}</div></Card>; })}</div>{questionTotal > 10 && <div className="mt-5 flex items-center justify-between text-sm text-fg-muted"><span>Page {questionPage + 1} of {Math.ceil(questionTotal / 10)}</span><div className="flex gap-2"><Button variant="secondary" disabled={questionPage === 0 || loadingQuestions} onClick={() => setQuestionPage((page) => page - 1)}>Previous</Button><Button variant="secondary" disabled={(questionPage + 1) * 10 >= questionTotal || loadingQuestions} onClick={() => setQuestionPage((page) => page + 1)}>Next</Button></div></div>}</div>;
}
