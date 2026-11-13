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
activated=0
cleanup() {
  if [[ "$activated" -eq 1 && "$?" -ne 0 && -f "$tmp/previous.tar" ]]; then
    echo "Model activation failed; restoring previous model volume" >&2
    docker run --rm -v "$VOLUME:/target" -v "$tmp:/backup:ro" alpine:3.20 sh -ec 'find /target -mindepth 1 -maxdepth 1 -exec rm -rf -- {} +; tar -C /target -xf /backup/previous.tar' || true
  fi
  docker rm "$container" >/dev/null 2>&1 || true
  rm -rf "$tmp"
}
trap cleanup EXIT
docker create --name "$container" "$MODEL_IMAGE" >/dev/null
docker cp "$container:/models/." "$tmp/"
docker run --rm -v "$tmp:/models:ro" alpine:3.20 sh -ec '
  manifests=$(find /models -name manifest.json -type f)
  test -n "$manifests"
  for manifest in $manifests; do
    grep -q '"name"[[:space:]]*:[[:space:]]*"' "$manifest"
    grep -q '"version"[[:space:]]*:[[:space:]]*"' "$manifest"
    grep -q '"task"[[:space:]]*:[[:space:]]*"' "$manifest"
    grep -q '"artifact"[[:space:]]*:[[:space:]]*"' "$manifest"
    artifact=$(sed -n "s/.*\"artifact\"[[:space:]]*:[[:space:]]*\"\([^\"]*\)\".*/\1/p" "$manifest" | head -n 1)
    test -n "$artifact" && test "$(basename "$artifact")" = "$artifact"
    test -f "$(dirname "$manifest")/$artifact"
  done
'
docker run --rm -v "$VOLUME:/target:ro" -v "$tmp:/backup" alpine:3.20 sh -ec 'tar -C /target -cf /backup/previous.tar .'
docker run --rm -v "$VOLUME:/target" -v "$tmp:/source:ro" alpine:3.20 sh -ec 'find /target -mindepth 1 -maxdepth 1 -exec rm -rf -- {} +; cp -a /source/. /target/'
activated=1
echo "Installed model artifact $MODEL_IMAGE into volume $VOLUME"
