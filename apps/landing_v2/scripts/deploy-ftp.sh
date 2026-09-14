#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Usage:
  npm run deploy

Required environment variables:
  DEPLOY_HOST      FTP host, for example ftp.example.com
  DEPLOY_USER      FTP username
  DEPLOY_PASSWORD  FTP password
  DEPLOY_PATH      Remote directory, for example public_html

Optional:
  DEPLOY_STRICT_TLS=1  Verify the FTP server certificate hostname.

You can place these values in .env.deploy. The file is ignored by git.
EOF
}

if [[ "${1:-}" == "--help" || "${1:-}" == "-h" ]]; then
  usage
  exit 0
fi

if [[ -f ".env.deploy" ]]; then
  set -a
  # shellcheck disable=SC1091
  source ".env.deploy"
  set +a
fi

required_vars=(DEPLOY_HOST DEPLOY_USER DEPLOY_PASSWORD DEPLOY_PATH)
for name in "${required_vars[@]}"; do
  if [[ -z "${!name:-}" ]]; then
    echo "Missing required environment variable: $name" >&2
    echo "Create .env.deploy from .env.deploy.example or export the variables." >&2
    exit 1
  fi
done

if ! command -v curl >/dev/null 2>&1; then
  echo "curl is required for FTP deployment." >&2
  exit 1
fi

npm run build

if [[ ! -d "out" ]]; then
  echo "Build completed, but out/ was not created." >&2
  exit 1
fi

tls_args=(--ssl-reqd)
if [[ "${DEPLOY_STRICT_TLS:-0}" != "1" ]]; then
  tls_args+=(--insecure)
fi

remote_base="ftp://${DEPLOY_HOST}:21/${DEPLOY_PATH%/}"
file_count="$(find out -type f | wc -l | tr -d ' ')"

echo "Uploading ${file_count} files to ${DEPLOY_HOST}/${DEPLOY_PATH%/}"

while IFS= read -r -d "" file; do
  rel="${file#out/}"
  curl \
    -sS \
    --fail \
    "${tls_args[@]}" \
    --ftp-create-dirs \
    --user "${DEPLOY_USER}:${DEPLOY_PASSWORD}" \
    -T "$file" \
    "${remote_base}/${rel}" \
    >/dev/null
  printf "."
done < <(find out -type f -print0)

printf "\nDeploy complete.\n"
