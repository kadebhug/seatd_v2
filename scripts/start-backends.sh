#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

if [[ -f .env.local ]]; then
  set -a
  # shellcheck disable=SC1091
  source .env.local
  set +a
elif [[ -f .env.example ]]; then
  printf 'warning: .env.local not found; loading .env.example. Run make bootstrap.\n' >&2
  set -a
  # shellcheck disable=SC1091
  source .env.example
  set +a
fi

if ! command -v go >/dev/null 2>&1; then
  printf 'error: go is required; install Go 1.24+\n' >&2
  exit 1
fi

pids=()

shutdown() {
  trap - EXIT INT TERM
  if [[ ${#pids[@]} -eq 0 ]]; then
    return
  fi
  printf 'Stopping backends.\n' >&2
  kill "${pids[@]}" 2>/dev/null || true
  wait "${pids[@]}" 2>/dev/null || true
}

trap shutdown EXIT
trap 'shutdown; exit 130' INT
trap 'shutdown; exit 143' TERM

printf 'Starting api (:%s), worker, and realtime (:%s).\n' \
  "${SEATD_API_PORT:-8080}" \
  "${SEATD_REALTIME_PORT:-8082}"

go run ./apps/api/cmd/api &
pids+=("$!")
go run ./apps/worker/cmd/worker &
pids+=("$!")
go run ./apps/realtime/cmd/realtime &
pids+=("$!")

wait -n "${pids[@]}"
status=$?
exit "$status"
