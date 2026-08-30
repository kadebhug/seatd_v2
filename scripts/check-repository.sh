#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

required_dirs=(
  apps/api apps/worker apps/realtime apps/web apps/guest apps/waiter apps/display
  packages/api-contract packages/event-schema packages/dart-seatd-client
  packages/typescript-seatd-client infra ops docs/adr
)

for directory in "${required_dirs[@]}"; do
  if [[ ! -d "$directory" ]]; then
    printf 'error: required directory is missing: %s\n' "$directory" >&2
    exit 1
  fi
done

while IFS= read -r nested_git; do
  printf 'error: nested Git metadata is not allowed: %s\n' "$nested_git" >&2
  exit 1
done < <(find apps packages infra ops docs -mindepth 2 -name .git -print)

environment="${SEATD_ENV:-local}"
case "$environment" in
  local|test|staging|production) ;;
  *)
    printf 'error: SEATD_ENV must be local, test, staging, or production (got %s)\n' "$environment" >&2
    exit 1
    ;;
esac

printf 'Repository checks passed for environment %s.\n' "$environment"
