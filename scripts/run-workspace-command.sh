#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 1 ]]; then
  printf 'usage: %s <bootstrap|fmt|lint|test|build|clean>\n' "$0" >&2
  exit 2
fi

command_name="$1"
case "$command_name" in
  bootstrap|fmt|lint|test|build|clean) ;;
  *)
    printf 'error: unsupported workspace command: %s\n' "$command_name" >&2
    exit 2
    ;;
esac

repo_root="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

hook_count=0
for area in packages apps infra ops; do
  while IFS= read -r hook; do
    hook_count=$((hook_count + 1))
    printf 'Running %s\n' "$hook"
    "$hook"
  done < <(find "$area" -mindepth 2 -maxdepth 3 -type f -path "*/scripts/$command_name" -perm -u+x -print | sort)
done

if [[ "$hook_count" -eq 0 ]]; then
  printf 'No component %s hooks registered.\n' "$command_name"
fi
