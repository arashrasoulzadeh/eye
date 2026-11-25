# Production deployment

Use the [production runbook](https://github.com/arashrasoulzadeh/eye/blob/main/deploy/PRODUCTION.md) as the operational source of truth.

Copy .env.example to .env, replace placeholders and keep secrets outside Git. Configure ADMIN_KEY, API_KEY_SALT, DATABASE_URL and exact CORS_ORIGINS. Production mode rejects development credentials.

```sh
./deploy/deploy.sh
```

The script pulls prebuilt GHCR images. Pin image tags for reproducible deployments. PostgreSQL requires DB_DRIVER=postgres. Redis Streams can be enabled with docker-compose.redis.yml. Kubernetes manifests, Helm and Terraform are in deploy/.

## Migration and recovery

Stop ingestion before migrating SQLite. Back up first and retain the original volume.

```sh
CONFIRM_MIGRATION=YES ./deploy/migrate.sh
```

Run deploy/backup/postgres-backup.sh daily and restore-drill.sh monthly against an isolated target. Store backups and checksums independently.

## Security and verification

Use a non-owner database role. Enable PostgreSQL RLS only after tenant-context integration tests pass. Verify health, authenticated session creation, ingestion and metrics after deployment. Configure Prometheus rules and Alertmanager receivers.
