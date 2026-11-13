# Eye production runbook

## Database

Copy `.env.example` to `.env`, replace every `replace-me` value, and keep the
file outside version control. Compose loads `.env` automatically; Kubernetes
values must be stored in the configured secret manager.

Set `DB_DRIVER=postgres` and provide `DATABASE_URL` (or `DATABASE_URL_FILE`) for a PostgreSQL/TimescaleDB database. The server runs the portable GORM migration first, then creates the Timescale hypertable and indexes.

The production Compose file uses prebuilt images from GHCR. Put `ADMIN_KEY`, `API_KEY_SALT`, and optional `GHCR_USERNAME`/`GHCR_TOKEN` in the project `.env`, then run `./deploy/deploy.sh`; it pulls images and never builds on the server. Set `EYE_SERVER_IMAGE`, `EYE_COLLECTOR_IMAGE`, or `EYE_ADMIN_IMAGE` in `.env` to pin another registry/tag.

The server mounts the persistent `eye-models` volume at `/models` and uses `MODEL_REGISTRY_DIR=/models`. Model artifacts use `/models/global/buyer/manifest.json` and `/models/global/buyer/model.json`; tenant overrides use `/models/tenants/<tenant-id>/buyer/`. Tenant artifacts override global artifacts, and the scheduled refresh consumes new valid artifacts without rebuilding the application image.

To publish the mounted registry as a standalone GHCR OCI artifact, set
`MODEL_IMAGE` and run `./deploy/publish-model.sh`. This builds only the model
artifact from `MODEL_REGISTRY_DIR` and pushes it independently of the server,
collector, and admin images.

On the server, update only the model volume with:

```sh
MODEL_IMAGE=ghcr.io/arashrasoulzadeh/eye-models:latest ./deploy/update-models.sh
```

This does not rebuild, remove, or reset application containers or data. The
The script validates every manifest and referenced artifact before activation,
backs up the current `eye-models` volume, and restores that backup if the
activation copy fails. The intelligence scheduler consumes the new validated
model on its next refresh. A failed update leaves the previous model set in
place; inspect the script output and retry after correcting the artifact.

To create the GHCR credentials, open GitHub → Settings → Developer settings → Personal access tokens → Tokens (classic) → Generate new token. Grant `read:packages` to pull private images, copy the token once, and add this to `.env`:

```dotenv
ADMIN_KEY=your-eye-admin-key
API_KEY_SALT=your-eye-api-key-salt
GHCR_USERNAME=arashrasoulzadeh
GHCR_TOKEN=ghp_or_your_token
```

For public GHCR images, omit `GHCR_USERNAME` and `GHCR_TOKEN`. Never commit `.env` or paste the token into shell history.

For a local production-like stack with PostgreSQL, pull the images first and then run `docker compose -f docker-compose.yml -f docker-compose.postgres.yml up -d --no-build server postgres`. Add `docker compose pull` before the command whenever you want the newest published images.

Add `-f docker-compose.redis.yml` to enable Redis Streams:

```sh
docker compose pull
docker compose -f docker-compose.yml -f docker-compose.redis.yml up -d --no-build
```

Kubernetes deployments should apply `deploy/k8s/redis.yaml` and the server manifest together.

Use a dedicated application database role. Do not run the API as the database owner. RLS policy generation exists in `internal/storage/rls.go`, but must only be enabled after every tenant request path establishes `app.tenant_id` inside a transaction.

Bootstrap the non-owner role with `deploy/postgres/roles.sql`, substituting its password through the deployment secret manager.

Set `ENABLE_POSTGRES_RLS=true` only after the API uses the non-owner role and the RLS integration workflow passes.

Set `DEAD_LETTER_NOTIFY_URL` to an internal HTTPS webhook if operators should be
notified when an alert delivery exhausts all retries. Notifications are
best-effort; the dead-letter record remains the source of truth.

## Required secrets

Provide `ADMIN_KEY`, `API_KEY_SALT`, and `DATABASE_URL` through a secret manager or mounted files. Never commit these values. Set `CORS_ORIGINS` to the exact customer origins in production.

## Backups

Run `deploy/backup/postgres-backup.sh` daily and copy the resulting dump and checksum to independent object storage. Run `deploy/backup/restore-drill.sh` against an isolated database at least monthly. The restore drill must not use the production database as its target.

## Deployment checks

Verify `/healthz`, tenant creation with `x-admin-key`, session creation with `x-api-key`, authenticated ingestion, and metrics access after every deployment. The GitHub Actions Compose workflow covers this sequence automatically.

## Alerting

Install `deploy/k8s/prometheus-rules.yaml` and configure the receiver in `deploy/k8s/alertmanager-config.yaml` with the deployment's Slack, PagerDuty, or internal webhook endpoint.
