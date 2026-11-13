"use client";

import { useEffect, useMemo, useState } from "react";
import { api, Event, Session } from "@/lib/api";
import { Card, CardHeader, StatCard } from "@/components/Card";
import { EntityLink } from "@/components/EntityLink";
import { JsonView } from "@/components/JsonView";
import Link from "next/link";

function decode(value: string) { try { return decodeURIComponent(value); } catch { return value; } }
function deviceLocation(session: Session) {
  try {
    const data = JSON.parse(session.DeviceInfo || "{}") as Record<string, unknown>;
    const nested = (data.location || data.geo) as Record<string, unknown> | undefined;
    const get = (key: string) => data[key] ?? nested?.[key];
    return [get("city"), get("region"), get("country")].filter((item) => item != null && String(item).trim()).map(String).join(", ") || "Location not provided";
  } catch { return "Location not provided"; }
}

export default function UserDetailPage({ params }: { params: Promise<{ id: string }> }) {
  const [ip, setIp] = useState(""), [userAgent, setUserAgent] = useState("");
  const [sessions, setSessions] = useState<Session[]>([]), [events, setEvents] = useState<Event[]>([]), [error, setError] = useState("");
  useEffect(() => { void params.then(({ id }) => { const [nextIP, nextUA] = decode(id).split("|"); setIp(nextIP || ""); setUserAgent(nextUA || ""); }); }, [params]);
  useEffect(() => { if (!ip) return; api.sessions.list({ filter: ip, limit: 100, offset: 0, sort: "last_seen", dir: "desc" }).then((page) => { const matches = page.items.filter((session) => session.IP === ip && session.UserAgent === userAgent); setSessions(matches); return Promise.all(matches.map((session) => api.events.list({ session_id: session.ID, limit: 100, offset: 0, sort: "desc" }))); }).then((lists) => setEvents(lists.flat())).catch((err) => setError(err instanceof Error ? err.message : "Failed to load user")); }, [ip, userAgent]);
  const totalEvents = sessions.reduce((sum, session) => sum + (session.EventCount || 0), 0);
  const locations = [...new Set(sessions.map(deviceLocation))];
  const online = sessions.some((session) => new Date(session.LastSeen).getTime() >= Date.now() - 5 * 60 * 1000);
  const eventNames = useMemo(() => [...events.reduce((map, event) => map.set(event.EventName, (map.get(event.EventName) || 0) + 1), new Map<string, number>())].sort((a, b) => b[1] - a[1]), [events]);
  return <div><div className="flex items-center justify-between gap-4"><div><Link href="/users" className="text-sm text-accent hover:underline">← Users</Link><h1 className="mt-2 text-xl font-semibold text-fg">User profile</h1><p className="mt-1 text-sm text-fg-muted">Fingerprint details, sessions, location, and activity.</p></div><span className={`rounded-full px-3 py-1 text-xs ${online ? "bg-success/10 text-success" : "bg-surface-2 text-fg-muted"}`}>{online ? "Online" : "Offline"}</span></div>{error && <p className="mt-4 rounded border border-danger/30 bg-danger/5 p-3 text-sm text-danger">{error}</p>}<div className="mt-6 grid gap-4 sm:grid-cols-2 lg:grid-cols-4"><StatCard label="Sessions" value={sessions.length.toLocaleString()} hint="Matching fingerprint"/><StatCard label="Events" value={totalEvents.toLocaleString()} hint="Recorded events"/><StatCard label="Location" value={locations[0] || "Location not provided"} hint="From device data"/><StatCard label="Last seen" value={sessions[0]?.LastSeen ? new Date(sessions[0].LastSeen).toLocaleString() : "—"} hint="Most recent session"/></div><Card className="mt-6"><CardHeader title="Identity"/><div className="grid gap-4 p-5 text-sm sm:grid-cols-2"><div><p className="text-xs text-fg-muted">IP address</p><p className="mt-1 font-mono text-fg">{ip || "Not provided"}</p></div><div><p className="text-xs text-fg-muted">User agent</p><p className="mt-1 break-all text-fg">{userAgent || "Not provided"}</p></div></div></Card><Card className="mt-6"><CardHeader title="Event breakdown"/><div className="divide-y divide-border">{eventNames.map(([name, count]) => <div key={name} className="flex justify-between px-5 py-3 text-sm"><EntityLink kind="event" id={name} label={name}/><span className="text-fg-muted">{count.toLocaleString()}</span></div>)}{!eventNames.length && <p className="p-5 text-sm text-fg-muted">No events found.</p>}</div></Card><Card className="mt-6"><CardHeader title="Session activity"/><div className="divide-y divide-border">{sessions.map((session) => <div key={session.ID} className="flex items-center justify-between gap-3 px-5 py-3 text-sm"><div><EntityLink kind="session" id={session.ID}/><p className="text-xs text-fg-muted">{deviceLocation(session)} · {session.UserAgent || "User agent not provided"}</p></div><span className="text-xs text-fg-muted">{new Date(session.LastSeen).toLocaleString()}</span></div>)}</div></Card><Card className="mt-6"><CardHeader title="Recent events"/><div className="divide-y divide-border">{events.slice(0, 50).map((event) => <div key={event.EventID} className="flex items-center justify-between gap-3 px-5 py-3 text-sm"><div><EntityLink kind="event" id={event.EventID} label={event.EventName}/><p className="text-xs text-fg-muted">Session <EntityLink kind="session" id={event.SessionID}/></p></div><span className="text-xs text-fg-muted">{new Date(event.ClientTS).toLocaleString()}</span><JsonView value={event.Payload}/></div>)}</div></Card></div>;
}
