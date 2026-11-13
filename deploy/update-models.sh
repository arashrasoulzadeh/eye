#!/usr/bin/env bash
set -euo pipefail

# Pull and install the standalone model OCI artifact without rebuilding or
# recreating the application containers or deleting any application data.
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
if [[ -f "$ROOT/.env" ]]; then set -a; source "$ROOT/.env"; set +a; fi
: "${MODEL_IMAGE:?Set MODEL_IMAGE, e.g. ghcr.io/arashrasoulzadeh/eye-models:latest}"
VOLUME="${MODEL_VOLUME:-eye-models}"
command -v docker >/dev/null || { echo "docker is required" >&2; exit 1; }

docker pull "$MODEL_IMAGE"
container="eye-model-import-$$"
tmp="$(mktemp -d)"
cleanup() { docker rm "$container" >/dev/null 2>&1 || true; rm -rf "$tmp"; }
trap cleanup EXIT
docker create --name "$container" "$MODEL_IMAGE" >/dev/null
docker cp "$container:/models/." "$tmp/"
docker run --rm -v "$VOLUME:/target" -v "$tmp:/source:ro" alpine:3.20 sh -c 'rm -rf /target/* /target/.[!.]* /target/..?* 2>/dev/null || true; cp -a /source/. /target/'
echo "Installed model artifact $MODEL_IMAGE into volume $VOLUME"
