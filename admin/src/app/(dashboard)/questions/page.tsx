"use client";

import { FormEvent, useCallback, useEffect, useState } from "react";
import { api, Question, Tenant } from "@/lib/api";
import { Card, CardHeader } from "@/components/Card";
import { Combobox } from "@/components/Combobox";
import { Button } from "@/components/Button";
import { DataTable, Column, TableQuery } from "@/components/DataTable";

type QuestionForm = { Name: string; clauses: string[]; MatchMode: "any" | "all" };
const initial: QuestionForm = { Name: "", clauses: [""], MatchMode: "any" };

export default function QuestionsPage() {
  const [tenants, setTenants] = useState<Tenant[]>([]), [tenant, setTenant] = useState("");
  const [popular, setPopular] = useState<string[]>([]), [search, setSearch] = useState("");
  const [items, setItems] = useState<Question[]>([]), [total, setTotal] = useState(0), [form, setForm] = useState<QuestionForm>(initial), [edit, setEdit] = useState<Question | null>(null);
  const [loading, setLoading] = useState(false), [saving, setSaving] = useState(false), [error, setError] = useState("");
  const load = useCallback(async (query: TableQuery = { filter: "", sortKey: "updated", sortDir: "desc", page: 0 }) => { if (!tenant) return; setLoading(true); try { const page = await api.questions.listPage(tenant, { filter: query.filter, dir: query.sortDir, limit: 10, offset: query.page * 10 }); setItems(page.items); setTotal(page.total); setError(""); } catch (e) { setError((e as Error).message); } finally { setLoading(false); } }, [tenant]);
  const loadEvents = useCallback(async (query = "") => { if (!tenant) return; try { const options = await api.eventOptions(tenant, query); setPopular((Array.isArray(options) ? options : []).map((event) => event.value)); } catch (e) { setError((e as Error).message); } }, [tenant]);
  useEffect(() => { api.tenants.list().then((list) => { setTenants(list); if (list[0]) setTenant(list[0].ID); }).catch((e) => setError(e.message)); }, []);
  useEffect(() => { if (tenant) { void load(); void loadEvents(); } }, [tenant, load, loadEvents]);
  async function submit(event: FormEvent) { event.preventDefault(); const clauses = form.clauses.map((value) => value.trim()).filter(Boolean); if (!tenant || !form.Name.trim() || !clauses.length) { setError("A tenant, question name, and at least one event are required."); return; } setSaving(true); setError(""); try { await api.questions.save({ ID: edit?.ID, TenantID: tenant, Name: form.Name.trim(), EventName: clauses[0], EventNames: clauses.join(","), MatchMode: form.MatchMode, RangeMode: "relative", RelativeDays: 1 }); setForm(initial); setEdit(null); await load(); } catch (e) { setError((e as Error).message); } finally { setSaving(false); } }
  function start(question: Question) { setEdit(question); setForm({ Name: question.Name, clauses: (question.EventNames || question.EventName).split(","), MatchMode: question.MatchMode || "any" }); }
  async function remove(question: Question) { if (!window.confirm(`Delete question “${question.Name}”?`)) return; try { await api.questions.remove(tenant, question.ID); await load(); } catch (e) { setError((e as Error).message); } }
  function addEvent(value: string) { if (!value || !popular.includes(value)) { setSearch(value); return; } setForm((current) => ({ ...current, clauses: [...new Set([...current.clauses.filter(Boolean), value])] })); setSearch(""); }
  const columns: Column<Question>[] = [
    { key: "name", header: "Name", render: (question) => <span className="font-medium text-fg">{question.Name}</span>, sortValue: (question) => question.Name },
    { key: "events", header: "Events", render: (question) => <span className="text-fg-muted">{(question.EventNames || question.EventName).split(",").join(" · ")}</span> },
    { key: "match", header: "Match", render: (question) => question.MatchMode, sortValue: (question) => question.MatchMode },
    { key: "updated", header: "Updated", render: (question) => question.UpdatedAt ? new Date(question.UpdatedAt * 1000).toLocaleString() : "—", sortValue: (question) => question.UpdatedAt || 0 },
    { key: "actions", header: "Actions", render: (question) => <div className="flex gap-2"><Button variant="secondary" onClick={() => start(question)}>Edit</Button><Button variant="danger" onClick={() => void remove(question)}>Delete</Button></div> },
  ];
  return <div><h1 className="text-xl font-semibold text-fg">Questions</h1><p className="mt-1 text-sm text-fg-muted">Questions define reusable event clauses. Time ranges and chart types are configured in Graphs.</p>{error && <p className="mt-4 rounded border border-danger/30 bg-danger/5 p-3 text-sm text-danger">{error}</p>}<label className="mt-5 block text-sm text-fg-muted">Tenant<Combobox value={tenant} onChange={setTenant} options={tenants.map((item) => ({ value: item.ID, label: item.Name }))} placeholder="Select tenant" maxVisible={5} /></label>
    <Card className="mt-5"><CardHeader title={edit ? "Edit question" : "New question"} /><form onSubmit={submit} className="grid gap-3 p-5"><input required placeholder="Question name" className="rounded border border-border bg-surface px-3 py-2" value={form.Name} onChange={(e) => setForm({ ...form, Name: e.target.value })} /><Combobox value={search} onChange={addEvent} onSearch={loadEvents} options={popular.map((value) => ({ value }))} placeholder="Type to search and add an event…" maxVisible={5} /><div className="flex flex-wrap gap-2">{form.clauses.filter(Boolean).map((value) => <button type="button" key={value} className="rounded-full bg-accent/10 px-3 py-1 text-xs text-accent" onClick={() => setForm({ ...form, clauses: form.clauses.filter((item) => item !== value) })}>{value} ×</button>)}</div><Combobox value={form.MatchMode} onChange={(value) => setForm({ ...form, MatchMode: value as "any" | "all" })} options={[{ value: "any", label: "Match any event" }, { value: "all", label: "Match all events" }]} placeholder="Match mode" maxVisible={5} /><div className="flex gap-2"><Button type="submit" loading={saving} disabled={!tenant}>{edit ? "Save changes" : "Create question"}</Button>{edit && <Button type="button" variant="secondary" onClick={() => { setEdit(null); setForm(initial); }}>Cancel</Button>}</div></form></Card>
    <Card className="mt-5"><CardHeader title={`Saved questions (${items.length})`} /><DataTable columns={columns} rows={items} rowKey={(question) => question.ID} emptyMessage="No questions for this tenant yet." loading={loading} serverSide totalRows={total} defaultSortKey="updated" onQueryChange={(query) => void load(query)} /></Card></div>;
}
