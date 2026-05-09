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
  ps ax -o pid=,args= | awk -v target="${bin_path}" -v rel="./${bin_name}" -v bare="${bin_name}" '
    {
      pid = $1
      $1 = ""
      sub(/^ +/, "", $0)
      split($0, argv, /[[:space:]]+/)
      cmd = argv[1]
      if (cmd == target || cmd == rel || cmd == bare) {
        print pid
        exit
      }
    }
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

list_managed_service_names() {
  local run_dir="$1"
  if [[ ! -d "${run_dir}" ]]; then
    return 0
  fi
  find "${run_dir}" -maxdepth 1 -type f -name '*.pid' -print | while read -r pid_file; do
    basename "${pid_file}" .pid
  done | sort
}
