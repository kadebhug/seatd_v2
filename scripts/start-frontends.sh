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

if ! command -v npm >/dev/null 2>&1; then
  printf 'error: npm is required; install Node.js 24 and npm 11\n' >&2
  exit 1
fi

web_port="${SEATD_WEB_PORT:-3000}"
guest_port="${SEATD_GUEST_PORT:-5173}"
api_base="${SEATD_API_BASE_URL:-http://localhost:${SEATD_API_PORT:-8080}}"
export SEATD_API_BASE_URL="$api_base"
export VITE_SEATD_API_BASE_URL="${VITE_SEATD_API_BASE_URL:-$api_base}"

pids=()

shutdown() {
  trap - EXIT INT TERM
  if [[ ${#pids[@]} -eq 0 ]]; then
    return
  fi
  printf 'Stopping frontends.\n' >&2
  kill "${pids[@]}" 2>/dev/null || true
  wait "${pids[@]}" 2>/dev/null || true
}

trap shutdown EXIT
trap 'shutdown; exit 130' INT
trap 'shutdown; exit 143' TERM

printf 'Starting web (:%s) and guest (:%s).\n' "$web_port" "$guest_port"

npm run dev --workspace=@seatd/web -- --port "$web_port" &
pids+=("$!")
npm run dev --workspace=@seatd/guest -- --port "$guest_port" &
pids+=("$!")

wait -n "${pids[@]}"
status=$?
exit "$status"
