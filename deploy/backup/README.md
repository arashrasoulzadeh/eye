# PostgreSQL backups

`postgres-backup.sh` creates custom-format dumps, writes a SHA-256 sidecar, and removes dumps older than `BACKUP_RETENTION_DAYS` (14 by default).

`restore-drill.sh` verifies the checksum, restores into the separate `DRILL_DATABASE_URL`, checks that the schema is readable, and drops only that drill database on exit.

Required tools: `pg_dump`, `pg_restore`, `createdb`, `dropdb`, and `psql`.
