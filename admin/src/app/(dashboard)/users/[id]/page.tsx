"use client";

import { useEffect, useMemo, useState } from "react";
import { api, Event, Session } from "@/lib/api";
import { Card, CardHeader, StatCard } from "@/components/Card";
import { EntityLink } from "@/components/EntityLink";
import { JsonView } from "@/components/JsonView";
import Link from "next/link";
import { useTimeRange } from "@/components/TimeRangeContext";

function decode(value: string) { try { return decodeURIComponent(value); } catch { return value; } }
function deviceLocation(session: Session) {
  const persisted = [session.City, session.Region, session.Country].filter((item) => item && item.trim()).join(", ");
  if (persisted) return persisted;
  try {
    const data = JSON.parse(session.DeviceInfo || "{}") as Record<string, unknown>;
    const nested = (data.location || data.geo) as Record<string, unknown> | undefined;
    const get = (key: string) => data[key] ?? nested?.[key];
    return [get("city"), get("region"), get("country")].filter((item) => item != null && String(item).trim()).map(String).join(", ") || "Location not provided";
  } catch { return "Location not provided"; }
}

export default function UserDetailPage({ params }: { params: Promise<{ id: string }> }) {
  const [ip, setIp] = useState(""), [userAgent, setUserAgent] = useState(""), [lifetime, setLifetime] = useState(false);
  const [sessions, setSessions] = useState<Session[]>([]), [events, setEvents] = useState<Event[]>([]), [error, setError] = useState("");
  const { from, to, hours } = useTimeRange();
  useEffect(() => { void params.then(({ id }) => { const [nextIP, nextUA] = decode(id).split("|"); setIp(nextIP || ""); setUserAgent(nextUA || ""); }); }, [params]);
  useEffect(() => { if (!ip) return; const range = lifetime ? {} : { from_ts: from, to_ts: to }; api.sessions.list({ filter: ip, limit: 100, offset: 0, sort: "last_seen", dir: "desc", ...range }).then((page) => { const matches = page.items.filter((session) => session.IP === ip && session.UserAgent === userAgent); setSessions(matches); return Promise.all(matches.map((session) => api.events.list({ session_id: session.ID, limit: 500, offset: 0, sort: "desc", ...range }))); }).then((lists) => setEvents(lists.flat())).catch((err) => setError(err instanceof Error ? err.message : "Failed to load user")); }, [ip, userAgent, from, to, lifetime]);
  const totalEvents = events.length, totalViews = events.filter((event) => event.EventName === "page_view").length;
  const activeMinutes = sessions.reduce((sum, session) => {
    const started = new Date(session.StartedAt).getTime(), ended = new Date(session.LastSeen).getTime();
    const windowStart = lifetime ? started : Math.max(started, from), windowEnd = lifetime ? ended : Math.min(ended, to);
    return sum + Math.max(0, (windowEnd - windowStart) / 60000);
  }, 0);
  const locations = [...new Set(sessions.map(deviceLocation))], online = sessions.some((session) => new Date(session.LastSeen).getTime() >= Date.now() - 5 * 60 * 1000);
  const eventNames = useMemo(() => [...events.reduce((map, event) => map.set(event.EventName, (map.get(event.EventName) || 0) + 1), new Map<string, number>())].sort((a, b) => b[1] - a[1]), [events]);
  const behavior = eventNames.slice(0, 5).map(([name, count]) => `${name} (${count})`).join(" · ") || "No behavior recorded";
  return <div><div className="flex items-center justify-between gap-4"><div><Link href="/users" className="text-sm text-accent hover:underline">← Users</Link><h1 className="mt-2 text-xl font-semibold text-fg">User profile</h1><p className="mt-1 text-sm text-fg-muted">Behavior, activity, and statistics for the selected time window.</p></div><div className="flex items-center gap-3"><label className="flex items-center gap-2 text-xs text-fg-muted"><input type="checkbox" checked={lifetime} onChange={(event) => setLifetime(event.target.checked)} /> Lifetime</label><span className={`rounded-full px-3 py-1 text-xs ${online ? "bg-success/10 text-success" : "bg-surface-2 text-fg-muted"}`}>{online ? "Online" : "Offline"}</span></div></div>{error && <p className="mt-4 rounded border border-danger/30 bg-danger/5 p-3 text-sm text-danger">{error}</p>}<p className="mt-3 text-xs text-fg-muted">{lifetime ? "Lifetime history" : `Global range: last ${hours} hours`}</p><div className="mt-6 grid gap-4 sm:grid-cols-2 lg:grid-cols-4"><StatCard label="Total events" value={totalEvents.toLocaleString()} hint="Filtered event count"/><StatCard label="Page views" value={totalViews.toLocaleString()} hint="page_view events"/><StatCard label="Active time" value={`${activeMinutes.toFixed(1)} min`} hint="Session duration sum"/><StatCard label="Sessions" value={sessions.length.toLocaleString()} hint="Matching fingerprint"/></div><Card className="mt-6"><CardHeader title="Behavior analysis"/><div className="grid gap-3 p-5 text-sm sm:grid-cols-3"><div><p className="text-xs text-fg-muted">Top behavior</p><p className="mt-1 text-fg">{behavior}</p></div><div><p className="text-xs text-fg-muted">Location</p><p className="mt-1 text-fg">{locations[0] || "Location not provided"}</p></div><div><p className="text-xs text-fg-muted">Last seen</p><p className="mt-1 text-fg">{sessions[0]?.LastSeen ? new Date(sessions[0].LastSeen).toLocaleString() : "—"}</p></div></div></Card><Card className="mt-6"><CardHeader title="Identity"/><div className="grid gap-4 p-5 text-sm sm:grid-cols-2"><div><p className="text-xs text-fg-muted">IP address</p><p className="mt-1 font-mono text-fg">{ip || "Not provided"}</p></div><div><p className="text-xs text-fg-muted">User agent</p><p className="mt-1 break-all text-fg">{userAgent || "Not provided"}</p></div></div></Card><Card className="mt-6"><CardHeader title="Event breakdown"/><div className="divide-y divide-border">{eventNames.map(([name, count]) => <div key={name} className="flex justify-between px-5 py-3 text-sm"><EntityLink kind="event" id={name} label={name}/><span className="text-fg-muted">{count.toLocaleString()}</span></div>)}{!eventNames.length && <p className="p-5 text-sm text-fg-muted">No events found.</p>}</div></Card><Card className="mt-6"><CardHeader title="Session activity"/><div className="divide-y divide-border">{sessions.map((session) => <div key={session.ID} className="flex items-center justify-between gap-3 px-5 py-3 text-sm"><div><EntityLink kind="session" id={session.ID}/><p className="text-xs text-fg-muted">{deviceLocation(session)} · {session.UserAgent || "User agent not provided"}</p></div><span className="text-xs text-fg-muted">{new Date(session.LastSeen).toLocaleString()}</span></div>)}</div></Card><Card className="mt-6"><CardHeader title="Recent events"/><div className="divide-y divide-border">{events.slice(0, 50).map((event) => <div key={event.EventID} className="flex items-center justify-between gap-3 px-5 py-3 text-sm"><div><EntityLink kind="event" id={event.EventID} label={event.EventName}/><p className="text-xs text-fg-muted">Session <EntityLink kind="session" id={event.SessionID}/></p></div><span className="text-xs text-fg-muted">{new Date(event.ClientTS).toLocaleString()}</span><JsonView value={event.Payload}/></div>)}</div></Card></div>;
}
