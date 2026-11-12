export type Tenant = { ID: string; Name: string; APIKey: string; CreatedAt: string; MonthlyEventLimit?: number; MonthlyStorageLimit?: number };
export type TenantPage = { items: Tenant[]; total: number; limit: number; offset: number };
export type Session = {
  ID: string;
  TenantID: string;
  StartedAt: string;
  LastSeen: string;
  EventCount: number;
  UserAgent: string;
  IP: string;
  DeviceInfo: string;
};
export type SessionPage = { items: Session[]; total: number; limit: number; offset: number };
export type Event = {
  EventID: string;
  SessionID: string;
  TenantID: string;
  EventName: string;
  ClientTS: number;
  IngestTS: string;
  Payload: string;
  Dims: string;
  Source: string;
  IsAggregated: boolean;
  AggregationID: string;
};
export type EventPage = { items: Event[]; total: number; limit: number; offset: number };
export type Schema = { ID: string; TenantID: string; Name: string; Version: number; Definition: string; CreatedAt: number };
export type SchemaPage = { items: Schema[]; total: number; limit: number; offset: number };
export type Metrics = { events: number; sessions: number; tenants: number };
export type DashboardCache = {
  events: number;
  sessions: number;
  tenants: number;
  events_by_hour: Array<{ label: string; value: number }>;
  events_by_tenant: Array<{ label: string; value: number }>;
  cached_at: number;
};
export type RuntimeSettings = { ID: number; RetentionHours: number; RateLimitPerSecond: number; UpdatedAt: number };
export type Cohort = { ID: string; TenantID: string; Name: string; EventName: string; MinCount: number; CreatedAt: number; UpdatedAt: number };
export type CohortPage = { items: Cohort[]; total: number; limit: number; offset: number };
export type AlertRule = {
  ID: string;
  TenantID: string;
  EventName: string;
  DimsKey: string;
  Metric: string;
  Threshold: number;
  WindowSeconds: number;
  WebhookURL: string;
  Destination: "webhook" | "slack" | "pagerduty" | "email";
  Enabled: boolean;
  LastFiredAt: number;
  LastFiredCount: number;
  CreatedAt: number;
  UpdatedAt: number;
};
export type AlertRulePage = { items: AlertRule[]; total: number; limit: number; offset: number };
export type Question = { ID: string; TenantID: string; Name: string; EventName: string; EventNames: string; MatchMode: "any" | "all"; ChartType: "line" | "bar"; RangeMode: "relative" | "absolute"; RelativeDays: number; FromTS: number; ToTS: number; CreatedAt: number; UpdatedAt: number };
export type GraphResult = { question: Question; from_ts: number; to_ts: number; points: Array<{ label: string; value: number }> };
export type EventOption = { value: string; count: number; last_ts: number };
export type TenantUsage = { tenant_id: string; period_start: number; period_end: number; event_count: number; payload_bytes: number; storage_bytes: number; event_limit: number; storage_limit: number; warnings?: string[] };
export type AdminUser = { id: string; email: string; name: string; role: "owner" | "admin" | "analyst" | "viewer"; status: "active" | "invited" | "suspended"; created_at: number; updated_at: number };
export type AdminTeam = { id: string; name: string; created_at: number; updated_at: number };
export type AdminTeamPage = { items: AdminTeam[]; total: number; limit: number; offset: number };
export type AdminTeamMember = { team_id: string; user_id: string; role: string; created_at: number };

class ApiError extends Error {
  status: number;
  constructor(message: string, status: number) {
    super(message);
    this.status = status;
  }
}

function handleUnauthorized() {
  // Fire-and-forget: clear the admin_key cookie, then bounce to /login.
  // A plain redirect (not router.push) guarantees this runs even though
  // this module has no access to the Next.js router.
  fetch("/api/logout", { method: "POST" }).finally(() => {
    if (typeof window !== "undefined") {
      // This module is outside React, so use a hard navigation after clearing
      // the cookie instead of importing a router into the API client.
      // eslint-disable-next-line @next/next/no-location-assign-relative-destination
      window.location.href = "/login";
    }
  });
}

async function req<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`/api/proxy/${path}`, {
    ...init,
    headers: { "content-type": "application/json", ...(init?.headers || {}) },
    cache: "no-store",
  });
  if (res.status === 401) {
    handleUnauthorized();
    throw new ApiError("Session expired — redirecting to login", 401);
  }
  if (!res.ok) {
    const text = await res.text().catch(() => "");
    throw new ApiError(text || res.statusText, res.status);
  }
  const text = await res.text();
  return (text ? JSON.parse(text) : null) as T;
}

let tenantsCache: { value: Tenant[]; expires: number } | null = null;
let tenantsRequest: Promise<Tenant[]> | null = null;
let sessionsCache: { value: SessionPage; expires: number } | null = null;
let sessionsRequest: Promise<SessionPage> | null = null;
function invalidateTenants() { tenantsCache = null; tenantsRequest = null; }
function invalidateSessions() { sessionsCache = null; sessionsRequest = null; }

export const api = {
  metrics: () => req<Metrics>("metrics"),

  dashboardCache: () => req<DashboardCache>("dashboard-cache"),

  tenants: {
    list: (params: { filter?: string; sort?: string; dir?: string; limit?: number; offset?: number } = {}) => { if (!Object.keys(params).length && tenantsCache && tenantsCache.expires > Date.now()) return Promise.resolve(tenantsCache.value); const query = Object.keys(params).length ? `?${new URLSearchParams(Object.fromEntries(Object.entries({ filter: params.filter ?? "", sort: params.sort ?? "created", dir: params.dir ?? "desc", limit: params.limit ?? 100, offset: params.offset ?? 0 }).map(([key, value]) => [key, String(value)])))}` : ""; if (!query && tenantsRequest) return tenantsRequest; const request = req<Tenant[]>(`tenants${query}`); if (query) return request; tenantsRequest = request.then((value) => { tenantsCache = { value, expires: Date.now() + 30_000 }; tenantsRequest = null; return value; }).catch((error) => { tenantsRequest = null; throw error; }); return tenantsRequest; },
    listPage: (params: { filter?: string; sort?: string; dir?: string; limit?: number; offset?: number } = {}) => req<TenantPage>(`tenants?page=true&filter=${encodeURIComponent(params.filter ?? "")}&sort=${params.sort ?? "created"}&dir=${params.dir ?? "desc"}&limit=${params.limit ?? 10}&offset=${params.offset ?? 0}`),
    clearCache: invalidateTenants,
    create: async (name: string, apiKey: string) => { const result = await req<Tenant>("tenants", { method: "POST", body: JSON.stringify({ Name: name, APIKey: apiKey }) }); invalidateTenants(); return result; },
    remove: async (id: string) => { const result = await req<void>(`tenants/${id}`, { method: "DELETE" }); invalidateTenants(); return result; },
    rotateKey: (id: string) => req<{ api_key: string }>(`tenants/${id}/rotate-key`, { method: "POST" }),
    usage: (id: string) => req<TenantUsage>(`tenant-usage/${encodeURIComponent(id)}`),
    setQuota: (id: string, eventLimit: number, storageLimit: number) => req<void>(`tenants/${encodeURIComponent(id)}/quota`, { method: "PUT", body: JSON.stringify({ event_limit: eventLimit, storage_limit: storageLimit }) }),
  },

  sessions: {
    list: (params: { limit?: number; offset?: number; filter?: string; sort?: string; dir?: string } = {}) => { const isDefault = !Object.keys(params).length; if (isDefault && sessionsCache && sessionsCache.expires > Date.now()) return Promise.resolve(sessionsCache.value); if (isDefault && sessionsRequest) return sessionsRequest; const request = req<SessionPage>(`sessions?${new URLSearchParams(Object.fromEntries(Object.entries({ limit: params.limit ?? 10, offset: params.offset ?? 0, filter: params.filter ?? "", sort: params.sort ?? "last_seen", dir: params.dir ?? "desc" }).map(([key, value]) => [key, String(value)])))}`); if (!isDefault) return request; sessionsRequest = request.then((value) => { sessionsCache = { value, expires: Date.now() + 10_000 }; sessionsRequest = null; return value; }).catch((error) => { sessionsRequest = null; throw error; }); return sessionsRequest; },
    get: (id: string) => req<Session>(`sessions?session_id=${encodeURIComponent(id)}`),
    create: (tenantId: string) =>
      req<Session>("sessions", { method: "POST", body: JSON.stringify({ tenant_id: tenantId }) }).then((value) => { invalidateSessions(); return value; }),
    clearCache: invalidateSessions,
  },

  events: {
    get: (id: string) => req<Event>(`events/${encodeURIComponent(id)}`),
    // Hits /query (not /events) because that's the endpoint that actually
    // supports the from_ts/to_ts time-range filter (added for Phase 6);
    // /events only ever supported event_name/tenant_id/session_id.
    list: (
      filters: {
        event_name?: string;
        tenant_id?: string;
        session_id?: string;
        from_ts?: number;
        to_ts?: number;
        sort?: "desc" | "asc";
        limit?: number;
        offset?: number;
      } = {}
    ) => {
      // Newest-first by default — the admin events page always wants the
      // most recent activity on top; pass sort: "asc" explicitly to opt out.
      const { sort = "desc", ...rest } = filters;
      const qs = new URLSearchParams(
        Object.entries({ ...rest, sort })
          .filter(([, v]) => v !== undefined && v !== "")
          .map(([k, v]) => [k, String(v)])
      );
      const suffix = qs.toString() ? `?${qs}` : "";
      return req<Event[]>(`query${suffix}`);
    },
    listPage: (filters: { event_name?: string; tenant_id?: string; session_id?: string; from_ts?: number; to_ts?: number; sort?: "desc" | "asc"; limit?: number; offset?: number } = {}) => { const qs = new URLSearchParams(Object.entries({ ...filters, page: "true" }).filter(([, v]) => v !== undefined && v !== "").map(([k, v]) => [k, String(v)])); return req<EventPage>(`query?${qs}`); },
  },
  eventOptions: (tenantId: string, query = "") => req<EventOption[]>(`event-options?tenant_id=${encodeURIComponent(tenantId)}&q=${encodeURIComponent(query)}`),
  slo: () => req<{ target: number; availability: number; requests: number; errors: number; error_budget: number; error_budget_remaining: number; within_slo: boolean }>("slo"),
  analytics: {
    funnel: (tenantId: string, steps: string[], fromTs?: number, toTs?: number) => req<Array<{ event_name: string; sessions: number; conversion_rate: number }>>("analytics/funnel", { method: "POST", body: JSON.stringify({ tenant_id: tenantId, steps, from_ts: fromTs, to_ts: toTs }) }),
    suggestedFunnel: (tenantId: string, fromTs?: number, toTs?: number) => req<Array<{ position: number; event_name: string; title: string; sessions: number }>>(`analytics/funnel/suggested?tenant_id=${encodeURIComponent(tenantId)}${fromTs ? `&from_ts=${fromTs}` : ""}${toTs ? `&to_ts=${toTs}` : ""}`),
    retention: (tenantId: string, granularity: "daily" | "weekly" = "daily") => req<Array<{ cohort_period: string; cohort_size: number; active_by_n: number[]; retained_pct: number[] }>>(`analytics/retention?tenant_id=${encodeURIComponent(tenantId)}&granularity=${granularity}`),
    views: (tenantId: string) => req<Array<{ id: string; tenant_id: string; name: string; query: string; created_at: number; updated_at: number }>>(`analytics/views?tenant_id=${encodeURIComponent(tenantId)}`),
    saveView: (tenantId: string, name: string, query: string) => req<unknown>(`analytics/views?tenant_id=${encodeURIComponent(tenantId)}`, { method: "POST", body: JSON.stringify({ name, query }) }),
    deleteView: (tenantId: string, id: string) => req<void>(`analytics/views/${encodeURIComponent(id)}?tenant_id=${encodeURIComponent(tenantId)}`, { method: "DELETE" }),
  },

  schemas: {
    list: (tenantId: string, params: { filter?: string; dir?: string } = {}) => req<Schema[]>(`schemas?tenant_id=${encodeURIComponent(tenantId)}&filter=${encodeURIComponent(params.filter ?? "")}&dir=${params.dir ?? "desc"}`),
    listPage: (tenantId: string, params: { filter?: string; dir?: string; limit?: number; offset?: number } = {}) => req<SchemaPage>(`schemas?tenant_id=${encodeURIComponent(tenantId)}&page=true&filter=${encodeURIComponent(params.filter ?? "")}&dir=${params.dir ?? "desc"}&limit=${params.limit ?? 10}&offset=${params.offset ?? 0}`),
    register: (tenantId: string, name: string, definition: string) =>
      req<Schema>("schemas", {
        method: "POST",
        body: JSON.stringify({ tenant_id: tenantId, name, definition }),
      }),
  },

  cohorts: {
    list: (tenantId: string, params: { filter?: string; dir?: string } = {}) => req<Cohort[]>(`cohorts?tenant_id=${encodeURIComponent(tenantId)}&filter=${encodeURIComponent(params.filter ?? "")}&dir=${params.dir ?? "desc"}`),
    listPage: (tenantId: string, params: { filter?: string; dir?: string; limit?: number; offset?: number } = {}) => req<CohortPage>(`cohorts?tenant_id=${encodeURIComponent(tenantId)}&page=true&filter=${encodeURIComponent(params.filter ?? "")}&dir=${params.dir ?? "desc"}&limit=${params.limit ?? 10}&offset=${params.offset ?? 0}`),
    create: (tenantId: string, name: string, eventName: string, minCount: number) =>
      req<Cohort>("cohorts", {
        method: "POST",
        body: JSON.stringify({ tenant_id: tenantId, name, event_name: eventName, min_count: minCount }),
      }),
    materialize: (id: string) => req<{ members: number }>(`cohorts/${id}/materialize`, { method: "POST" }),
    remove: (id: string) => req<void>(`cohorts/${id}`, { method: "DELETE" }),
    members: (id: string) => req<string[]>(`cohorts/${id}/members`),
  },

  alerts: {
    list: (tenantId: string, params: { filter?: string; dir?: string } = {}) => req<AlertRule[]>(`alerts?tenant_id=${encodeURIComponent(tenantId)}&filter=${encodeURIComponent(params.filter ?? "")}&dir=${params.dir ?? "desc"}`),
    listPage: (tenantId: string, params: { filter?: string; dir?: string; limit?: number; offset?: number } = {}) => req<AlertRulePage>(`alerts?tenant_id=${encodeURIComponent(tenantId)}&page=true&filter=${encodeURIComponent(params.filter ?? "")}&dir=${params.dir ?? "desc"}&limit=${params.limit ?? 10}&offset=${params.offset ?? 0}`),
    create: (rule: { tenant_id: string; event_name: string; dims_key?: string; metric?: string; threshold: number; window_seconds?: number; webhook_url: string; destination?: string }) =>
      req<AlertRule>("alerts", { method: "POST", body: JSON.stringify(rule) }),
    update: (id: string, rule: { event_name: string; dims_key?: string; metric?: string; threshold: number; window_seconds?: number; webhook_url: string; destination?: string }) =>
      req<AlertRule>(`alerts/${encodeURIComponent(id)}`, { method: "PUT", body: JSON.stringify(rule) }),
    remove: (id: string) => req<void>(`alerts/${id}`, { method: "DELETE" }),
    setEnabled: (id: string, enabled: boolean) => req<AlertRule>(`alerts/${id}/${enabled ? "enable" : "disable"}`, { method: "POST" }),
  },

  questions: {
    list: (tenantId: string) => req<Question[]>(`questions?tenant_id=${encodeURIComponent(tenantId)}`),
    listPage: (tenantId: string, params: { filter?: string; dir?: string; limit?: number; offset?: number } = {}) => req<{ items: Question[]; total: number; limit: number; offset: number }>(`questions?tenant_id=${encodeURIComponent(tenantId)}&page=true&filter=${encodeURIComponent(params.filter ?? "")}&dir=${params.dir ?? "desc"}&limit=${params.limit ?? 10}&offset=${params.offset ?? 0}`),
    save: (question: Partial<Question> & { tenant_id?: string }) => req<Question>("questions", { method: question.ID ? "PUT" : "POST", body: JSON.stringify({ ...question, tenant_id: question.tenant_id || question.TenantID }) }),
    remove: (tenantId: string, id: string) => req<void>(`questions/${encodeURIComponent(id)}?tenant_id=${encodeURIComponent(tenantId)}`, { method: "DELETE" }),
    results: (tenantId: string, id: string, range?: { from_ts?: number; to_ts?: number }) => req<GraphResult>(`questions/${encodeURIComponent(id)}/results?tenant_id=${encodeURIComponent(tenantId)}${range?.from_ts ? `&from_ts=${range.from_ts}&to_ts=${range.to_ts}` : ""}`),
  },

  settings: {
    get: () => req<RuntimeSettings>("settings"),
    update: (partial: { retention_hours?: number; rate_limit_per_second?: number }) =>
      req<RuntimeSettings>("settings", { method: "PUT", body: JSON.stringify(partial) }),
  },

  adminUsers: {
    list: (filter = "") => req<AdminUser[]>(`admin-users?filter=${encodeURIComponent(filter)}`),
    listPage: (params: { filter?: string; sort?: string; dir?: string; limit?: number; offset?: number } = {}) => req<{ items: AdminUser[]; total: number; limit: number; offset: number }>(`admin-users?page=true&filter=${encodeURIComponent(params.filter ?? "")}&sort=${encodeURIComponent(params.sort ?? "created")}&dir=${params.dir ?? "desc"}&limit=${params.limit ?? 10}&offset=${params.offset ?? 0}`),
    create: (body: Partial<AdminUser>) => req<AdminUser & { invitation_token?: string }>("admin-users", { method: "POST", body: JSON.stringify(body) }),
    update: (id: string, body: Partial<AdminUser>) => req<AdminUser>(`admin-users/${encodeURIComponent(id)}`, { method: "PUT", body: JSON.stringify(body) }),
    remove: (id: string) => req<void>(`admin-users/${encodeURIComponent(id)}`, { method: "DELETE" }),
  },

  adminTeams: {
    list: () => req<AdminTeam[]>("admin-teams"),
    listPage: (params: { filter?: string; dir?: string; limit?: number; offset?: number } = {}) => req<AdminTeamPage>(`admin-teams?page=true&filter=${encodeURIComponent(params.filter ?? "")}&dir=${params.dir ?? "desc"}&limit=${params.limit ?? 10}&offset=${params.offset ?? 0}`),
    create: (name: string) => req<AdminTeam>("admin-teams", { method: "POST", body: JSON.stringify({ name }) }),
    update: (teamId: string, name: string) => req<AdminTeam>(`admin-teams/${encodeURIComponent(teamId)}`, { method: "PUT", body: JSON.stringify({ name }) }),
    delete: (teamId: string) => req<void>(`admin-teams/${encodeURIComponent(teamId)}`, { method: "DELETE" }),
    members: (teamId: string) => req<AdminTeamMember[]>(`admin-teams/${encodeURIComponent(teamId)}/members`),
    addMember: (teamId: string, userId: string, role?: string) => req<AdminTeamMember>(`admin-teams/${encodeURIComponent(teamId)}/members`, { method: "POST", body: JSON.stringify({ user_id: userId, role }) }),
    removeMember: (teamId: string, userId: string) => req<void>(`admin-teams/${encodeURIComponent(teamId)}/members/${encodeURIComponent(userId)}`, { method: "DELETE" }),
  },

  tenantData: {
    delete: (tenantId: string) => req<{ tenant_id: string; deleted: Record<string, number> }>(`tenants/data/${encodeURIComponent(tenantId)}?confirm=true`, { method: "DELETE" }),
  },

  operations: {
    flags: (tenantId: string) => req<unknown[]>(`feature-flags?tenant_id=${encodeURIComponent(tenantId)}`),
    flagsPage: (tenantId: string, params: { filter?: string; dir?: string; limit?: number; offset?: number } = {}) => req<{ items: unknown[]; total: number; limit: number; offset: number }>(`feature-flags?tenant_id=${encodeURIComponent(tenantId)}&page=true&filter=${encodeURIComponent(params.filter ?? "")}&dir=${params.dir ?? "desc"}&limit=${params.limit ?? 10}&offset=${params.offset ?? 0}`),
    setFlag: (tenantId: string, key: string, enabled: boolean) => req<unknown>("feature-flags", { method: "PUT", body: JSON.stringify({ tenant_id: tenantId, key, enabled }) }),
    deleteFlag: (tenantId: string, key: string) => req<void>(`feature-flags?tenant_id=${encodeURIComponent(tenantId)}&key=${encodeURIComponent(key)}`, { method: "DELETE" }),
    aggregations: (tenantId: string) => req<unknown[]>(`aggregations?tenant_id=${encodeURIComponent(tenantId)}`),
    aggregationsPage: (tenantId: string, params: { filter?: string; dir?: string; limit?: number; offset?: number } = {}) => req<{ items: unknown[]; total: number; limit: number; offset: number }>(`aggregations?tenant_id=${encodeURIComponent(tenantId)}&page=true&filter=${encodeURIComponent(params.filter ?? "")}&dir=${params.dir ?? "desc"}&limit=${params.limit ?? 10}&offset=${params.offset ?? 0}`),
    createAggregation: (body: Record<string, unknown>) => req<unknown>("aggregations", { method: "POST", body: JSON.stringify(body) }),
    updateAggregation: (id: string, body: Record<string, unknown>) => req<unknown>(`aggregations/${encodeURIComponent(id)}`, { method: "PUT", body: JSON.stringify(body) }),
    deleteAggregation: (id: string) => req<void>(`aggregations/${encodeURIComponent(id)}`, { method: "DELETE" }),
    refreshAggregations: (tenantId: string) => req<unknown>(`aggregations/refresh?tenant_id=${encodeURIComponent(tenantId)}`, { method: "POST" }),
    backfillAggregation: (id: string) => req<unknown>(`aggregations/${encodeURIComponent(id)}/backfill`, { method: "POST" }),
    anomalies: () => req<unknown[]>("anomalies"),
    anomaliesPage: (params: { filter?: string; dir?: string; limit?: number; offset?: number } = {}) => req<{ items: unknown[]; total: number; limit: number; offset: number }>(`anomalies?page=true&filter=${encodeURIComponent(params.filter ?? "")}&dir=${params.dir ?? "desc"}&limit=${params.limit ?? 10}&offset=${params.offset ?? 0}`),
    auditLog: () => req<unknown[]>("audit-log?limit=100"),
    auditLogPage: (params: { filter?: string; dir?: string; limit?: number; offset?: number } = {}) => req<{ items: unknown[]; total: number; limit: number; offset: number }>(`audit-log?page=true&filter=${encodeURIComponent(params.filter ?? "")}&dir=${params.dir ?? "desc"}&limit=${params.limit ?? 10}&offset=${params.offset ?? 0}`),
    deadLetters: (params: { filter?: string; dir?: string; page?: number; limit?: number } = {}) => req<{ items: unknown[]; total: number; limit: number; offset: number }>(`webhooks/dead-letter?page=true&limit=${params.limit ?? 25}&offset=${(params.page ?? 0) * (params.limit ?? 25)}&filter=${encodeURIComponent(params.filter ?? "")}&dir=${params.dir ?? "desc"}`),
    deleteDeadLetter: (id: string) => req<void>(`webhooks/dead-letter?id=${encodeURIComponent(id)}`, { method: "DELETE" }),
    replayDeadLetter: (id: string) => req<void>(`webhooks/dead-letter/${encodeURIComponent(id)}/replay`, { method: "POST" }),
    replay: (tenantId: string, fromTs: number, toTs: number) => req<unknown>("replay", { method: "POST", body: JSON.stringify({ tenant_id: tenantId, from_ts: fromTs, to_ts: toTs }) }),
    createExperiment: (body: { name: string; tenant_id: string; variants: string; traffic: number }) => req<unknown>("experiments", { method: "POST", body: JSON.stringify(body) }),
    experiments: (tenantId: string) => req<unknown[]>(`experiments?tenant_id=${encodeURIComponent(tenantId)}`),
    experimentsPage: (tenantId: string, params: { filter?: string; dir?: string; limit?: number; offset?: number } = {}) => req<{ items: unknown[]; total: number; limit: number; offset: number }>(`experiments?tenant_id=${encodeURIComponent(tenantId)}&page=true&filter=${encodeURIComponent(params.filter ?? "")}&dir=${params.dir ?? "desc"}&limit=${params.limit ?? 10}&offset=${params.offset ?? 0}`),
    updateExperiment: (id: string, body: Record<string, unknown>) => req<unknown>(`experiments/${encodeURIComponent(id)}`, { method: "PUT", body: JSON.stringify(body) }),
    deleteExperiment: (id: string) => req<void>(`experiments/${encodeURIComponent(id)}`, { method: "DELETE" }),
    assignExperiment: (id: string, sessionId: string) => req<unknown>(`experiments/${encodeURIComponent(id)}/assign?session_id=${encodeURIComponent(sessionId)}`),
  },

  ingest: {
    send: (events: Record<string, unknown>[]) =>
      req<{ accepted: number; rejected: number }>("ingest", {
        method: "POST",
        body: JSON.stringify({ events }),
      }),
  },
};

export { ApiError };
