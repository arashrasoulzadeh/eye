# Dimensions API

`POST /query-explorer?mode=dimensions`

Admin-user authentication is required. JSON body limit is 32,768 bytes; unknown fields are rejected.

| Field | Type | Meaning |
|---|---|---|
| `tenant_id` | string | Required tenant |
| `from_ts, to_ts` | integer | Inclusive Unix milliseconds |
| `types` | string[] | Event names, maximum 100 |
| `group, split` | string | Dimension key or @event |
| `metric` | string | Dimension to measure |
| `operation` | string | Required aggregation |
| `filters` | object[] | key, value, mode; maximum 20 |
| `filter_logic` | string | and (default) or or |
| `filter_key, filter_value, filter_mode` | string | Legacy single filter |
| `time_bucket` | string | Empty, hour, day, month; UTC |
| `explode` | boolean | Deduplicate array members per event |
| `full_range` | boolean | Full scan or database fast path |
| `summary` | boolean | Database counts, before dimension filters |
| `correlate` | string | Second numeric dimension |
| `discover_correlations` | boolean | Discover numeric associations |
| `drill_group, drill_split` | string/null | Select drill-down group and optional split |

## Operations and filters

`count`, `sum`, `avg`, `min`, `max`, `distinct`, `missing`, `median`, `p90`, `p95`, `p99`, `stddev`.

Multi-filter modes: `equal`, `not_equal`, `contains`, `exists`, `missing`.

Nested keys use JSON Pointer. Contains is case-insensitive. Standard deviation is population standard deviation. Percentiles use linear interpolation.

```json
{"tenant_id":"example","from_ts":1700000000000,"to_ts":1700086400000,"types":["purchase"],"group":"country","metric":"amount","operation":"sum","filters":[],"explode":false}
```

## Results and limits

Rows contain group, split, value, count and valid. Result fields also include keys, types, profiles, events, correlation, correlation_n, associations, association_scanned, matched, users, sessions, identified_events, scanned, truncated and execution. User and session counts are calculated in summary mode.

| Limit | Value |
|---|---:|
| Timeout | 8 s |
| Default scan | 20000 |
| Returned rows | 1000 |
| Internal groups | 5000 |
| Distinct entries / percentile samples (each) | 200000 |
| Drill events | 100 |
| Association scan | 10000 |
| Association dimensions | 16 |
| Minimum association pairs | 30 |
| Returned associations | 10 |

Inspect truncated. Correlation is not causation. Full-range execution still has time and memory limits.
