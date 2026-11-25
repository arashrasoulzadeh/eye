# Analytics and exploration

## Dimensions

Select a tenant and global time range. Add event types, choose grouping and split dimensions, and select an aggregation. Enable array explosion to count unique array items separately. Advanced analysis includes time buckets, dimension filters and comparisons. Inspect the scanned event count, groups and valid values before interpreting results. Search and inspect the result table, or switch to a chart.

## SQL explorer

The explorer runs read-only queries against events. Use schema suggestions, query history and the global time toggle. Results can be sorted, charted and exported as CSV or JSON.

```sql
SELECT type, COUNT(*) AS total FROM events GROUP BY type ORDER BY total DESC;
```

```sql
SELECT type, timestamp, user_id FROM events ORDER BY timestamp DESC LIMIT 20;
```

Only implemented SQL expressions are supported; editor suggestions are not a guarantee of backend support.
