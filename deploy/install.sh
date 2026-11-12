#!/usr/bin/env bash
set -euo pipefail

# One-command production bootstrap for a Docker host.
# Usage: ADMIN_KEY=... API_KEY_SALT=... ./deploy/install.sh
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
if [[ -f "$ROOT/.env" ]]; then
  set -a
  # shellcheck disable=SC1091
  source "$ROOT/.env"
  set +a
fi
: "${ADMIN_KEY:?ADMIN_KEY is required}"
: "${API_KEY_SALT:?API_KEY_SALT is required}"
command -v docker >/dev/null || { echo "docker is required" >&2; exit 1; }
docker compose version >/dev/null || { echo "docker compose plugin is required" >&2; exit 1; }
cd "$ROOT"
docker compose config >/dev/null
docker compose pull
# This deliberately does not run `down`, `down -v`, prune, or remove volumes.
# Compose recreates containers when the image changes while preserving volumes.
docker compose up -d --no-build
for attempt in {1..30}; do
  if docker compose ps --status running --services | grep -qx server && docker compose ps --status running --services | grep -qx admin; then
    break
  fi
  if [[ "$attempt" -eq 30 ]]; then
    echo "Eye services did not become running" >&2
    docker compose ps
    exit 1
  fi
  sleep 2
done
docker compose ps
echo "Eye is running: admin http://127.0.0.1:3003, API http://127.0.0.1:8080"
