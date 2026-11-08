"use client";
import { Card, CardHeader } from "@/components/Card";

export default function SettingsPage() {
  return (
    <div>
      <h1 className="text-xl font-semibold text-fg">Settings</h1>
      <p className="mt-1 text-sm text-fg-muted">Current backend configuration (read-only — there is no settings-update API yet).</p>

      <Card className="mt-6">
        <CardHeader title="Backend" />
        <dl className="grid grid-cols-1 gap-4 p-5 sm:grid-cols-2">
          <div>
            <dt className="text-xs font-medium uppercase tracking-wide text-fg-muted">Admin auth</dt>
            <dd className="mt-1 text-sm text-fg">x-admin-key header, checked by the Go backend's AdminAuth middleware</dd>
          </div>
          <div>
            <dt className="text-xs font-medium uppercase tracking-wide text-fg-muted">Storage</dt>
            <dd className="mt-1 text-sm text-fg">SQLite via GORM (dev) — see CLAUDE.md</dd>
          </div>
          <div>
            <dt className="text-xs font-medium uppercase tracking-wide text-fg-muted">Retention</dt>
            <dd className="mt-1 text-sm text-fg">Configured via RETENTION_HOURS env var on the server; not yet editable from this panel</dd>
          </div>
          <div>
            <dt className="text-xs font-medium uppercase tracking-wide text-fg-muted">Rate limiting / sampling</dt>
            <dd className="mt-1 text-sm text-fg">Enforced server-side (per-IP + per-tenant token bucket); no admin API to change limits yet</dd>
          </div>
        </dl>
      </Card>

      <Card className="mt-6 border-dashed">
        <div className="p-5 text-sm text-fg-muted">
          Editable settings (retention, sampling, rate limits via API) are not implemented — there is no backend
          endpoint for it yet. We're not shipping controls here that wouldn't actually do anything.
        </div>
      </Card>
    </div>
  );
}
