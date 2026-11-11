"use client";
import { useCallback, useEffect, useState } from "react";
import { api, AlertRule, Tenant, ApiError } from "@/lib/api";
import { Card, CardHeader } from "@/components/Card";
import { Button } from "@/components/Button";
import { Input, Label, Select } from "@/components/Input";
import { Combobox, ComboOption } from "@/components/Combobox";
import { DataTable, Column, TableQuery } from "@/components/DataTable";
import { Badge } from "@/components/Badge";
import { ConfirmDialog } from "@/components/ConfirmDialog";
import { useToast } from "@/components/Toast";

export default function AlertsPage() {
  const [tenants, setTenants] = useState<Tenant[]>([]);
  const [tenantId, setTenantId] = useState("");
  const [rules, setRules] = useState<AlertRule[]>([]);
  const [totalRules, setTotalRules] = useState(0);
  const [eventNameOptions, setEventNameOptions] = useState<ComboOption[]>([]);

  const [eventName, setEventName] = useState("");
  const [dimsKey, setDimsKey] = useState("");
  const [metric, setMetric] = useState("count");
  const [windowSeconds, setWindowSeconds] = useState(300);
  const [threshold, setThreshold] = useState(10);
  const [webhookUrl, setWebhookUrl] = useState("");
  const [destination, setDestination] = useState("webhook");
  const [editing, setEditing] = useState<AlertRule | null>(null);
  const [creating, setCreating] = useState(false);
  const [loading, setLoading] = useState(true);
  const [toDelete, setToDelete] = useState<AlertRule | null>(null);
  const { push } = useToast();

  useEffect(() => {
    api.tenants.list().then((ts) => {
      setTenants(ts);
      if (ts[0]) setTenantId(ts[0].ID);
    }).catch((e) => push("error", e instanceof ApiError ? e.message : "Failed to load tenants"));
  }, [push]);

  useEffect(() => { if (!tenantId) return; api.events.list({ tenant_id: tenantId, limit: 100, sort: "desc" }).then((events) => setEventNameOptions([...new Set(events.map((event) => event.EventName))].map((value) => ({ value })))).catch((e) => push("error", e instanceof ApiError ? e.message : "Failed to load event names")); }, [push, tenantId]);

  const refresh = useCallback(async (query: TableQuery = { filter: "", sortKey: "created", sortDir: "desc", page: 0 }) => {
    if (!tenantId) return;
    setLoading(true);
    try {
      const page = await api.alerts.listPage(tenantId, { filter: query.filter, dir: query.sortDir, limit: 10, offset: query.page * 10 }); setRules(page.items); setTotalRules(page.total);
    } catch (e) {
      push("error", e instanceof ApiError ? e.message : "Failed to load alert rules");
    } finally { setLoading(false); }
  }, [push, tenantId]);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  async function create() {
    if (!eventName.trim() || !webhookUrl.trim() || !tenantId) return;
    setCreating(true);
    try {
      await api.alerts.create({ tenant_id: tenantId, event_name: eventName.trim(), ...(dimsKey.trim() ? { dims_key: dimsKey.trim() } : {}), ...(metric === "absence" ? { metric, window_seconds: windowSeconds } : {}), threshold, webhook_url: webhookUrl.trim(), ...(destination !== "webhook" ? { destination } : {}) });
      setEventName("");
      setDimsKey("");
      setMetric("count"); setWindowSeconds(300);
      setThreshold(10);
      setWebhookUrl("");
      setDestination("webhook");
      push("success", "Alert rule created");
      await refresh();
    } catch (e) {
      push("error", e instanceof ApiError ? e.message : "Failed to create alert rule");
    } finally {
      setCreating(false);
    }
  }

  function beginEdit(rule: AlertRule) {
    setEditing(rule); setEventName(rule.EventName); setDimsKey(rule.DimsKey); setMetric(rule.Metric); setWindowSeconds(rule.WindowSeconds || 300); setThreshold(rule.Threshold); setWebhookUrl(rule.WebhookURL); setDestination(rule.Destination || "webhook");
  }

  function cancelEdit() {
    setEditing(null); setEventName(""); setDimsKey(""); setMetric("count"); setWindowSeconds(300); setThreshold(10); setWebhookUrl(""); setDestination("webhook");
  }

  async function saveEdit() {
    if (!editing || !eventName.trim() || !webhookUrl.trim()) return;
    setCreating(true);
    try {
      await api.alerts.update(editing.ID, { event_name: eventName.trim(), dims_key: dimsKey.trim(), metric, ...(metric === "absence" ? { window_seconds: windowSeconds } : {}), threshold, webhook_url: webhookUrl.trim(), ...(destination !== "webhook" ? { destination } : {}) });
      push("success", "Alert rule updated"); cancelEdit(); await refresh();
    } catch (e) { push("error", e instanceof ApiError ? e.message : "Failed to update alert rule"); }
    finally { setCreating(false); }
  }

  async function toggle(r: AlertRule) {
    try {
      await api.alerts.setEnabled(r.ID, !r.Enabled);
      push("success", `Rule ${r.Enabled ? "disabled" : "enabled"}`);
      await refresh();
    } catch (e) {
      push("error", e instanceof ApiError ? e.message : "Failed to update rule");
    }
  }

  async function remove(r: AlertRule) {
    try {
      await api.alerts.remove(r.ID);
      push("success", "Alert rule deleted");
      setToDelete(null);
      await refresh();
    } catch (e) {
      push("error", e instanceof ApiError ? e.message : "Failed to delete rule");
    }
  }

  const columns: Column<AlertRule>[] = [
    { key: "event", header: "Event", render: (r) => <span className="font-medium text-fg">{r.EventName}</span>, sortValue: (r) => r.EventName },
    { key: "threshold", header: "Threshold", render: (r) => <span className="text-fg-muted">{r.Metric} &ge; {r.Threshold}</span> },
    { key: "window", header: "Window", render: (r) => r.Metric === "count" ? <span className="text-xs text-fg-muted">lifetime</span> : <span className="text-xs text-fg-muted">{Math.round((r.WindowSeconds || 300) / 60)} min</span>, sortValue: (r) => r.WindowSeconds || 0 },
    { key: "dimensions", header: "Dimensions", render: (r) => <span className="text-xs text-fg-muted">{r.DimsKey || "all dimensions"}</span>, sortValue: (r) => r.DimsKey },
    { key: "webhook", header: "Webhook", render: (r) => <span className="truncate text-xs text-fg-muted">{r.WebhookURL}</span> },
    { key: "destination", header: "Destination", render: (r) => <Badge>{r.Destination || "webhook"}</Badge>, sortValue: (r) => r.Destination || "webhook" },
    {
      key: "status",
      header: "Status",
      render: (r) => (
        <Badge tone={r.Enabled ? "success" : "neutral"}>{r.Enabled ? "enabled" : "disabled"}</Badge>
      ),
    },
    {
      key: "fired",
      header: "Last fired",
      render: (r) => (r.LastFiredAt ? new Date(r.LastFiredAt * 1000).toLocaleString() : "never"),
    },
    {
      key: "actions",
      header: "",
      render: (r) => (
        <div className="flex justify-end gap-2">
          <Button variant="secondary" onClick={() => beginEdit(r)}>
            Edit
          </Button>
          <Button variant="secondary" onClick={() => toggle(r)}>
            {r.Enabled ? "Disable" : "Enable"}
          </Button>
          <Button variant="danger" onClick={() => setToDelete(r)}>
            Delete
          </Button>
        </div>
      ),
    },
  ];

  return (
    <div>
      <h1 className="text-xl font-semibold text-fg">Alerts</h1>
      <p className="mt-1 text-sm text-fg-muted">
        Threshold rules checked against materialized event counts and dispatched to a webhook. Evaluation is a 1-minute poll, not
        real-time push (see phases.md Phase 8).
      </p>
      <details className="mt-4 rounded-lg border border-border bg-surface-2 px-4 py-3 text-sm"><summary className="cursor-pointer font-medium text-fg">Alert help</summary><p className="mt-2 text-fg-muted">Rules are evaluated by the server polling job. Enable only tested webhook destinations; delivery failures are retried and eventually appear in Dead Letters.</p></details>

      <Card className="mt-6">
        <CardHeader title="New rule" />
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
            <Label>Event name</Label>
            <Combobox value={eventName} onChange={setEventName} options={eventNameOptions} placeholder={metric === "conversion" ? "landing,purchase" : "error"} />
            {metric === "conversion" && <p className="mt-1 text-[11px] text-fg-muted">Enter the ordered sequence as comma-separated event names.</p>}
          </div>
          <div className="w-36">
            <Label>Rule type</Label>
            <Select value={metric} onChange={(e) => setMetric(e.target.value)}><option value="count">Count</option><option value="absence">Absence</option><option value="conversion">Conversion</option><option value="anomaly">Anomaly (z-score)</option></Select>
          </div>
          {(metric === "absence" || metric === "conversion") && <div className="w-36"><Label>Window (seconds)</Label><Input type="number" min={30} value={windowSeconds} onChange={(e) => setWindowSeconds(Math.max(30, Number(e.target.value) || 30))} /></div>}
          <div className="w-28">
            <Label>Threshold (count)</Label>
            <Input type="number" min={1} value={threshold} onChange={(e) => setThreshold(Math.max(1, Number(e.target.value) || 1))} />
          </div>
          <div className="min-w-[180px]">
            <Label>Dimensions key (optional)</Label>
            <Input value={dimsKey} onChange={(e) => setDimsKey(e.target.value)} placeholder="country=ir" />
          </div>
          <div className="min-w-[220px] flex-1">
            <Label>Destination</Label>
            <Select value={destination} onChange={(e) => setDestination(e.target.value)}><option value="webhook">Generic webhook</option><option value="slack">Slack webhook</option><option value="pagerduty">PagerDuty Events API</option><option value="email">Email HTTP gateway</option></Select>
          </div>
          <div className="min-w-[220px] flex-1">
            <Label>Webhook URL</Label>
            <Input value={webhookUrl} onChange={(e) => setWebhookUrl(e.target.value)} placeholder="https://example.com/hook" />
          </div>
          {editing ? <><Button onClick={saveEdit} loading={creating} disabled={!eventName.trim() || !webhookUrl.trim() || !tenantId}>Save rule</Button><Button variant="secondary" onClick={cancelEdit}>Cancel</Button></> : <Button onClick={create} loading={creating} disabled={!eventName.trim() || !webhookUrl.trim() || !tenantId}>Create rule</Button>}
        </div>
      </Card>

      <Card className="mt-6">
        <CardHeader title={`Alert rules (${rules.length})`} />
        <DataTable columns={columns} rows={rules} rowKey={(r) => r.ID} emptyMessage="No alert rules yet for this tenant." loading={loading} serverSide onQueryChange={(query) => void refresh(query)} totalRows={totalRules} />
      </Card>

      <ConfirmDialog
        open={!!toDelete}
        title="Delete alert rule?"
        description="This permanently removes the rule. It will stop being evaluated and will no longer dispatch webhooks."
        confirmLabel="Delete rule"
        danger
        onConfirm={() => toDelete && remove(toDelete)}
        onCancel={() => setToDelete(null)}
      />
    </div>
  );
}
