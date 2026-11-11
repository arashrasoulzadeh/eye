"use client";
import { useCallback, useEffect, useState } from "react";
import { api, AlertRule, Tenant, ApiError } from "@/lib/api";
import { Card, CardHeader } from "@/components/Card";
import { Button } from "@/components/Button";
import { Input, Label, Select } from "@/components/Input";
import { Combobox, ComboOption } from "@/components/Combobox";
import { DataTable, Column } from "@/components/DataTable";
import { Badge } from "@/components/Badge";
import { ConfirmDialog } from "@/components/ConfirmDialog";
import { useToast } from "@/components/Toast";

export default function AlertsPage() {
  const [tenants, setTenants] = useState<Tenant[]>([]);
  const [tenantId, setTenantId] = useState("");
  const [rules, setRules] = useState<AlertRule[]>([]);
  const [eventNameOptions, setEventNameOptions] = useState<ComboOption[]>([]);

  const [eventName, setEventName] = useState("");
  const [threshold, setThreshold] = useState(10);
  const [webhookUrl, setWebhookUrl] = useState("");
  const [creating, setCreating] = useState(false);
  const [toDelete, setToDelete] = useState<AlertRule | null>(null);
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

  const refresh = useCallback(async () => {
    if (!tenantId) return;
    try {
      setRules(await api.alerts.list(tenantId));
    } catch (e) {
      push("error", e instanceof ApiError ? e.message : "Failed to load alert rules");
    }
  }, [push, tenantId]);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  async function create() {
    if (!eventName.trim() || !webhookUrl.trim() || !tenantId) return;
    setCreating(true);
    try {
      await api.alerts.create({ tenant_id: tenantId, event_name: eventName.trim(), threshold, webhook_url: webhookUrl.trim() });
      setEventName("");
      setThreshold(10);
      setWebhookUrl("");
      push("success", "Alert rule created");
      await refresh();
    } catch (e) {
      push("error", e instanceof ApiError ? e.message : "Failed to create alert rule");
    } finally {
      setCreating(false);
    }
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
    { key: "webhook", header: "Webhook", render: (r) => <span className="truncate text-xs text-fg-muted">{r.WebhookURL}</span> },
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
            <Combobox value={eventName} onChange={setEventName} options={eventNameOptions} placeholder="error" />
          </div>
          <div className="w-28">
            <Label>Threshold (count)</Label>
            <Input type="number" min={1} value={threshold} onChange={(e) => setThreshold(Math.max(1, Number(e.target.value) || 1))} />
          </div>
          <div className="min-w-[220px] flex-1">
            <Label>Webhook URL</Label>
            <Input value={webhookUrl} onChange={(e) => setWebhookUrl(e.target.value)} placeholder="https://example.com/hook" />
          </div>
          <Button onClick={create} loading={creating} disabled={!eventName.trim() || !webhookUrl.trim() || !tenantId}>
            Create rule
          </Button>
        </div>
      </Card>

      <Card className="mt-6">
        <CardHeader title={`Alert rules (${rules.length})`} />
        <DataTable columns={columns} rows={rules} rowKey={(r) => r.ID} emptyMessage="No alert rules yet for this tenant." />
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
