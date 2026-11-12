#!/usr/bin/env bash
set -euo pipefail

# Pull prebuilt images published by GitHub Actions and restart the stack.
# Usage: ADMIN_KEY=... API_KEY_SALT=... ./deploy/deploy.sh
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# Compose reads .env for interpolation, and this script also loads it so the
# required keys and optional GHCR login work without repeating secrets on the
# command line. Explicitly exported environment variables win over .env.
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

# Optional private-GHCR login. Public images work without these variables.
if [[ -n "${GHCR_TOKEN:-}" ]]; then
  : "${GHCR_USERNAME:?GHCR_USERNAME is required with GHCR_TOKEN}"
  printf '%s' "$GHCR_TOKEN" | docker login ghcr.io --username "$GHCR_USERNAME" --password-stdin
fi

docker compose pull --quiet
# This deliberately does not run `down`, `down -v`, prune, or remove volumes.
# Compose recreates containers when the image changes while preserving volumes.
docker compose up -d --no-build

for attempt in {1..30}; do
  if docker compose ps --status running --services | grep -qx server && docker compose ps --status running --services | grep -qx admin; then
    docker compose ps
    echo "Eye is running from prebuilt images: admin http://127.0.0.1:3003, API http://127.0.0.1:8080"
    exit 0
  fi
  if [[ "$attempt" -eq 30 ]]; then
    docker compose ps
    docker compose logs --tail=80 server admin >&2 || true
    echo "Eye services did not become running" >&2
    exit 1
  fi
  sleep 2
done
