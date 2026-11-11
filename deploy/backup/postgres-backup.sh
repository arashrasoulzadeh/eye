#!/usr/bin/env bash
set -euo pipefail

: "${DATABASE_URL:?DATABASE_URL is required}"
: "${BACKUP_DIR:=/var/backups/eye}"
: "${BACKUP_RETENTION_DAYS:=14}"

mkdir -p "$BACKUP_DIR"
timestamp="$(date -u +%Y%m%dT%H%M%SZ)"
output="$BACKUP_DIR/eye-$timestamp.dump"
pg_dump --format=custom --no-owner --file="$output" "$DATABASE_URL"
sha256sum "$output" > "$output.sha256"
find "$BACKUP_DIR" -type f -name 'eye-*.dump' -mtime "+$BACKUP_RETENTION_DAYS" -delete
find "$BACKUP_DIR" -type f -name 'eye-*.dump.sha256' -mtime "+$BACKUP_RETENTION_DAYS" -delete
echo "$output"
