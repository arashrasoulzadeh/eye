# PostgreSQL backups

`postgres-backup.sh` creates custom-format dumps, writes a SHA-256 sidecar, and removes dumps older than `BACKUP_RETENTION_DAYS` (14 by default).

Set `BACKUP_S3_URI=s3://bucket/eye-backups` to upload both files using the AWS CLI and its standard credentials/environment configuration. S3-compatible endpoints can be configured with the normal AWS CLI endpoint options.

`restore-drill.sh` verifies the checksum, restores into the separate `DRILL_DATABASE_URL`, checks that the schema is readable, and drops only that drill database on exit.

Required tools: `pg_dump`, `pg_restore`, `createdb`, `dropdb`, and `psql`.

The monthly `.github/workflows/restore-drill.yml` workflow uses the same script against `EYE_DRILL_DATABASE_URL`; configure the database and object-storage secrets before enabling it.
