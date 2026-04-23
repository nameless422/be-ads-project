#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

compose_cmd() {
  if docker compose version >/dev/null 2>&1; then
    echo "docker compose"
    return
  fi
  if command -v docker-compose >/dev/null 2>&1; then
    echo "docker-compose"
    return
  fi

  echo "missing docker compose capability: neither 'docker compose' nor 'docker-compose' is available" >&2
  exit 1
}

cd "${ROOT_DIR}"
COMPOSE="$(compose_cmd)"
${COMPOSE} -f docker-compose.dev.yml up -d
${COMPOSE} -f docker-compose.dev.yml ps
