#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
RUN_DIR="${ROOT_DIR}/run"

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

  local pgid
  pgid="$(ps -o pgid= -p "${pid}" 2>/dev/null | tr -d '[:space:]')"
  if [[ -n "${pgid}" ]]; then
    kill -- -"${pgid}" 2>/dev/null || kill "${pid}" 2>/dev/null || true
  else
    kill "${pid}" 2>/dev/null || true
  fi

  for _ in {1..10}; do
    if ! kill -0 "${pid}" 2>/dev/null; then
      return 0
    fi
    sleep 1
  done

  if [[ -n "${pgid}" ]]; then
    kill -9 -- -"${pgid}" 2>/dev/null || kill -9 "${pid}" 2>/dev/null || true
  else
    kill -9 "${pid}" 2>/dev/null || true
  fi
}

for name in "${SERVICES[@]}"; do
  pid_file="${RUN_DIR}/${name}.pid"
  if [[ -f "${pid_file}" ]]; then
    pid="$(cat "${pid_file}" 2>/dev/null || true)"
    stop_pid "${pid}"
    rm -f "${pid_file}"
    echo "${name} stopped"
  fi
done

pkill -f "${ROOT_DIR}/run/control-plane|${ROOT_DIR}/run/collector-worker|${ROOT_DIR}/run/transformer-worker|${ROOT_DIR}/run/bi-api" 2>/dev/null || true
