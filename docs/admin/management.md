# Administration and operations handbook

## Alerts

`/alerts`

Select tenant and event, rule type, window in seconds, count threshold, optional dimension key and destination. Create, edit, enable or disable, or delete rules. Verify webhook destination before enabling.

[Page source](https://github.com/arashrasoulzadeh/eye/blob/main/admin/src/app/%28dashboard%29/alerts/page.tsx)

## Cohorts

`/cohorts`

Choose tenant, name, event and minimum count. Create the cohort, materialize membership and inspect the result; remove obsolete cohorts.

[Page source](https://github.com/arashrasoulzadeh/eye/blob/main/admin/src/app/%28dashboard%29/cohorts/page.tsx)

## Experiments

`/experiments`

Choose tenant, name, comma-separated variants and traffic allocation. Create or edit experiments. Use experiment ID and session ID to assign a variant. Assignment is not proof of statistical significance.

[Page source](https://github.com/arashrasoulzadeh/eye/blob/main/admin/src/app/%28dashboard%29/experiments/page.tsx)

## Feature flags

`/feature-flags`

Select tenant and flag key, then create or update its enabled state. Delete obsolete flags only after checking consumers.

[Page source](https://github.com/arashrasoulzadeh/eye/blob/main/admin/src/app/%28dashboard%29/feature-flags/page.tsx)

## Aggregations

`/aggregations`

Set tenant, name, JSON rule, output event, threshold and window. Create or edit definitions, refresh or backfill results. Distinguish derived events from raw events.

[Page source](https://github.com/arashrasoulzadeh/eye/blob/main/admin/src/app/%28dashboard%29/aggregations/page.tsx)

## Operations

`/operations`

Inspect flags, aggregations, anomalies, audit records and dead letters. Select tenant scope and review recent records before refreshing aggregations or changing flags.

[Page source](https://github.com/arashrasoulzadeh/eye/blob/main/admin/src/app/%28dashboard%29/operations/page.tsx)

## Compression

`/compression`

Inspect storage inventory, event types, age groups, indexes and estimated savings. Treat pruning as destructive; verify retention requirements and backups first.

[Page source](https://github.com/arashrasoulzadeh/eye/blob/main/admin/src/app/%28dashboard%29/compression/page.tsx)

## Dead letters

`/dead-letters`

Filter failed deliveries by URL, ID or error and inspect payload, attempts and last error. Correct the destination before replay; delete only when no longer needed.

[Page source](https://github.com/arashrasoulzadeh/eye/blob/main/admin/src/app/%28dashboard%29/dead-letters/page.tsx)

## Schemas

`/schemas`

Select tenant, schema name and JSON definition. Register the schema and inspect the resulting list. Validate representative payloads before changing ingestion policy.

[Page source](https://github.com/arashrasoulzadeh/eye/blob/main/admin/src/app/%28dashboard%29/schemas/page.tsx)

## Admin teams

`/admin-teams`

Create a team, select it, add users with team roles, update team details or remove membership. Review access before deleting a team.

[Page source](https://github.com/arashrasoulzadeh/eye/blob/main/admin/src/app/%28dashboard%29/admin-teams/page.tsx)

## Admin users

`/admin-users`

Create or invite a user with email, name, role and status. Protect invitation tokens. Edit or remove accounts after reviewing access requirements.

[Page source](https://github.com/arashrasoulzadeh/eye/blob/main/admin/src/app/%28dashboard%29/admin-users/page.tsx)

## Settings

`/settings`

Review retention hours and ingestion rate limit in events per second. Save only after evaluating data retention and traffic requirements.

[Page source](https://github.com/arashrasoulzadeh/eye/blob/main/admin/src/app/%28dashboard%29/settings/page.tsx)
