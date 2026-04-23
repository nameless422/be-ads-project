#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
RUN_DIR="${ROOT_DIR}/run"
LOG_DIR="${ROOT_DIR}/logs"
BOOT_LOG="${LOG_DIR}/startup.log"

source "${SCRIPT_DIR}/common.sh"

SERVICES=(
  "collector-worker:./cmd/collector-worker:"
  "transformer-worker:./cmd/transformer-worker:"
  "bi-api:./cmd/bi-api:8080"
  "control-plane:./cmd/control-plane:"
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

start_service() {
  local name="$1"
  local pkg="$2"
  local port="$3"
  local bin_path="${RUN_DIR}/${name}"
  local pid_file="${RUN_DIR}/${name}.pid"
  local stdout_log="${LOG_DIR}/${name}.stdout.log"

  local existing_pid
  if existing_pid="$(resolve_service_pid "${pid_file}" "${bin_path}")"; then
    write_pid_file "${pid_file}" "${existing_pid}"
    echo "${name} already running, pid=${existing_pid}"
    return 0
  fi
  rm -f "${pid_file}"

  require_port_free "${port}"

  echo "[build] ${name}" >>"${BOOT_LOG}"
  go build -o "${bin_path}" "${pkg}" >>"${BOOT_LOG}" 2>&1

  nohup "${bin_path}" >>"${stdout_log}" 2>&1 < /dev/null &
  local pid=$!
  write_pid_file "${pid_file}" "${pid}"

  sleep 1
  local resolved_pid
  if ! resolved_pid="$(resolve_service_pid "${pid_file}" "${bin_path}")"; then
    echo "${name} failed to start, check ${stdout_log} and ${BOOT_LOG}"
    rm -f "${pid_file}"
    exit 1
  fi
  write_pid_file "${pid_file}" "${resolved_pid}"

  echo "${name} started, pid=${resolved_pid}"
}

cd "${ROOT_DIR}"

for entry in "${SERVICES[@]}"; do
  IFS=":" read -r name pkg port <<<"${entry}"
  start_service "${name}" "${pkg}" "${port}"
done

echo "startup log: ${BOOT_LOG}"
