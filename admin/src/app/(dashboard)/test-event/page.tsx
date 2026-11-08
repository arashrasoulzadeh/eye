"use client";
import { useEffect, useState } from "react";
import { api, Tenant, Session, Event, ApiError } from "@/lib/api";
import { Card, CardHeader } from "@/components/Card";
import { Button } from "@/components/Button";
import { Input, Label, Select, Textarea } from "@/components/Input";
import { Badge } from "@/components/Badge";
import { CopyButton } from "@/components/CopyButton";
import { useToast } from "@/components/Toast";

export default function TestEventPage() {
  const [tenants, setTenants] = useState<Tenant[]>([]);
  const [tenantId, setTenantId] = useState("");
  const [sessions, setSessions] = useState<Session[]>([]);
  const [sessionId, setSessionId] = useState("");
  const [creatingSession, setCreatingSession] = useState(false);

  const [eventName, setEventName] = useState("test_click");
  const [payload, setPayload] = useState('{\n  "button": "signup"\n}');
  const [dims, setDims] = useState('{\n  "platform": "web"\n}');
  const [jsonError, setJsonError] = useState("");
  const [sending, setSending] = useState(false);
  const [result, setResult] = useState<{ accepted: number; rejected: number } | null>(null);

  const [recent, setRecent] = useState<Event[]>([]);
  const { push } = useToast();

  useEffect(() => {
    api.tenants.list().then((ts) => {
      setTenants(ts);
      if (ts[0]) setTenantId(ts[0].ID);
    });
  }, []);

  useEffect(() => {
    if (!tenantId) return;
    api.sessions.list().then((all) => {
      const mine = all.filter((s) => s.TenantID === tenantId);
      setSessions(mine);
      setSessionId(mine[0]?.ID || "");
    });
  }, [tenantId]);

  async function createSession() {
    if (!tenantId) return;
    setCreatingSession(true);
    try {
      const s = await api.sessions.create(tenantId);
      setSessions((prev) => [s, ...prev]);
      setSessionId(s.ID);
      push("success", "New session created");
    } catch (e) {
      push("error", e instanceof ApiError ? e.message : "Failed to create session");
    } finally {
      setCreatingSession(false);
    }
  }

  async function refreshRecent() {
    if (!tenantId) return;
    try {
      const list = await api.events.list({ tenant_id: tenantId, session_id: sessionId || undefined });
      setRecent(list);
    } catch {
      // non-fatal; leave previous list in place
    }
  }

  useEffect(() => {
    refreshRecent();
  }, [tenantId, sessionId]);

  async function send() {
    if (!sessionId) {
      push("error", "Create or select a session first");
      return;
    }
    let parsedPayload: unknown, parsedDims: unknown;
    try {
      parsedPayload = JSON.parse(payload);
      parsedDims = JSON.parse(dims);
    } catch {
      setJsonError("Payload and dims must be valid JSON");
      return;
    }
    setJsonError("");
    setSending(true);
    try {
      const res = await api.ingest.send([
        {
          event_id: crypto.randomUUID(),
          session_id: sessionId,
          event_name: eventName,
          client_ts: Date.now(),
          payload: parsedPayload,
          dims: parsedDims,
        },
      ]);
      setResult(res);
      push(res.accepted > 0 ? "success" : "error", `Accepted ${res.accepted}, rejected ${res.rejected}`);
      await refreshRecent();
    } catch (e) {
      push("error", e instanceof ApiError ? e.message : "Failed to send event");
    } finally {
      setSending(false);
    }
  }

  return (
    <div>
      <h1 className="text-xl font-semibold text-fg">Test Event</h1>
      <p className="mt-1 text-sm text-fg-muted">Manually send an event through /ingest and watch it land.</p>

      <div className="mt-6 grid grid-cols-1 gap-6 lg:grid-cols-2">
        <Card>
          <CardHeader title="Send an event" />
          <div className="space-y-3 p-5">
            <div>
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

            <div>
              <Label>Session</Label>
              <div className="flex gap-2">
                <Select value={sessionId} onChange={(e) => setSessionId(e.target.value)} disabled={sessions.length === 0}>
                  {sessions.length === 0 && <option value="">No sessions — create one</option>}
                  {sessions.map((s) => (
                    <option key={s.ID} value={s.ID}>
                      {s.ID.slice(0, 16)}… ({s.EventCount} events)
                    </option>
                  ))}
                </Select>
                <Button variant="secondary" onClick={createSession} loading={creatingSession} disabled={!tenantId}>
                  New session
                </Button>
              </div>
            </div>

            <div>
              <Label>Event name</Label>
              <Input value={eventName} onChange={(e) => setEventName(e.target.value)} placeholder="test_click" />
            </div>

            <div>
              <Label>Payload (JSON)</Label>
              <Textarea rows={4} value={payload} onChange={(e) => setPayload(e.target.value)} />
            </div>

            <div>
              <Label>Dims (JSON)</Label>
              <Textarea rows={3} value={dims} onChange={(e) => setDims(e.target.value)} />
            </div>

            {jsonError && <p className="text-xs text-danger">{jsonError}</p>}

            <Button onClick={send} loading={sending} disabled={!sessionId || !eventName.trim()} className="w-full">
              Send event
            </Button>

            {result && (
              <div className="flex gap-2 pt-1">
                <Badge tone="success">accepted: {result.accepted}</Badge>
                <Badge tone="danger">rejected: {result.rejected}</Badge>
              </div>
            )}
          </div>
        </Card>

        <Card>
          <CardHeader
            title={`Recent events for this tenant (${recent.length})`}
            action={
              <Button variant="ghost" onClick={refreshRecent}>
                Refresh
              </Button>
            }
          />
          <div className="max-h-[560px] divide-y divide-border overflow-y-auto">
            {recent.length === 0 && <p className="px-5 py-8 text-center text-sm text-fg-muted">No events sent yet.</p>}
            {recent.map((e) => (
              <div key={e.EventID} className="px-5 py-3">
                <div className="flex items-center justify-between">
                  <span className="text-sm font-medium text-fg">{e.EventName}</span>
                  <span className="text-xs text-fg-muted">{e.IngestTS ? new Date(e.IngestTS).toLocaleTimeString() : ""}</span>
                </div>
                <div className="mt-1 flex items-center gap-2 text-xs text-fg-muted">
                  <span>session</span>
                  <CopyButton value={e.SessionID} label={e.SessionID.slice(0, 10) + "…"} />
                </div>
                <pre className="mt-1 overflow-auto rounded-md bg-surface-2 p-2 text-xs text-fg">{e.Payload}</pre>
              </div>
            ))}
          </div>
        </Card>
      </div>
    </div>
  );
}
