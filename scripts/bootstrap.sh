#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

for tool in git make bash; do
  if ! command -v "$tool" >/dev/null 2>&1; then
    printf 'error: required tool not found: %s\n' "$tool" >&2
    exit 1
  fi
done

./scripts/check-repository.sh

if [[ ! -f .env.local ]]; then
  cp .env.example .env.local
  printf 'Created .env.local from .env.example.\n'
fi

./scripts/run-workspace-command.sh bootstrap
printf 'Seatd local bootstrap complete.\n'
