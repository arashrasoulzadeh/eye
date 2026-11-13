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
: "${GITHUB_SHA:=$(git -C "$ROOT" rev-parse HEAD)}"
: "${ADMIN_KEY:?ADMIN_KEY is required}"
: "${API_KEY_SALT:?API_KEY_SALT is required}"
command -v docker >/dev/null || { echo "docker is required" >&2; exit 1; }
docker compose version >/dev/null || { echo "docker compose plugin is required" >&2; exit 1; }

cd "$ROOT"
COMPOSE=(docker compose -f docker-compose.yml)
if [[ "${DB_DRIVER:-sqlite}" =~ ^(postgres|postgresql|timescale)$ ]]; then
  COMPOSE+=( -f docker-compose.postgres.yml )
fi
"${COMPOSE[@]}" config >/dev/null

# Optional private-GHCR login. Public images work without these variables.
if [[ -n "${GHCR_TOKEN:-}" ]]; then
  : "${GHCR_USERNAME:?GHCR_USERNAME is required with GHCR_TOKEN}"
  printf '%s' "$GHCR_TOKEN" | docker login ghcr.io --username "$GHCR_USERNAME" --password-stdin
fi

"${COMPOSE[@]}" pull --quiet

# Never restart with a stale `latest` image when the registry is still
# building the commit that was checked out on this server. The publish
# workflow writes this OCI revision label for every service image.
if [[ "${WAIT_FOR_IMAGE:-true}" == "true" && -n "${GITHUB_SHA:-}" ]]; then
  for attempt in {1..60}; do
    "${COMPOSE[@]}" pull --quiet server admin collector >/dev/null 2>&1 || true
    server_revision="$(docker image inspect "${EYE_SERVER_IMAGE:-ghcr.io/arashrasoulzadeh/eye-server:latest}" --format '{{index .Config.Labels "org.opencontainers.image.revision"}}' 2>/dev/null || true)"
    admin_revision="$(docker image inspect "${EYE_ADMIN_IMAGE:-ghcr.io/arashrasoulzadeh/eye-admin:latest}" --format '{{index .Config.Labels "org.opencontainers.image.revision"}}' 2>/dev/null || true)"
    if [[ "$server_revision" == "$GITHUB_SHA" || "$server_revision" == "${GITHUB_SHA:0:7}" ]] && [[ "$admin_revision" == "sha-${GITHUB_SHA}" || "$admin_revision" == "sha-${GITHUB_SHA:0:7}" ]]; then
      break
    fi
    if [[ "$attempt" -eq 60 ]]; then
      echo "GHCR images are not published for commit $GITHUB_SHA yet; refusing stale deployment." >&2
      exit 1
    fi
    sleep 10
  done
fi
# Optional standalone model artifact. This updates only the persistent model
# volume; leaving MODEL_IMAGE unset keeps deployments application-only.
if [[ -n "${MODEL_IMAGE:-}" ]]; then
  "$ROOT/deploy/update-models.sh"
fi
# This deliberately does not run `down`, `down -v`, prune, or remove volumes.
# Compose recreates containers when the image changes while preserving volumes.
"${COMPOSE[@]}" up -d --no-build

for attempt in {1..30}; do
  if "${COMPOSE[@]}" ps --status running --services | grep -qx server && "${COMPOSE[@]}" ps --status running --services | grep -qx admin; then
    "${COMPOSE[@]}" ps
    echo "Eye is running from prebuilt images: admin http://127.0.0.1:3003, API http://127.0.0.1:8080"
    exit 0
  fi
  if [[ "$attempt" -eq 30 ]]; then
    "${COMPOSE[@]}" ps
    "${COMPOSE[@]}" logs --tail=80 server admin >&2 || true
    echo "Eye services did not become running" >&2
    exit 1
  fi
  sleep 2
done
