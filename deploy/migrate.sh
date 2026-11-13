#!/usr/bin/env bash
set -euo pipefail

# Non-destructive SQLite -> PostgreSQL migration.
# Usage: CONFIRM_MIGRATION=YES ./deploy/migrate.sh
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
if [[ -f "$ROOT/.env" ]]; then set -a; source "$ROOT/.env"; set +a; fi
: "${POSTGRES_PASSWORD:?POSTGRES_PASSWORD is required in .env}"
if [[ "${CONFIRM_MIGRATION:-}" != "YES" ]]; then
  echo "Refusing to migrate without CONFIRM_MIGRATION=YES" >&2
  echo "The existing eye-data SQLite volume will be preserved." >&2
  exit 2
fi
command -v docker >/dev/null || { echo "docker is required" >&2; exit 1; }
cd "$ROOT"
COMPOSE=(docker compose -f docker-compose.yml -f docker-compose.postgres.yml)
"${COMPOSE[@]}" config >/dev/null
"${COMPOSE[@]}" pull server collector admin
if ! "${COMPOSE[@]}" run --rm --no-deps -T server test -x /app/eye-migrate >/dev/null 2>&1; then
  echo "The pulled server image does not contain /app/eye-migrate." >&2
  echo "Wait for the image workflow from the migration release to finish, then rerun this script." >&2
  exit 1
fi
"${COMPOSE[@]}" stop server collector admin
restore_old() {
  if [[ "${migration_ok:-0}" != "1" ]]; then
    echo "Migration failed; restarting the original SQLite stack. SQLite data was not removed." >&2
    docker compose up -d --no-build server collector admin || true
  fi
}
trap restore_old EXIT
"${COMPOSE[@]}" up -d postgres
for attempt in {1..30}; do
  if "${COMPOSE[@]}" exec -T postgres pg_isready -U "${POSTGRES_USER:-eye}" -d "${POSTGRES_DB:-eye}" >/dev/null 2>&1; then
    break
  fi
  if [[ "$attempt" -eq 30 ]]; then echo "PostgreSQL did not become ready" >&2; exit 1; fi
  sleep 2
done
"${COMPOSE[@]}" run --rm -T server /app/eye-migrate -sqlite /data/eye.db
"${COMPOSE[@]}" up -d --no-build server collector admin
migration_ok=1
echo "Migration complete. PostgreSQL is active; the original SQLite volume remains preserved."
