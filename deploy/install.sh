#!/usr/bin/env bash
set -euo pipefail

# One-command production bootstrap for a Docker host.
# Usage: ADMIN_KEY=... API_KEY_SALT=... ./deploy/install.sh
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
: "${ADMIN_KEY:?ADMIN_KEY is required}"
: "${API_KEY_SALT:?API_KEY_SALT is required}"
command -v docker >/dev/null || { echo "docker is required" >&2; exit 1; }
docker compose version >/dev/null || { echo "docker compose plugin is required" >&2; exit 1; }
cd "$ROOT"
docker compose up -d --build
docker compose ps
echo "Eye is running: admin http://127.0.0.1:3003, API http://127.0.0.1:8080"
