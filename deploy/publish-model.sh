#!/usr/bin/env bash
set -euo pipefail

# Package an already validated local model registry as an OCI image for GHCR.
# The application image is not rebuilt.
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
REGISTRY_DIR="${MODEL_REGISTRY_DIR:-$ROOT/models}"
IMAGE="${MODEL_IMAGE:?Set MODEL_IMAGE, e.g. ghcr.io/arashrasoulzadeh/eye-models:latest}"

[[ -d "$REGISTRY_DIR" ]] || { echo "model registry directory not found: $REGISTRY_DIR" >&2; exit 1; }
command -v docker >/dev/null || { echo "docker is required" >&2; exit 1; }
docker build -f "$ROOT/deploy/model-artifact.Dockerfile" -t "$IMAGE" "$REGISTRY_DIR"
docker push "$IMAGE"
echo "Published model artifact: $IMAGE"
