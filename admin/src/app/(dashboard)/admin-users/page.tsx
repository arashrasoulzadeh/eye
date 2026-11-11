"use client";

import { useCallback, useEffect, useState } from "react";
import { api, AdminUser, ApiError } from "@/lib/api";
import { Card, CardHeader } from "@/components/Card";
import { Button } from "@/components/Button";
import { ConfirmDialog } from "@/components/ConfirmDialog";
import { Input, Label, Select } from "@/components/Input";
import { DataTable, Column, TableQuery } from "@/components/DataTable";
import { Badge } from "@/components/Badge";
import { useToast } from "@/components/Toast";

const empty = { email: "", name: "", role: "viewer" as AdminUser["role"], status: "invited" as AdminUser["status"] };

export default function AdminUsersPage() {
  const [users, setUsers] = useState<AdminUser[]>([]), [total, setTotal] = useState(0), [query, setQuery] = useState<TableQuery>({ filter: "", sortKey: "created", sortDir: "desc", page: 0 });
  const [form, setForm] = useState(empty), [editing, setEditing] = useState<AdminUser | null>(null), [remove, setRemove] = useState<AdminUser | null>(null), [loading, setLoading] = useState(true), [saving, setSaving] = useState(false), [error, setError] = useState("");
  const { push } = useToast();
  const load = useCallback(async (next = query) => { setLoading(true); try { const result = await api.adminUsers.listPage({ filter: next.filter, sort: next.sortKey ?? "created", dir: next.sortDir, limit: 10, offset: next.page * 10 }); setUsers(result.items); setTotal(result.total); setError(""); } catch (e) { setError(e instanceof ApiError ? e.message : "Failed to load admin users"); } finally { setLoading(false); } }, [query]);
  useEffect(() => { void load(); }, [load]);
  function edit(user: AdminUser) { setEditing(user); setForm({ email: user.email, name: user.name, role: user.role, status: user.status }); }
  function reset() { setEditing(null); setForm(empty); }
  async function save() { if (!form.email.trim()) return; setSaving(true); try { if (editing) await api.adminUsers.update(editing.id, form); else await api.adminUsers.create(form); push("success", editing ? "Admin user updated" : "Admin user invited"); reset(); await load(); } catch (e) { push("error", e instanceof ApiError ? e.message : "Failed to save admin user"); } finally { setSaving(false); } }
  async function destroy() { if (!remove) return; try { await api.adminUsers.remove(remove.id); push("success", "Admin user removed"); setRemove(null); await load(); } catch (e) { push("error", e instanceof ApiError ? e.message : "Failed to remove admin user"); } }
  const columns: Column<AdminUser>[] = [
    { key: "name", header: "User", render: (u) => <div><p className="font-medium text-fg">{u.name || u.email}</p><p className="text-xs text-fg-muted">{u.email}</p></div>, sortValue: (u) => u.name || u.email, filterValue: (u) => `${u.name} ${u.email}` },
    { key: "role", header: "Role", render: (u) => <Badge>{u.role}</Badge>, sortValue: (u) => u.role },
    { key: "status", header: "Status", render: (u) => <Badge tone={u.status === "active" ? "success" : "neutral"}>{u.status}</Badge>, sortValue: (u) => u.status },
    { key: "created", header: "Created", render: (u) => new Date(u.created_at * 1000).toLocaleString(), sortValue: (u) => u.created_at },
    { key: "actions", header: "Actions", render: (u) => <div className="flex justify-end gap-2"><Button variant="secondary" onClick={() => edit(u)}>Edit</Button><Button variant="danger" onClick={() => setRemove(u)}>Remove</Button></div> },
  ];
  return <div><div className="flex items-end justify-between gap-4"><div><h1 className="text-xl font-semibold text-fg">Admin Users</h1><p className="mt-1 text-sm text-fg-muted">Manage workspace identities, roles, invitations, and suspended access.</p></div><Button variant="secondary" onClick={() => void load()} loading={loading}>Refresh</Button></div><details className="mt-4 rounded-lg border border-border bg-surface-2 px-4 py-3 text-sm"><summary className="cursor-pointer font-medium text-fg">Admin user help</summary><p className="mt-2 text-fg-muted">This directory stores authorization state. Authentication is handled by the configured admin key or OIDC gateway; no passwords or tokens are stored here.</p></details>{error && <p className="mt-4 rounded border border-danger/30 bg-danger/5 p-3 text-sm text-danger">{error}</p>}<Card className="mt-6"><CardHeader title={editing ? "Edit admin user" : "Invite admin user"}/><div className="grid gap-3 p-5 md:grid-cols-4"><div><Label>Email</Label><Input type="email" value={form.email} onChange={e => setForm({ ...form, email: e.target.value })} placeholder="admin@example.com"/></div><div><Label>Name</Label><Input value={form.name} onChange={e => setForm({ ...form, name: e.target.value })} placeholder="Alex Admin"/></div><div><Label>Role</Label><Select value={form.role} onChange={e => setForm({ ...form, role: e.target.value as AdminUser["role"] })}><option value="owner">Owner</option><option value="admin">Admin</option><option value="analyst">Analyst</option><option value="viewer">Viewer</option></Select></div><div><Label>Status</Label><Select value={form.status} onChange={e => setForm({ ...form, status: e.target.value as AdminUser["status"] })}><option value="invited">Invited</option><option value="active">Active</option><option value="suspended">Suspended</option></Select></div></div><div className="flex gap-2 px-5 pb-5"><Button onClick={() => void save()} loading={saving} disabled={!form.email.trim()}>{editing ? "Save user" : "Create invitation"}</Button>{editing && <Button variant="secondary" onClick={reset}>Cancel</Button>}</div></Card><Card className="mt-6"><CardHeader title={`Users (${total})`}/><DataTable columns={columns} rows={users} rowKey={u => u.id} emptyMessage="No admin users match this filter." loading={loading} serverSide totalRows={total} pageSize={10} defaultSortKey="created" onQueryChange={(next) => { setQuery(next); void load(next); }}/></Card><ConfirmDialog open={!!remove} title="Remove admin user?" description="This removes the workspace authorization record. It does not delete the identity from your OIDC provider." confirmLabel="Remove user" danger onConfirm={() => void destroy()} onCancel={() => setRemove(null)}/></div>;
}
