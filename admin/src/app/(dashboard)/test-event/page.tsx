"use client";
import { useCallback, useEffect, useState } from "react";
import { api, Tenant, Session, Event, ApiError } from "@/lib/api";
import { Card, CardHeader } from "@/components/Card";
import { Button } from "@/components/Button";
import { Input, Label, Select, Textarea } from "@/components/Input";
import { Badge } from "@/components/Badge";
import { EntityLink } from "@/components/EntityLink";
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
  const [count, setCount] = useState(1);
  const [randomizePerEvent, setRandomizePerEvent] = useState(false);
  const [jsonError, setJsonError] = useState("");
  const [sending, setSending] = useState(false);
  const [result, setResult] = useState<{ accepted: number; rejected: number } | null>(null);

  const EVENT_NAMES = ["test_click", "page_view", "signup", "purchase", "button_click", "form_submit", "video_play", "error"];
  const BUTTONS = ["signup", "checkout", "learn_more", "upgrade", "share"];
  const PLATFORMS = ["web", "ios", "android", "macos", "windows"];
  const pick = <T,>(arr: T[]) => arr[Math.floor(Math.random() * arr.length)];

  function randomEvent() {
    return {
      name: pick(EVENT_NAMES),
      payload: { button: pick(BUTTONS), value: Math.floor(Math.random() * 1000), ts: Date.now() },
      dims: { platform: pick(PLATFORMS), locale: pick(["en-US", "en-GB", "de-DE", "fr-FR"]) },
    };
  }

  function randomize() {
    const r = randomEvent();
    setEventName(r.name);
    setPayload(JSON.stringify(r.payload, null, 2));
    setDims(JSON.stringify(r.dims, null, 2));
  }

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

  const refreshRecent = useCallback(async () => {
    if (!tenantId) return;
    try {
      const list = await api.events.list({ tenant_id: tenantId, session_id: sessionId || undefined });
      setRecent(list);
    } catch {
      // non-fatal; leave previous list in place
    }
  }, [sessionId, tenantId]);

  useEffect(() => {
    void refreshRecent();
  }, [refreshRecent]);

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
      const n = Math.max(1, Math.min(1000, count || 1));
      const events = Array.from({ length: n }, () => {
        const r = randomizePerEvent && n > 1 ? randomEvent() : null;
        return {
          event_id: crypto.randomUUID(),
          session_id: sessionId,
          event_name: r ? r.name : eventName,
          client_ts: Date.now(),
          payload: r ? r.payload : parsedPayload,
          dims: r ? r.dims : parsedDims,
        };
      });
      const res = await api.ingest.send(events);
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

            <div className="flex items-end justify-between gap-2">
              <div className="flex-1">
                <Label>Event name</Label>
                <Input value={eventName} onChange={(e) => setEventName(e.target.value)} placeholder="test_click" />
              </div>
              <Button variant="secondary" onClick={randomize} type="button">
                🎲 Randomize
              </Button>
            </div>

            <div>
              <Label>Payload (JSON)</Label>
              <Textarea rows={4} value={payload} onChange={(e) => setPayload(e.target.value)} />
            </div>

            <div>
              <Label>Dims (JSON)</Label>
              <Textarea rows={3} value={dims} onChange={(e) => setDims(e.target.value)} />
            </div>

            <div>
              <Label>Count (events to send)</Label>
              <Input
                type="number"
                min={1}
                max={1000}
                value={count}
                onChange={(e) => setCount(Math.max(1, Math.min(1000, Number(e.target.value) || 1)))}
              />
            </div>

            {count > 1 && (
              <label className="flex items-center gap-2 text-sm text-fg-muted">
                <input
                  type="checkbox"
                  checked={randomizePerEvent}
                  onChange={(e) => setRandomizePerEvent(e.target.checked)}
                  className="h-4 w-4 rounded border-border accent-accent"
                />
                Randomize each event individually
              </label>
            )}

            {jsonError && <p className="text-xs text-danger">{jsonError}</p>}

            <Button onClick={send} loading={sending} disabled={!sessionId || !eventName.trim()} className="w-full">
              Send {count > 1 ? `${count} events` : "event"}
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
                  <EntityLink kind="session" id={e.SessionID} />
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
