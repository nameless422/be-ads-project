#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
RUN_DIR="${ROOT_DIR}/run"

source "${SCRIPT_DIR}/common.sh"

SERVICES=(
  "control-plane"
  "collector-worker"
  "transformer-worker"
  "bi-api"
)

stop_pid() {
  local pid="$1"
  if [[ -z "${pid}" ]] || ! kill -0 "${pid}" 2>/dev/null; then
    return 0
  fi

  pkill -TERM -P "${pid}" 2>/dev/null || true
  kill "${pid}" 2>/dev/null || true

  for _ in {1..10}; do
    if ! kill -0 "${pid}" 2>/dev/null; then
      return 0
    fi
    sleep 1
  done

  pkill -KILL -P "${pid}" 2>/dev/null || true
  kill -9 "${pid}" 2>/dev/null || true
}

for name in "${SERVICES[@]}"; do
  pid_file="${RUN_DIR}/${name}.pid"
  bin_path="${RUN_DIR}/${name}"
  pid="$(resolve_service_pid "${pid_file}" "${bin_path}" 2>/dev/null || true)"
  stop_pid "${pid}"
  rm -f "${pid_file}"
  echo "${name} stopped"
done

pkill -f "${ROOT_DIR}/run/control-plane|${ROOT_DIR}/run/collector-worker|${ROOT_DIR}/run/transformer-worker|${ROOT_DIR}/run/bi-api" 2>/dev/null || true
