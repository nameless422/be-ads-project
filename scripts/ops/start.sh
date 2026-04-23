#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
RUN_DIR="${ROOT_DIR}/run"
LOG_DIR="${ROOT_DIR}/logs"
BOOT_LOG="${LOG_DIR}/startup.log"

SERVICES=(
  "control-plane:./cmd/control-plane:"
  "collector-worker:./cmd/collector-worker:"
  "transformer-worker:./cmd/transformer-worker:"
  "bi-api:./cmd/bi-api:8080"
)

mkdir -p "${RUN_DIR}" "${LOG_DIR}"

require_port_free() {
  local port="$1"
  if [[ -z "${port}" ]]; then
    return 0
  fi
  if lsof -nP -iTCP:"${port}" -sTCP:LISTEN >/dev/null 2>&1; then
    echo "port ${port} is already in use; run ./scripts/ops/status.sh to inspect"
    exit 1
  fi
}

is_running() {
  local pid="$1"
  [[ -n "${pid}" ]] && kill -0 "${pid}" 2>/dev/null
}

start_service() {
  local name="$1"
  local pkg="$2"
  local port="$3"
  local bin_path="${RUN_DIR}/${name}"
  local pid_file="${RUN_DIR}/${name}.pid"
  local stdout_log="${LOG_DIR}/${name}.stdout.log"

  if [[ -f "${pid_file}" ]]; then
    local existing_pid
    existing_pid="$(cat "${pid_file}")"
    if is_running "${existing_pid}"; then
      echo "${name} already running, pid=${existing_pid}"
      return 0
    fi
    rm -f "${pid_file}"
  fi

  require_port_free "${port}"

  echo "[build] ${name}" >>"${BOOT_LOG}"
  go build -o "${bin_path}" "${pkg}" >>"${BOOT_LOG}" 2>&1

  nohup "${bin_path}" >>"${stdout_log}" 2>&1 < /dev/null &
  local pid=$!
  echo "${pid}" >"${pid_file}"

  sleep 1
  if ! is_running "${pid}"; then
    echo "${name} failed to start, check ${stdout_log} and ${BOOT_LOG}"
    rm -f "${pid_file}"
    exit 1
  fi

  echo "${name} started, pid=${pid}"
}

cd "${ROOT_DIR}"

for entry in "${SERVICES[@]}"; do
  IFS=":" read -r name pkg port <<<"${entry}"
  start_service "${name}" "${pkg}" "${port}"
done

echo "startup log: ${BOOT_LOG}"
