#!/usr/bin/env bash

service_pid_from_file() {
  local pid_file="$1"
  if [[ ! -f "${pid_file}" ]]; then
    return 1
  fi

  local pid
  pid="$(cat "${pid_file}" 2>/dev/null || true)"
  if [[ -n "${pid}" ]] && kill -0 "${pid}" 2>/dev/null; then
    printf '%s\n' "${pid}"
    return 0
  fi
  return 1
}

service_pid_from_process() {
  local bin_path="$1"
  local bin_name
  bin_name="$(basename "${bin_path}")"
  ps ax -o pid=,comm=,args= | awk -v target="${bin_path}" -v name="${bin_name}" '
    index($0, target) { print $1; exit }
    $2 == name { print $1; exit }
    $3 == ("./" name) { print $1; exit }
  '
}

resolve_service_pid() {
  local pid_file="$1"
  local bin_path="$2"

  local pid
  if pid="$(service_pid_from_file "${pid_file}")"; then
    printf '%s\n' "${pid}"
    return 0
  fi

  pid="$(service_pid_from_process "${bin_path}")"
  if [[ -n "${pid}" ]] && kill -0 "${pid}" 2>/dev/null; then
    printf '%s\n' "${pid}"
    return 0
  fi
  return 1
}

write_pid_file() {
  local pid_file="$1"
  local pid="$2"
  mkdir -p "$(dirname "${pid_file}")"
  printf '%s\n' "${pid}" >"${pid_file}"
}
