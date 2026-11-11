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
if [[ -n "${BACKUP_S3_URI:-}" ]]; then
  command -v aws >/dev/null 2>&1 || { echo "aws CLI is required when BACKUP_S3_URI is set" >&2; exit 1; }
  aws s3 cp "$output" "$BACKUP_S3_URI/$(basename "$output")"
  aws s3 cp "$output.sha256" "$BACKUP_S3_URI/$(basename "$output.sha256")"
fi
find "$BACKUP_DIR" -type f -name 'eye-*.dump' -mtime "+$BACKUP_RETENTION_DAYS" -delete
find "$BACKUP_DIR" -type f -name 'eye-*.dump.sha256' -mtime "+$BACKUP_RETENTION_DAYS" -delete
echo "$output"
