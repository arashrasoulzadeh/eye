# Analytics API

All routes below are registered with admin authentication. Supply tenant_id. Time bounds are Unix milliseconds.

| Route | HTTP | Parameters and behavior |
|---|---|---|
| `/analytics/funnel` | POST | JSON tenant_id, steps (nonempty), from_ts, to_ts |
| `/analytics/funnel/pinned` | GET / POST / DELETE | Read, replace or clear pinned funnel; name and two distinct steps required for POST |
| `/analytics/funnel/suggested` | GET | tenant_id, from_ts, to_ts |
| `/analytics/retention` | GET | granularity daily (default) or weekly; from_ts, to_ts |
| `/analytics/views` | GET / POST | List or save view model; tenant_id required |
| `/analytics/session-events` | GET | Events per session |
| `/analytics/devices` | GET | Device breakdown |
| `/analytics/top-n` | GET | event_name, limit (default 10) |
| `/analytics/sliding-window` | GET | event_name required; window_seconds defaults to 60 |
| `/analytics/distinct-sessions` | GET | event_name; returns distinct_sessions |
| `/analytics/percentiles` | GET | Session event percentiles |
| `/analytics/geography` | GET | dimension defaults to country |
| `/analytics/buckets` | GET | event_name, bucket (default hour), increasing from_ts and to_ts required |

Note: the views handler implements PUT and DELETE with a path ID, but the current mux registration is only /analytics/views; subpath reachability is not guaranteed.

Example funnel:

```json
{"tenant_id":"example","steps":["page_view","purchase"],"from_ts":1700000000000,"to_ts":1700086400000}
```
