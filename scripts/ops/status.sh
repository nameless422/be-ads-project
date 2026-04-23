#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
RUN_DIR="${ROOT_DIR}/run"
LOG_DIR="${ROOT_DIR}/logs"

source "${SCRIPT_DIR}/common.sh"

SERVICES=(
  "control-plane"
  "collector-worker"
  "transformer-worker"
  "bi-api"
)

for name in "${SERVICES[@]}"; do
  pid_file="${RUN_DIR}/${name}.pid"
  bin_path="${RUN_DIR}/${name}"
  if pid="$(resolve_service_pid "${pid_file}" "${bin_path}" 2>/dev/null)"; then
    write_pid_file "${pid_file}" "${pid}"
    echo "${name}: running pid=${pid}"
  elif [[ -f "${pid_file}" ]]; then
    echo "${name}: stale pid file"
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
