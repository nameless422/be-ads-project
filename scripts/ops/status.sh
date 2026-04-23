#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
RUN_DIR="${ROOT_DIR}/run"
LOG_DIR="${ROOT_DIR}/logs"

SERVICES=(
  "control-plane"
  "collector-worker"
  "transformer-worker"
  "bi-api"
)

for name in "${SERVICES[@]}"; do
  pid_file="${RUN_DIR}/${name}.pid"
  if [[ -f "${pid_file}" ]]; then
    pid="$(cat "${pid_file}" 2>/dev/null || true)"
    if [[ -n "${pid}" ]] && kill -0 "${pid}" 2>/dev/null; then
      echo "${name}: running pid=${pid}"
    else
      echo "${name}: stale pid file"
    fi
  else
    echo "${name}: stopped"
  fi
done

echo
echo "listening ports:"
lsof -nP -iTCP:8080 -sTCP:LISTEN 2>/dev/null || true
lsof -nP -iTCP:18080 -sTCP:LISTEN 2>/dev/null || true

echo
echo "recent logs:"
for file in \
  "${LOG_DIR}/control-plane.stdout.log" \
  "${LOG_DIR}/collector-worker.stdout.log" \
  "${LOG_DIR}/transformer-worker.stdout.log" \
  "${LOG_DIR}/bi-api.stdout.log"; do
  if [[ -f "${file}" ]]; then
    echo "==> ${file}"
    tail -n 5 "${file}"
  fi
done
