# SQL reference

The explorer compiles a restricted SQL dialect over tenant-scoped events, not arbitrary database SQL.

## Expressions

Supported functions: `COUNT, SUM, AVG, MIN, MAX, ROUND, LOWER, UPPER, COALESCE, LENGTH, ABS`.

Fields: `event_id, tenant_id, session_id, user_id, type, event_name, timestamp, client_ts, ingest_ts, payload, dims, source, is_aggregated, aggregation_id, experiment_id, variant`.

Use single quotes for strings. Escape an apostrophe by doubling it. Use explicit AS aliases. Property names can be referenced as `payload.country` or `dims.country`; unqualified custom properties fall back from payload to dimensions.

```sql
SELECT dims.country, COUNT(*) AS total FROM events
WHERE type = 'click' GROUP BY dims.country ORDER BY total DESC;
```

The compiler recognizes WHERE, JOIN, INNER, LEFT, ON, EXISTS, AND, OR, NOT, IS NULL, IN, BETWEEN, LIKE, GROUP BY, HAVING, ORDER BY, LIMIT, OFFSET and DISTINCT. Not every combination is valid on both databases. UNION, INTERSECT, EXCEPT, WITH, INTO, OVER and RETURNING are rejected.

## Limits

- 16,000 input bytes and 512 tokens.
- Five-second execution context.
- Maximum 1,000 returned rows; inspect the truncated flag.
- Property discovery samples 200 recent events and returns at most 500 keys.
- Tenant scope is mandatory; optional time bounds apply to client timestamps.

Source: [compiler](https://github.com/arashrasoulzadeh/eye/blob/main/internal/service/query_explorer.go).
