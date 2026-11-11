"use client";
import { useCallback, useEffect, useState } from "react";
import { api, ApiError, RuntimeSettings } from "@/lib/api";
import { Card, CardHeader } from "@/components/Card";
import { Button } from "@/components/Button";
import { Input, Label } from "@/components/Input";
import { useToast } from "@/components/Toast";

export default function SettingsPage() {
  const [settings, setSettings] = useState<RuntimeSettings | null>(null);
  const [retentionHours, setRetentionHours] = useState(0);
  const [rateLimitPerSecond, setRateLimitPerSecond] = useState(0);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const { push } = useToast();

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const s = await api.settings.get();
      setSettings(s);
      setRetentionHours(s.RetentionHours);
      setRateLimitPerSecond(s.RateLimitPerSecond);
    } catch (e) {
      push("error", e instanceof ApiError ? e.message : "Failed to load settings");
    } finally {
      setLoading(false);
    }
  }, [push]);

  useEffect(() => {
    void load();
  }, [load]);

  async function save() {
    setSaving(true);
    try {
      const s = await api.settings.update({ retention_hours: retentionHours, rate_limit_per_second: rateLimitPerSecond });
      setSettings(s);
      push("success", "Settings saved — the retention job and ingest rate limit pick this up immediately, no restart needed");
    } catch (e) {
      push("error", e instanceof ApiError ? e.message : "Failed to save settings");
    } finally {
      setSaving(false);
    }
  }

  return (
    <div>
      <h1 className="text-xl font-semibold text-fg">Settings</h1>
      <p className="mt-1 text-sm text-fg-muted">
        Retention and the ingest rate limit are DB-backed and live: a save here takes effect on the retention job&apos;s
        next tick and the very next ingest request, not just on the next server restart.
      </p>
      <details className="mt-4 rounded-lg border border-border bg-surface-2 px-4 py-3 text-sm">
        <summary className="cursor-pointer font-medium text-fg">Settings help</summary>
        <p className="mt-2 text-fg-muted">Changes are persisted by the API and picked up by the retention worker and ingest limiter without restarting the server.</p>
      </details>

      <Card className="mt-6">
        <CardHeader title="Retention &amp; rate limit" />
        <div className="flex flex-wrap items-end gap-4 p-5">
          <div className="w-48">
            <Label>Retention (hours)</Label>
            <Input
              type="number"
              min={1}
              value={retentionHours}
              onChange={(e) => setRetentionHours(Math.max(1, Number(e.target.value) || 1))}
              disabled={loading}
            />
          </div>
          <div className="w-48">
            <Label>Ingest rate limit (events/sec)</Label>
            <Input
              type="number"
              min={1}
              value={rateLimitPerSecond}
              onChange={(e) => setRateLimitPerSecond(Math.max(1, Number(e.target.value) || 1))}
              disabled={loading}
            />
          </div>
          <Button onClick={save} loading={saving} disabled={loading}>
            Save
          </Button>
        </div>
        {settings && (
          <p className="px-5 pb-5 text-xs text-fg-muted">
            Last updated: {settings.UpdatedAt ? new Date(settings.UpdatedAt * 1000).toLocaleString() : "never (showing defaults)"}
          </p>
        )}
      </Card>

      <Card className="mt-6 border-dashed">
        <div className="p-5 text-sm text-fg-muted">
          Adaptive sampling (Phase 8) is driven by live system load, not a manual setting here — see the dashboard&apos;s
          sample-rate metric. There is no admin control for it, by design.
        </div>
      </Card>
    </div>
  );
}
