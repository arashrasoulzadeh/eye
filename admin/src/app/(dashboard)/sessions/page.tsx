"use client";
import { useEffect, useState } from "react";
import { api, Session, ApiError } from "@/lib/api";
import { Card, CardHeader } from "@/components/Card";
import { DataTable, Column } from "@/components/DataTable";
import { EntityLink } from "@/components/EntityLink";
import { Badge } from "@/components/Badge";
import { useToast } from "@/components/Toast";

export default function SessionsPage() {
  const [sessions, setSessions] = useState<Session[]>([]);
  const { push } = useToast();

  useEffect(() => {
    api.sessions
      .list()
      .then(setSessions)
      .catch((e: ApiError) => push("error", e.message));
  }, []);

  const columns: Column<Session>[] = [
    { key: "id", header: "Session ID", render: (s) => <EntityLink kind="session" id={s.ID} truncate={12} /> },
    { key: "tenant", header: "Tenant ID", render: (s) => <EntityLink kind="tenant" id={s.TenantID} truncate={12} /> },
    { key: "events", header: "Events", render: (s) => <Badge tone="accent">{s.EventCount}</Badge>, sortValue: (s) => s.EventCount },
    { key: "started", header: "Started", render: (s) => (s.StartedAt ? new Date(s.StartedAt).toLocaleString() : "—"), sortValue: (s) => s.StartedAt },
    { key: "last_seen", header: "Last seen", render: (s) => (s.LastSeen ? new Date(s.LastSeen).toLocaleString() : "—"), sortValue: (s) => s.LastSeen },
  ];

  return (
    <div>
      <h1 className="text-xl font-semibold text-fg">Sessions</h1>
      <p className="mt-1 text-sm text-fg-muted">Sessions created by tenants, including test sessions from the Test Event page.</p>

      <Card className="mt-6">
        <CardHeader title={`All sessions (${sessions.length})`} />
        <DataTable columns={columns} rows={sessions} rowKey={(s) => s.ID} emptyMessage="No sessions yet." />
      </Card>
    </div>
  );
}
