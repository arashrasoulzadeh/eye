# Eye production runbook

## Database

Set `DB_DRIVER=postgres` and provide `DATABASE_URL` (or `DATABASE_URL_FILE`) for a PostgreSQL/TimescaleDB database. The server runs the portable GORM migration first, then creates the Timescale hypertable and indexes.

Use a dedicated application database role. Do not run the API as the database owner. RLS policy generation exists in `internal/storage/rls.go`, but must only be enabled after every tenant request path establishes `app.tenant_id` inside a transaction.

## Required secrets

Provide `ADMIN_KEY`, `API_KEY_SALT`, and `DATABASE_URL` through a secret manager or mounted files. Never commit these values. Set `CORS_ORIGINS` to the exact customer origins in production.

## Backups

Run `deploy/backup/postgres-backup.sh` daily and copy the resulting dump and checksum to independent object storage. Run `deploy/backup/restore-drill.sh` against an isolated database at least monthly. The restore drill must not use the production database as its target.

## Deployment checks

Verify `/healthz`, tenant creation with `x-admin-key`, session creation with `x-api-key`, authenticated ingestion, and metrics access after every deployment. The GitHub Actions Compose workflow covers this sequence automatically.

## Alerting

Install `deploy/k8s/prometheus-rules.yaml` and configure the receiver in `deploy/k8s/alertmanager-config.yaml` with the deployment's Slack, PagerDuty, or internal webhook endpoint.
