#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
output="$ROOT/docs/changelog.md"
{
  echo "# Changelog"
  echo
  echo "The 50 most recent commits on the current branch, newest first. This page is regenerated automatically by the commit hook."
  echo
  git -C "$ROOT" log -50 --date=short --pretty=format:'- **%ad** — %s (`%h`)' \
    | sed 's/[[:space:]]*$//' \
    || true
  echo
} > "$output"
