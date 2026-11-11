"use client";
import { useCallback, useEffect, useState } from "react";
import { api, Cohort, Tenant, ApiError } from "@/lib/api";
import { Card, CardHeader } from "@/components/Card";
import { Button } from "@/components/Button";
import { Input, Label, Select } from "@/components/Input";
import { Combobox, ComboOption } from "@/components/Combobox";
import { DataTable, Column, TableQuery } from "@/components/DataTable";
import { Badge } from "@/components/Badge";
import { useToast } from "@/components/Toast";

export default function CohortsPage() {
  const [tenants, setTenants] = useState<Tenant[]>([]);
  const [tenantId, setTenantId] = useState("");
  const [cohorts, setCohorts] = useState<Cohort[]>([]);
  const [totalCohorts, setTotalCohorts] = useState(0);
  const [eventNameOptions, setEventNameOptions] = useState<ComboOption[]>([]);

  const [name, setName] = useState("");
  const [eventName, setEventName] = useState("");
  const [minCount, setMinCount] = useState(1);
  const [creating, setCreating] = useState(false);
  const [memberCounts, setMemberCounts] = useState<Record<string, number>>({});
  const [materializing, setMaterializing] = useState<string | null>(null);
  const { push } = useToast();

  useEffect(() => {
    api.tenants.list().then((ts) => {
      setTenants(ts);
      if (ts[0]) setTenantId(ts[0].ID);
    });
    api.events
      .list({})
      .then((all) => {
        const counts = new Map<string, number>();
        for (const e of all) counts.set(e.EventName, (counts.get(e.EventName) || 0) + 1);
        setEventNameOptions([...counts.entries()].sort((a, b) => b[1] - a[1]).map(([n]) => ({ value: n })));
      })
      .catch(() => {});
  }, []);

  const refresh = useCallback(async (query: TableQuery = { filter: "", sortKey: "created", sortDir: "desc", page: 0 }) => {
    if (!tenantId) return;
    try {
      const page = await api.cohorts.listPage(tenantId, { filter: query.filter, dir: query.sortDir, limit: 10, offset: query.page * 10 }); setCohorts(page.items); setTotalCohorts(page.total);
    } catch (e) {
      push("error", e instanceof ApiError ? e.message : "Failed to load cohorts");
    }
  }, [push, tenantId]);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  async function create() {
    if (!name.trim() || !eventName.trim() || !tenantId) return;
    setCreating(true);
    try {
      await api.cohorts.create(tenantId, name.trim(), eventName.trim(), minCount);
      setName("");
      setEventName("");
      setMinCount(1);
      push("success", `Cohort "${name.trim()}" created`);
      await refresh();
    } catch (e) {
      push("error", e instanceof ApiError ? e.message : "Failed to create cohort");
    } finally {
      setCreating(false);
    }
  }

  async function materialize(c: Cohort) {
    setMaterializing(c.ID);
    try {
      const { members } = await api.cohorts.materialize(c.ID);
      setMemberCounts((prev) => ({ ...prev, [c.ID]: members }));
      push("success", `"${c.Name}" now has ${members} member${members === 1 ? "" : "s"}`);
    } catch (e) {
      push("error", e instanceof ApiError ? e.message : "Failed to materialize cohort");
    } finally {
      setMaterializing(null);
    }
  }

  const columns: Column<Cohort>[] = [
    { key: "name", header: "Name", render: (c) => <span className="font-medium text-fg">{c.Name}</span>, sortValue: (c) => c.Name },
    { key: "rule", header: "Rule", render: (c) => <span className="text-fg-muted">{c.EventName} &ge; {c.MinCount}&times;</span> },
    {
      key: "members",
      header: "Members",
      render: (c) => (c.ID in memberCounts ? <Badge tone="accent">{memberCounts[c.ID]}</Badge> : <span className="text-fg-muted">not materialized</span>),
    },
    {
      key: "actions",
      header: "",
      render: (c) => (
        <div className="flex justify-end">
          <Button variant="secondary" onClick={() => materialize(c)} loading={materializing === c.ID}>
            Materialize
          </Button>
        </div>
      ),
    },
  ];

  return (
    <div>
      <h1 className="text-xl font-semibold text-fg">Cohorts</h1>
      <p className="mt-1 text-sm text-fg-muted">
        Rule-based segments: sessions with at least N occurrences of an event. Materialization is on-demand (run it below), not a live
        stream — matching the backend&apos;s honest recompute job (see phases.md Phase 6).
      </p>
      <details className="mt-4 rounded-lg border border-border bg-surface-2 px-4 py-3 text-sm"><summary className="cursor-pointer font-medium text-fg">Cohort help</summary><p className="mt-2 text-fg-muted">Creating a cohort stores its rule. Use Materialize to recompute matching sessions; membership is a snapshot, not a live subscription.</p></details>

      <Card className="mt-6">
        <CardHeader title="New cohort" />
        <div className="flex flex-wrap items-end gap-3 p-5">
          <div className="min-w-[160px]">
            <Label>Tenant</Label>
            <Select value={tenantId} onChange={(e) => setTenantId(e.target.value)}>
              {tenants.length === 0 && <option value="">Create a tenant first</option>}
              {tenants.map((t) => (
                <option key={t.ID} value={t.ID}>
                  {t.Name}
                </option>
              ))}
            </Select>
          </div>
          <div className="min-w-[180px]">
            <Label>Name</Label>
            <Input value={name} onChange={(e) => setName(e.target.value)} placeholder="power_users" />
          </div>
          <div className="min-w-[180px]">
            <Label>Event name</Label>
            <Combobox value={eventName} onChange={setEventName} options={eventNameOptions} placeholder="purchase" />
          </div>
          <div className="w-28">
            <Label>Min count</Label>
            <Input type="number" min={1} value={minCount} onChange={(e) => setMinCount(Math.max(1, Number(e.target.value) || 1))} />
          </div>
          <Button onClick={create} loading={creating} disabled={!name.trim() || !eventName.trim() || !tenantId}>
            Create cohort
          </Button>
        </div>
      </Card>

      <Card className="mt-6">
        <CardHeader title={`Cohorts (${cohorts.length})`} />
        <DataTable columns={columns} rows={cohorts} rowKey={(c) => c.ID} emptyMessage="No cohorts yet for this tenant." serverSide onQueryChange={(query) => void refresh(query)} totalRows={totalCohorts} />
      </Card>
    </div>
  );
}
