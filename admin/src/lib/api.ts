export type Tenant = { ID: string; Name: string; APIKey: string; CreatedAt: string };
export type Session = { ID: string; TenantID: string; StartedAt: string; LastSeen: string; EventCount: number };
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
export type Schema = { ID: string; TenantID: string; Name: string; Version: number; Definition: string; CreatedAt: number };
export type Metrics = { events: number; sessions: number; tenants: number };
export type Cohort = { ID: string; TenantID: string; Name: string; EventName: string; MinCount: number; CreatedAt: number; UpdatedAt: number };
export type AlertRule = {
  ID: string;
  TenantID: string;
  EventName: string;
  DimsKey: string;
  Metric: string;
  Threshold: number;
  WebhookURL: string;
  Enabled: boolean;
  LastFiredAt: number;
  LastFiredCount: number;
  CreatedAt: number;
  UpdatedAt: number;
};

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

export const api = {
  metrics: () => req<Metrics>("metrics"),

  tenants: {
    list: () => req<Tenant[]>("tenants"),
    create: (name: string, apiKey: string) =>
      req<Tenant>("tenants", { method: "POST", body: JSON.stringify({ Name: name, APIKey: apiKey }) }),
    remove: (id: string) => req<void>(`tenants/${id}`, { method: "DELETE" }),
    rotateKey: (id: string) => req<{ api_key: string }>(`tenants/${id}/rotate-key`, { method: "POST" }),
  },

  sessions: {
    list: () => req<Session[]>("sessions"),
    create: (tenantId: string) =>
      req<Session>("sessions", { method: "POST", body: JSON.stringify({ tenant_id: tenantId }) }),
  },

  events: {
    // Hits /query (not /events) because that's the endpoint that actually
    // supports the from_ts/to_ts time-range filter (added for Phase 6);
    // /events only ever supported event_name/tenant_id/session_id.
    list: (
      filters: { event_name?: string; tenant_id?: string; session_id?: string; from_ts?: number; to_ts?: number } = {}
    ) => {
      const qs = new URLSearchParams(
        Object.entries(filters)
          .filter(([, v]) => v !== undefined && v !== "")
          .map(([k, v]) => [k, String(v)])
      );
      const suffix = qs.toString() ? `?${qs}` : "";
      return req<Event[]>(`query${suffix}`);
    },
  },

  schemas: {
    list: (tenantId: string) => req<Schema[]>(`schemas?tenant_id=${encodeURIComponent(tenantId)}`),
    register: (tenantId: string, name: string, definition: string) =>
      req<Schema>("schemas", {
        method: "POST",
        body: JSON.stringify({ tenant_id: tenantId, name, definition }),
      }),
  },

  cohorts: {
    list: (tenantId: string) => req<Cohort[]>(`cohorts?tenant_id=${encodeURIComponent(tenantId)}`),
    create: (tenantId: string, name: string, eventName: string, minCount: number) =>
      req<Cohort>("cohorts", {
        method: "POST",
        body: JSON.stringify({ tenant_id: tenantId, name, event_name: eventName, min_count: minCount }),
      }),
    materialize: (id: string) => req<{ members: number }>(`cohorts/${id}/materialize`, { method: "POST" }),
    members: (id: string) => req<string[]>(`cohorts/${id}/members`),
  },

  alerts: {
    list: (tenantId: string) => req<AlertRule[]>(`alerts?tenant_id=${encodeURIComponent(tenantId)}`),
    create: (rule: { tenant_id: string; event_name: string; dims_key?: string; metric?: string; threshold: number; webhook_url: string }) =>
      req<AlertRule>("alerts", { method: "POST", body: JSON.stringify(rule) }),
    remove: (id: string) => req<void>(`alerts/${id}`, { method: "DELETE" }),
    setEnabled: (id: string, enabled: boolean) => req<AlertRule>(`alerts/${id}/${enabled ? "enable" : "disable"}`, { method: "POST" }),
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
