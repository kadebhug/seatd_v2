#!/usr/bin/env bash
set -euo pipefail

if [[ $# -lt 1 ]]; then
  printf 'usage: %s <go|gofmt> [args...]\n' "$0" >&2
  exit 2
fi

repo_root="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

tool="$1"
if command -v "$tool" >/dev/null 2>&1; then
  "$@"
  exit 0
fi

command -v docker >/dev/null 2>&1 || {
  printf 'error: %s is required; install Go 1.24+ or Docker\n' "$tool" >&2
  exit 1
}

docker run --rm -v "$repo_root:/src" -w /src golang:1.24-alpine "$@"
