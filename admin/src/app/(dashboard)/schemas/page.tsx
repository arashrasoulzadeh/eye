"use client";
import { useCallback, useEffect, useState } from "react";
import { api, Tenant, Schema, ApiError } from "@/lib/api";
import { Card, CardHeader } from "@/components/Card";
import { Button } from "@/components/Button";
import { Input, Label, Select, Textarea } from "@/components/Input";
import { DataTable, Column, TableQuery } from "@/components/DataTable";
import { Badge } from "@/components/Badge";
import { useToast } from "@/components/Toast";
import { useTenant } from "@/components/TenantContext";

export default function SchemasPage() {
  const { tenantId: globalTenant } = useTenant();
  const [tenants, setTenants] = useState<Tenant[]>([]);
  const [tenantId, setTenantId] = useState("");
  useEffect(() => { if (globalTenant && globalTenant !== tenantId) setTenantId(globalTenant); }, [globalTenant, tenantId]);
  const [schemas, setSchemas] = useState<Schema[]>([]);
  const [totalSchemas, setTotalSchemas] = useState(0);
  const [name, setName] = useState("");
  const [definition, setDefinition] = useState('{\n  "type": "object",\n  "properties": {}\n}');
  const [jsonError, setJsonError] = useState("");
  const [saving, setSaving] = useState(false), [loading, setLoading] = useState(true);
  const { push } = useToast();

  useEffect(() => {
    api.tenants.list().then((ts) => {
      setTenants(ts);
      if (ts[0]) setTenantId(ts[0].ID);
    }).catch((e) => push("error", e instanceof ApiError ? e.message : "Failed to load tenants"));
  }, [push]);

  const loadSchemas = useCallback(async (id: string, query: TableQuery = { filter: "", sortKey: "version", sortDir: "desc", page: 0 }) => {
    if (!id) return;
    setLoading(true);
    try {
      const page = await api.schemas.listPage(id, { filter: query.filter, dir: query.sortDir, limit: 10, offset: query.page * 10 }); setSchemas(page.items); setTotalSchemas(page.total);
    } catch (e) {
      push("error", e instanceof ApiError ? e.message : "Failed to load schemas");
    } finally { setLoading(false); }
  }, [push]);

  useEffect(() => {
    void loadSchemas(tenantId);
  }, [loadSchemas, tenantId]);

  async function register() {
    try {
      JSON.parse(definition);
    } catch {
      setJsonError("Definition must be valid JSON");
      return;
    }
    setJsonError("");
    setSaving(true);
    try {
      await api.schemas.register(tenantId, name.trim(), definition);
      push("success", `Registered "${name.trim()}"`);
      setName("");
      await loadSchemas(tenantId);
    } catch (e) {
      push("error", e instanceof ApiError ? e.message : "Failed to register schema");
    } finally {
      setSaving(false);
    }
  }

  const columns: Column<Schema>[] = [
    { key: "name", header: "Name", render: (s) => <span className="font-medium text-fg">{s.Name}</span>, sortValue: (s) => s.Name },
    { key: "version", header: "Version", render: (s) => <Badge tone="accent">v{s.Version}</Badge>, sortValue: (s) => s.Version },
    { key: "definition", header: "Definition", render: (s) => <pre className="max-w-xs overflow-auto text-xs text-fg-muted">{s.Definition}</pre> },
  ];

  return (
    <div>
      <h1 className="text-xl font-semibold text-fg">Schemas</h1>
      <p className="mt-1 text-sm text-fg-muted">Register and version event payload schemas per tenant.</p>
      <details className="mt-4 rounded-lg border border-border bg-surface-2 px-4 py-3 text-sm"><summary className="cursor-pointer font-medium text-fg">Schema help</summary><p className="mt-2 text-fg-muted">Schemas are immutable versions. Registering the same name creates a new version; existing event history is not rewritten.</p></details>

      <Card className="mt-6">
        <CardHeader title="Register schema" />
        <div className="space-y-3 p-5">
          <div>
            <Label>Tenant</Label>
            <Select value={tenantId} onChange={(e) => setTenantId(e.target.value)}>
              {tenants.length === 0 && <option value="">No tenants yet</option>}
              {tenants.map((t) => (
                <option key={t.ID} value={t.ID}>
                  {t.Name}
                </option>
              ))}
            </Select>
          </div>
          <div>
            <Label>Schema name</Label>
            <Input value={name} onChange={(e) => setName(e.target.value)} placeholder="click_event" />
          </div>
          <div>
            <Label>Definition (JSON)</Label>
            <Textarea rows={6} value={definition} onChange={(e) => setDefinition(e.target.value)} />
            {jsonError && <p className="mt-1 text-xs text-danger">{jsonError}</p>}
          </div>
          <Button onClick={register} loading={saving} disabled={!tenantId || !name.trim()}>
            Register version
          </Button>
        </div>
      </Card>

      <Card className="mt-6">
        <CardHeader title={`Versions for this tenant (${schemas.length})`} />
        <DataTable columns={columns} rows={schemas} rowKey={(s) => s.ID} emptyMessage="No schemas registered for this tenant yet." loading={loading} serverSide onQueryChange={(query) => void loadSchemas(tenantId, query)} totalRows={totalSchemas} />
      </Card>
    </div>
  );
}
