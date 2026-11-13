#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
output="$ROOT/docs/changelog.md"
{
  echo "# Changelog"
  echo
  echo "The 50 most recent commits on the current branch, newest first. Each entry includes the commit timestamp, subject, short hash, and full hash. This page is regenerated automatically by the commit hook."
  echo
  git -C "$ROOT" log -50 --date=iso-strict --pretty=format:'- **%ad** — %s (`%h`, `%H`)' \
    | sed 's/[[:space:]]*$//' \
    || true
  echo
} > "$output"
