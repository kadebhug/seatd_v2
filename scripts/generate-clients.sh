#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

node packages/api-contract/scripts/validate.mjs
node packages/event-schema/scripts/validate.mjs
node packages/typescript-seatd-client/scripts/generate.mjs
node packages/dart-seatd-client/scripts/generate.mjs
