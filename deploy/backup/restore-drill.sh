#!/usr/bin/env bash
set -euo pipefail

: "${BACKUP_FILE:?BACKUP_FILE is required}"
: "${DRILL_DATABASE_URL:?DRILL_DATABASE_URL is required}"

if [[ ! -f "$BACKUP_FILE" ]]; then echo "backup file not found: $BACKUP_FILE" >&2; exit 1; fi
if [[ -f "$BACKUP_FILE.sha256" ]]; then sha256sum --check "$BACKUP_FILE.sha256"; fi
createdb "$DRILL_DATABASE_URL"
trap 'dropdb "$DRILL_DATABASE_URL"' EXIT
pg_restore --no-owner --clean --if-exists --dbname="$DRILL_DATABASE_URL" "$BACKUP_FILE"
psql "$DRILL_DATABASE_URL" -v ON_ERROR_STOP=1 -c 'SELECT 1 FROM tenants LIMIT 1;'
echo "restore drill passed"
