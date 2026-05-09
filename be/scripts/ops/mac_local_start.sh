#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
BE_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
REPO_DIR="$(cd "${BE_DIR}/.." && pwd)"
FE_DIR="${REPO_DIR}/fe"

INSTALL_MISSING_TOOLS="${INSTALL_MISSING_TOOLS:-1}"
SKIP_VERIFY="${SKIP_VERIFY:-0}"
CHECK_ONLY="${CHECK_ONLY:-0}"
RESET_STACK="${RESET_STACK:-0}"
DOCKER_WAIT_SECONDS="${DOCKER_WAIT_SECONDS:-180}"

usage() {
  cat <<'EOF'
Usage:
  ./be/scripts/ops/mac_local_start.sh [options]

Options:
  --no-install     Do not install missing Homebrew packages.
  --skip-verify    Start the stack but skip make verify.
  --check-only     Check local Mac dependencies without starting the stack.
  --reset          Run make down before startup. This removes local stack containers.
  -h, --help       Show this help.

Environment:
  INSTALL_MISSING_TOOLS=0  Same as --no-install.
  SKIP_VERIFY=1            Same as --skip-verify.
  CHECK_ONLY=1             Same as --check-only.
  RESET_STACK=1            Same as --reset.
  DOCKER_WAIT_SECONDS=180  Seconds to wait for Docker Desktop.
EOF
}

while [[ "$#" -gt 0 ]]; do
  case "$1" in
    --no-install)
      INSTALL_MISSING_TOOLS=0
      ;;
    --skip-verify)
      SKIP_VERIFY=1
      ;;
    --check-only)
      CHECK_ONLY=1
      ;;
    --reset)
      RESET_STACK=1
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "unknown option: $1"
      usage
      exit 1
      ;;
  esac
  shift
done

info() {
  printf '[be-ads] %s\n' "$*"
}

warn() {
  printf '[be-ads][warn] %s\n' "$*" >&2
}

die() {
  printf '[be-ads][error] %s\n' "$*" >&2
  exit 1
}

has_cmd() {
  command -v "$1" >/dev/null 2>&1
}

ensure_macos() {
  [[ "$(uname -s)" == "Darwin" ]] || die "this bootstrap script is for macOS only"
}

ensure_xcode_cli_tools() {
  if xcode-select -p >/dev/null 2>&1; then
    return 0
  fi

  warn "Xcode Command Line Tools are not installed."
  warn "A macOS installer prompt will open. Re-run this script after it finishes."
  xcode-select --install >/dev/null 2>&1 || true
  exit 1
}

ensure_homebrew() {
  if has_cmd brew; then
    return 0
  fi

  die "Homebrew is required. Install it from https://brew.sh, then re-run this script."
}

brew_install_formula() {
  local formula="$1"
  if brew list --formula "${formula}" >/dev/null 2>&1; then
    return 0
  fi
  if [[ "${INSTALL_MISSING_TOOLS}" != "1" ]]; then
    die "missing Homebrew formula: ${formula}"
  fi
  info "installing ${formula} with Homebrew"
  brew install "${formula}"
}

brew_install_cask() {
  local cask="$1"
  if brew list --cask "${cask}" >/dev/null 2>&1; then
    return 0
  fi
  if [[ "${INSTALL_MISSING_TOOLS}" != "1" ]]; then
    die "missing Homebrew cask: ${cask}"
  fi
  info "installing ${cask} with Homebrew"
  brew install --cask "${cask}"
}

ensure_toolchain() {
  brew_install_formula go
  brew_install_formula node

  has_cmd go || die "go is still unavailable after installation"
  has_cmd npm || die "npm is still unavailable after installation"
  has_cmd curl || die "curl is unavailable"
  has_cmd lsof || die "lsof is unavailable"
  has_cmd make || die "make is unavailable"
}

ensure_docker_desktop() {
  if ! has_cmd docker || [[ ! -d "/Applications/Docker.app" && ! -d "${HOME}/Applications/Docker.app" ]]; then
    brew_install_cask docker
  fi

  has_cmd docker || die "docker command is unavailable"

  if docker info >/dev/null 2>&1; then
    return 0
  fi

  info "starting Docker Desktop"
  if [[ -d "/Applications/Docker.app" || -d "${HOME}/Applications/Docker.app" ]]; then
    open -a Docker >/dev/null 2>&1 || true
  fi

  local waited=0
  while ! docker info >/dev/null 2>&1; do
    if (( waited >= DOCKER_WAIT_SECONDS )); then
      die "Docker Desktop is not ready after ${DOCKER_WAIT_SECONDS}s"
    fi
    sleep 3
    waited=$((waited + 3))
    printf '.'
  done
  printf '\n'
}

ensure_ports_free() {
  local ports=(8080 3307 3308 8123 9000 4222 8222)
  local busy=0
  for port in "${ports[@]}"; do
    if lsof -nP -iTCP:"${port}" -sTCP:LISTEN >/dev/null 2>&1; then
      warn "port ${port} is already in use"
      lsof -nP -iTCP:"${port}" -sTCP:LISTEN || true
      busy=1
    fi
  done
  if [[ "${busy}" == "1" ]]; then
    die "required local ports are busy; stop the conflicting process and re-run"
  fi
}

install_frontend_dependencies() {
  if [[ ! -f "${FE_DIR}/package.json" ]]; then
    die "frontend package.json not found at ${FE_DIR}"
  fi
  info "installing frontend dependencies"
  if [[ -f "${FE_DIR}/package-lock.json" ]]; then
    npm --prefix "${FE_DIR}" ci
  else
    npm --prefix "${FE_DIR}" install
  fi
}

reset_existing_stack() {
  cd "${BE_DIR}"
  if [[ "${RESET_STACK}" == "1" ]]; then
    info "resetting local stack"
    make down || true
  fi
}

run_startup() {
  cd "${BE_DIR}"

  info "starting local stack"
  make up

  info "waiting for bi-api health"
  local waited=0
  until curl -fsS http://127.0.0.1:8080/healthz >/dev/null 2>&1; do
    if (( waited >= 60 )); then
      make status || true
      die "bi-api did not become healthy after 60s"
    fi
    sleep 2
    waited=$((waited + 2))
  done

  if [[ "${SKIP_VERIFY}" != "1" ]]; then
    info "running local verification"
    make verify
  fi

  cat <<EOF

be_ads local stack is ready.
  control panel: http://127.0.0.1:8080/
  bi panel:      http://127.0.0.1:8080/bi

Useful commands:
  cd ${BE_DIR}
  make status
  make down
EOF
}

main() {
  ensure_macos
  ensure_xcode_cli_tools
  ensure_homebrew
  ensure_toolchain
  ensure_docker_desktop

  if [[ "${CHECK_ONLY}" == "1" ]]; then
    info "Mac local dependency check passed"
    exit 0
  fi

  reset_existing_stack
  ensure_ports_free
  install_frontend_dependencies
  run_startup
}

main "$@"
