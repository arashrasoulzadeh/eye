"use client";
import { useCallback, useEffect, useState } from "react";
import { api, Tenant, ApiError } from "@/lib/api";
import { Card, CardHeader } from "@/components/Card";
import { Button } from "@/components/Button";
import { Input, Label } from "@/components/Input";
import { DataTable, Column, TableQuery } from "@/components/DataTable";
import { ConfirmDialog } from "@/components/ConfirmDialog";
import { CopyButton } from "@/components/CopyButton";
import { useToast } from "@/components/Toast";

export default function TenantsPage() {
  const [tenants, setTenants] = useState<Tenant[]>([]);
  const [totalTenants, setTotalTenants] = useState(0);
  const [name, setName] = useState("");
  const [creating, setCreating] = useState(false);
  const [toDelete, setToDelete] = useState<Tenant | null>(null);
  const [rotated, setRotated] = useState<{ tenant: Tenant; key: string } | null>(null);
  const { push } = useToast();

  const refresh = useCallback(async (query: TableQuery = { filter: "", sortKey: "created", sortDir: "desc", page: 0 }) => {
    try {
      const page = await api.tenants.listPage({ filter: query.filter, sort: query.sortKey ?? "created", dir: query.sortDir, limit: 10, offset: query.page * 10 }); setTenants(page.items); setTotalTenants(page.total);
    } catch (e) {
      push("error", e instanceof ApiError ? e.message : "Failed to load tenants");
    }
  }, [push]);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  async function create() {
    if (!name.trim()) return;
    setCreating(true);
    try {
      const apiKey = crypto.randomUUID().replace(/-/g, "");
      await api.tenants.create(name.trim(), apiKey);
      setName("");
      push("success", `Tenant "${name.trim()}" created`);
      await refresh();
    } catch (e) {
      push("error", e instanceof ApiError ? e.message : "Failed to create tenant");
    } finally {
      setCreating(false);
    }
  }

  async function remove(t: Tenant) {
    try {
      await api.tenants.remove(t.ID);
      push("success", `Tenant "${t.Name}" deleted`);
      setToDelete(null);
      await refresh();
    } catch (e) {
      push("error", e instanceof ApiError ? e.message : "Failed to delete tenant");
    }
  }

  async function rotate(t: Tenant) {
    try {
      const { api_key } = await api.tenants.rotateKey(t.ID);
      setRotated({ tenant: t, key: api_key });
    } catch (e) {
      push("error", e instanceof ApiError ? e.message : "Failed to rotate key");
    }
  }

  const columns: Column<Tenant>[] = [
    {
      key: "name",
      header: "Name",
      render: (t) => (
        <a href={`/tenants/${t.ID}`} className="font-medium text-fg hover:text-accent hover:underline">
          {t.Name}
        </a>
      ),
      sortValue: (t) => t.Name,
    },
    { key: "id", header: "ID", render: (t) => <CopyButton value={t.ID} label={t.ID.slice(0, 10) + "…"} /> },
    { key: "created", header: "Created", render: (t) => (t.CreatedAt ? new Date(t.CreatedAt).toLocaleString() : "—"), sortValue: (t) => t.CreatedAt },
    {
      key: "actions",
      header: "",
      render: (t) => (
        <div className="flex justify-end gap-2">
          <Button variant="secondary" onClick={() => rotate(t)}>
            Rotate key
          </Button>
          <Button variant="danger" onClick={() => setToDelete(t)}>
            Delete
          </Button>
        </div>
      ),
    },
  ];

  return (
    <div>
      <h1 className="text-xl font-semibold text-fg">Tenants</h1>
      <p className="mt-1 text-sm text-fg-muted">Create tenants, rotate API keys, configure quotas, and remove tenant records.</p>
      <details className="mt-4 rounded-lg border border-border bg-surface-2 px-4 py-3 text-sm">
        <summary className="cursor-pointer font-medium text-fg">Tenant management help</summary>
        <p className="mt-2 text-fg-muted">API keys are shown only when created or rotated. Open a tenant to review usage, set monthly quotas, inspect sessions, and view event activity.</p>
      </details>

      <Card className="mt-6">
        <CardHeader title="New tenant" />
        <div className="flex flex-wrap items-end gap-3 p-5">
          <div className="min-w-[220px] flex-1">
            <Label>Name</Label>
            <Input value={name} onChange={(e) => setName(e.target.value)} placeholder="acme-corp" onKeyDown={(e) => e.key === "Enter" && create()} />
          </div>
          <Button onClick={create} loading={creating} disabled={!name.trim()}>
            Create tenant
          </Button>
        </div>
      </Card>

      <Card className="mt-6">
        <CardHeader title={`All tenants (${tenants.length})`} />
        <DataTable columns={columns} rows={tenants} rowKey={(t) => t.ID} emptyMessage="No tenants yet — create one above." serverSide onQueryChange={(query) => void refresh(query)} totalRows={totalTenants} />
      </Card>

      <ConfirmDialog
        open={!!toDelete}
        title={`Delete "${toDelete?.Name}"?`}
        description="This permanently removes the tenant. Its existing sessions and events are not deleted automatically."
        confirmLabel="Delete tenant"
        danger
        onConfirm={() => toDelete && remove(toDelete)}
        onCancel={() => setToDelete(null)}
      />

      {rotated && (
        <ConfirmDialog
          open
          title={`New API key for ${rotated.tenant.Name}`}
          description={`${rotated.key}\n\nThis is shown once and cannot be recovered later — copy it now.`}
          confirmLabel="Done"
          onConfirm={() => setRotated(null)}
          onCancel={() => setRotated(null)}
        />
      )}
    </div>
  );
}
