"use client";

import { useEffect, useMemo, useState } from "react";
import { api, Event, Session, Tenant, ApiError } from "@/lib/api";
import { Card, CardHeader, StatCard } from "@/components/Card";
import { Combobox } from "@/components/Combobox";
import { LineTrend, BarBreakdown } from "@/components/Chart";
import { Badge } from "@/components/Badge";
import { EntityLink } from "@/components/EntityLink";
import { JsonView } from "@/components/JsonView";

type UserRow = { id: string; sessions: number; events: number; lastSeen: string; online: boolean; browser: string; location: string; ip: string; userAgent: string };

function detail(session: Session, key: string): string {
  try { const value = JSON.parse(session.DeviceInfo || "{}")[key]; return value == null || value === "" ? "Unknown" : String(value); } catch { return "Unknown"; }
}
function browser(ua: string): string {
  if (/edg\//i.test(ua)) return "Edge";
  if (/chrome|crios/i.test(ua)) return "Chrome";
  if (/firefox|fxios/i.test(ua)) return "Firefox";
  if (/safari/i.test(ua) && !/chrome/i.test(ua)) return "Safari";
  if (/android/i.test(ua)) return "Android WebView";
  return ua ? "Other" : "Unknown";
}

export default function UsersPage() {
  const [tenants, setTenants] = useState<Tenant[]>([]), [tenant, setTenant] = useState(""), [sessions, setSessions] = useState<Session[]>([]), [events, setEvents] = useState<Event[]>([]), [loading, setLoading] = useState(true), [error, setError] = useState("");
  async function load() {
    if (!tenant) return;
    setLoading(true); setError("");
    try {
      const [sessionPage, recent] = await Promise.all([api.sessions.list({ tenant_id: tenant, limit: 100, offset: 0, sort: "last_seen", dir: "desc" }), api.events.list({ tenant_id: tenant, limit: 50, offset: 0, sort: "desc" })]);
      setSessions(sessionPage.items); setEvents(recent);
    } catch (e) { setError(e instanceof ApiError ? e.message : "Failed to load users"); } finally { setLoading(false); }
  }
  useEffect(() => { api.tenants.list().then((items) => { setTenants(items); if (items[0]) setTenant(items[0].ID); }).catch((e) => setError(e.message)); }, []);
  useEffect(() => { void load(); const timer = window.setInterval(() => void load(), 15000); return () => window.clearInterval(timer); }, [tenant]);

  const users = useMemo<UserRow[]>(() => {
    const grouped = new Map<string, UserRow>();
    sessions.forEach((session) => {
      const id = `${session.IP || "unknown"}|${session.UserAgent || "unknown"}`;
      const current = grouped.get(id) || { id, sessions: 0, events: 0, lastSeen: session.LastSeen, online: false, browser: browser(session.UserAgent), location: [detail(session, "city"), detail(session, "country")].filter((x) => x !== "Unknown").join(", ") || "Unknown", ip: session.IP || "Unknown", userAgent: session.UserAgent || "Unknown" };
      current.sessions += 1; current.events += session.EventCount || 0; current.lastSeen = new Date(current.lastSeen).getTime() > new Date(session.LastSeen).getTime() ? current.lastSeen : session.LastSeen; current.online = new Date(current.lastSeen).getTime() >= Date.now() - 5 * 60 * 1000;
      grouped.set(id, current);
    });
    return [...grouped.values()].sort((a, b) => new Date(b.lastSeen).getTime() - new Date(a.lastSeen).getTime());
  }, [sessions]);
  const browserChart = useMemo(() => [...users.reduce((m, user) => m.set(user.browser, (m.get(user.browser) || 0) + 1), new Map<string, number>())].map(([label, value]) => ({ label, value })), [users]);
  const locationChart = useMemo(() => [...users.reduce((m, user) => m.set(user.location, (m.get(user.location) || 0) + 1), new Map<string, number>())].sort((a, b) => b[1] - a[1]).slice(0, 8).map(([label, value]) => ({ label, value })), [users]);
  const activityChart = useMemo(() => { const counts = new Map<string, number>(); events.forEach((event) => { const label = new Date(event.ClientTS).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" }); counts.set(label, (counts.get(label) || 0) + 1); }); return [...counts.entries()].map(([label, value]) => ({ label, value })); }, [events]);
  const copy = (value: unknown) => void navigator.clipboard?.writeText(typeof value === "string" ? value : JSON.stringify(value, null, 2));

  return <div><div className="flex flex-wrap items-end justify-between gap-4"><div><h1 className="text-xl font-semibold text-fg">Users</h1><p className="mt-1 text-sm text-fg-muted">Tenant-scoped user intelligence built from session fingerprints. Data refreshes every 15 seconds.</p></div><Combobox value={tenant} onChange={setTenant} options={tenants.map((item) => ({ value: item.ID, label: item.Name }))} placeholder="Select tenant" /></div><details className="mt-4 rounded-lg border border-border bg-surface-2 px-4 py-3 text-sm"><summary className="cursor-pointer font-medium text-fg">Users data help</summary><p className="mt-2 text-fg-muted">A user is currently grouped by IP address and user agent because the event contract has no explicit user_id. Location comes from session DeviceInfo when supplied. Online means seen within the last five minutes.</p></details>{error && <p className="mt-4 rounded border border-danger/30 bg-danger/5 p-3 text-sm text-danger">{error}</p>}<div className="mt-6 grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-5"><StatCard label="Users" value={users.length.toLocaleString()} hint="Unique fingerprints"/><StatCard label="Online" value={users.filter((user) => user.online).length.toLocaleString()} hint="Last 5 minutes"/><StatCard label="Sessions" value={sessions.length.toLocaleString()} hint="Latest 100 loaded"/><StatCard label="Events" value={users.reduce((sum, user) => sum + user.events, 0).toLocaleString()} hint="Across loaded sessions"/><StatCard label="Browsers" value={new Set(users.map((user) => user.browser)).size.toLocaleString()} hint="Detected families"/></div><div className="mt-6 grid gap-6 lg:grid-cols-3"><Card><CardHeader title="Activity stream"/><div className="p-4"><LineTrend data={activityChart}/></div></Card><Card><CardHeader title="Browsers"/><div className="p-4"><BarBreakdown data={browserChart}/></div></Card><Card><CardHeader title="Locations"/><div className="p-4"><BarBreakdown data={locationChart}/></div></Card></div><Card className="mt-6"><CardHeader title={`User profiles (${users.length})`} action={<span className="text-xs text-fg-muted">{loading ? "Refreshing…" : "Live snapshot"}</span>}/>{!users.length && !loading ? <p className="p-5 text-sm text-fg-muted">No users found for this tenant.</p> : <div className="overflow-x-auto"><table className="w-full text-left text-sm"><thead><tr className="border-b border-border text-xs uppercase text-fg-muted"><th className="px-5 py-3">Status</th><th className="px-5 py-3">Location</th><th className="px-5 py-3">Browser / IP</th><th className="px-5 py-3">Sessions</th><th className="px-5 py-3">Events</th><th className="px-5 py-3">Last seen</th><th className="px-5 py-3 text-right">Actions</th></tr></thead><tbody>{users.map((user) => <tr key={user.id} className="border-b border-border last:border-0 hover:bg-surface-2/60"><td className="px-5 py-3"><Badge tone={user.online ? "success" : "neutral"}>{user.online ? "Online" : "Offline"}</Badge></td><td className="px-5 py-3"><div className="font-medium text-fg">{user.location}</div></td><td className="px-5 py-3"><div>{user.browser}</div><div className="font-mono text-xs text-fg-muted">{user.ip}</div></td><td className="px-5 py-3">{user.sessions}</td><td className="px-5 py-3">{user.events}</td><td className="px-5 py-3 text-xs text-fg-muted">{user.lastSeen ? new Date(user.lastSeen).toLocaleString() : "—"}</td><td className="px-5 py-3 text-right"><button className="text-xs text-accent" onClick={() => copy(user.userAgent)}>Copy UA</button></td></tr>)}</tbody></table></div>}</Card><Card className="mt-6"><CardHeader title="Recent tenant events"/><div className="divide-y divide-border">{events.length ? events.map((event, index) => <div key={event.EventID || `${event.EventName}-${event.ClientTS}-${index}`} className="flex items-center justify-between gap-3 px-5 py-3 text-sm"><div><EntityLink kind="event" id={event.EventID} label={event.EventName}/><p className="text-xs text-fg-muted">Session <EntityLink kind="session" id={event.SessionID}/></p></div><span className="text-xs text-fg-muted">{new Date(event.ClientTS).toLocaleString()}</span><JsonView value={event.Payload}/></div>) : <p className="p-5 text-sm text-fg-muted">No recent events.</p>}</div></Card></div>;
}
