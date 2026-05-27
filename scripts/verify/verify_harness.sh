#!/usr/bin/env bash

set -euo pipefail

export PATH="/opt/homebrew/bin:/usr/local/go/bin:/usr/local/bin:${PATH}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"

failures=0

info() {
  printf 'info: %s\n' "$1"
}

pass() {
  printf 'pass: %s\n' "$1"
}

fail() {
  printf 'fail: %s\n' "$1"
  failures=$((failures + 1))
}

require_file() {
  local path="$1"
  if [[ -f "${ROOT_DIR}/${path}" ]]; then
    pass "file exists: ${path}"
  else
    fail "missing file: ${path}"
  fi
}

require_executable() {
  local path="$1"
  if [[ -x "${ROOT_DIR}/${path}" ]]; then
    pass "executable: ${path}"
  else
    fail "not executable: ${path}"
  fi
}

check_harness_files() {
  info "checking harness docs"
  require_file "AGENTS.md"
  require_file "HARNESS.md"
  require_file ".github/workflows/harness-check.yml"
  require_file ".github/PULL_REQUEST_TEMPLATE.md"
  require_file ".github/ISSUE_TEMPLATE/feature_request.yml"
  require_file "docs/harness/README.md"
  require_file "docs/harness/dev-map.md"
  require_file "docs/harness/playbook.md"
  require_file "docs/harness/workflow.md"
  require_file "docs/harness/workflow.json"
  require_file "docs/harness/task-board.md"
  require_file "docs/harness/tasks/_template/01-spec.md"
  require_file "docs/harness/tasks/_template/02-design.md"
  require_file "docs/harness/tasks/_template/03-gate.md"
  require_file "docs/harness/tasks/_template/04-review.md"
  require_file "docs/harness/tasks/_template/05-validation.md"
  require_file "scripts/verify/verify_harness.sh"
  require_file "scripts/README.md"
  require_file "be/Makefile"
  require_file "be/README.md"
  require_file "be/go.mod"
  require_file "be/cmd/bi-api/main.go"
  require_file "be/internal/modules/reporting/infrastructure/httpapi/server.go"
  require_file "be/scripts/README.md"
  require_file "be/scripts/verify/verify_local_stack.sh"
  require_file "be/scripts/verify/verify_debezium_pipeline.sh"
  require_file "fe/README.md"
  require_file "fe/package.json"
  require_file "fe/src/App.tsx"
}

check_harness_references() {
  info "checking harness references"
  if grep -q 'make harness-check' "${ROOT_DIR}/.github/workflows/harness-check.yml"; then
    pass "CI runs make harness-check"
  else
    fail "CI workflow does not run make harness-check"
  fi

  if grep -q 'playbook.md' "${ROOT_DIR}/HARNESS.md"; then
    pass "HARNESS links playbook"
  else
    fail "HARNESS missing playbook link"
  fi

  if grep -q 'tasks/_template' "${ROOT_DIR}/docs/harness/playbook.md"; then
    pass "playbook links task templates"
  else
    fail "playbook missing task template link"
  fi

  if grep -q 'PULL_REQUEST_TEMPLATE.md' "${ROOT_DIR}/HARNESS.md"; then
    pass "HARNESS links PR template"
  else
    fail "HARNESS missing PR template link"
  fi

  if grep -q 'frontend-change' "${ROOT_DIR}/docs/harness/playbook.md"; then
    pass "playbook keeps task skills"
  else
    fail "playbook missing task skills"
  fi

  if grep -q 'Codex' "${ROOT_DIR}/docs/harness/playbook.md"; then
    pass "playbook keeps Codex usage"
  else
    fail "playbook missing Codex usage"
  fi
}

check_directory_layout() {
  info "checking frontend/backend directory layout"

  local root_backend_paths=(
    "cmd"
    "internal"
    "vendor"
    "deploy"
    "go.mod"
    "go.sum"
    "docker-compose.dev.yml"
    ".env.stack.example"
    ".env.storage.example"
    ".env.google-ads.example"
  )

  local path
  for path in "${root_backend_paths[@]}"; do
    if [[ -e "${ROOT_DIR}/${path}" ]]; then
      fail "backend path should live under be/: ${path}"
    fi
  done

  if [[ -d "${ROOT_DIR}/be/cmd" && -d "${ROOT_DIR}/be/internal" && -f "${ROOT_DIR}/be/go.mod" ]]; then
    pass "backend source lives under be/"
  else
    fail "backend source layout incomplete under be/"
  fi

  if [[ -d "${ROOT_DIR}/fe/src" && -f "${ROOT_DIR}/fe/package.json" ]]; then
    pass "frontend source lives under fe/"
  else
    fail "frontend source layout incomplete under fe/"
  fi
}

check_frontend_serving() {
  info "checking frontend serving boundary"
  local api_file="be/internal/modules/reporting/infrastructure/httpapi/server.go"

  if grep -q 'mux.HandleFunc("/bi", s.handleBIApp)' "${ROOT_DIR}/${api_file}" &&
    grep -q 'const biFrontendDistDir = "../fe/dist"' "${ROOT_DIR}/${api_file}"; then
    pass "BI frontend is served from fe/dist"
  else
    fail "BI frontend must be served from fe/dist"
  fi

  if grep -q 'mux.HandleFunc("/bi", s.handleBIPanel)' "${ROOT_DIR}/${api_file}"; then
    fail "BI route still points to backend-rendered panel"
  else
    pass "BI route does not use backend-rendered panel"
  fi
}

check_workflow_definition() {
  info "checking workflow definition"
  if ! command -v python3 >/dev/null 2>&1; then
    fail "python3 required to inspect workflow definition"
    return
  fi

  if python3 - "${ROOT_DIR}/docs/harness/workflow.json" <<'PY'
import json
import sys

path = sys.argv[1]
with open(path, "r", encoding="utf-8") as fh:
    data = json.load(fh)

stages = data.get("stages")
if not isinstance(stages, list) or not stages:
    raise SystemExit("workflow.json must contain a non-empty stages list")

stage_ids = {stage.get("id") for stage in stages}
required = {
    "pm-route",
    "requirements",
    "design",
    "gate",
    "implementation",
    "review",
    "validation",
    "pm-close",
}
missing = sorted(required - stage_ids)
if missing:
    raise SystemExit(f"missing workflow stages: {', '.join(missing)}")

for stage in stages:
    for key in ("id", "role", "input", "output", "next"):
        if key not in stage:
            raise SystemExit(f"stage {stage.get('id', '<unknown>')} missing {key}")

print(f"workflow_stages={len(stages)}")
PY
  then
    pass "workflow.json"
  else
    fail "workflow.json"
  fi
}

check_git_hygiene() {
  info "checking generated files are not tracked"
  local tracked
  tracked="$(cd "${ROOT_DIR}" && git ls-files \
    'fe/dist/*' \
    'fe/node_modules/*' \
    'logs/*' \
    'run/*' \
    'be/logs/*' \
    'be/run/*' || true)"

  if [[ -z "${tracked}" ]]; then
    pass "generated runtime/frontend files are untracked"
  else
    printf '%s\n' "${tracked}"
    fail "generated runtime/frontend files are tracked"
  fi
}

check_portability() {
  info "checking clone portability"
  local matches
  matches="$(cd "${ROOT_DIR}" && rg -n '/Users/zhongyi\\.zhang|file://' \
    --glob '!be/vendor/**' \
    --glob '!fe/node_modules/**' \
    --glob '!fe/dist/**' \
    --glob '!scripts/verify/verify_harness.sh' \
    . || true)"

  if [[ -z "${matches}" ]]; then
    pass "no local absolute paths in tracked project files"
  else
    printf '%s\n' "${matches}"
    fail "local absolute paths found"
  fi
}

check_secret_placeholders() {
  info "checking obvious secret placeholders"
  local matches
  matches="$(cd "${ROOT_DIR}" && rg -n 'yOCuBAsqVI0UyT94T7jDow' \
    --glob '!be/vendor/**' \
    --glob '!fe/node_modules/**' \
    --glob '!fe/dist/**' \
    --glob '!scripts/verify/verify_harness.sh' \
    . || true)"

  if [[ -z "${matches}" ]]; then
    pass "no known local Google Ads token literal"
  else
    printf '%s\n' "${matches}"
    fail "known local Google Ads token literal found"
  fi
}

check_markdown_links() {
  info "checking local markdown links"
  if ! command -v python3 >/dev/null 2>&1; then
    fail "python3 required to inspect markdown links"
    return
  fi

  if python3 - "${ROOT_DIR}" <<'PY'
from pathlib import Path
import re
import sys
from urllib.parse import unquote

root = Path(sys.argv[1])
files = [
    root / "AGENTS.md",
    root / "HARNESS.md",
    root / "CONTRIBUTING.md",
    root / "README.md",
    root / "scripts" / "README.md",
    root / "be" / "scripts" / "README.md",
    root / "be" / "README.md",
    root / "fe" / "README.md",
]
files.extend(sorted((root / "docs" / "harness").rglob("*.md")))

missing = []
for path in files:
    if not path.exists():
        continue
    text = path.read_text(encoding="utf-8")
    for match in re.findall(r"(?<!!)\[[^\]]+\]\(([^)]+)\)", text):
        ref = match.strip()
        if not ref or ref.startswith(("#", "http://", "https://", "mailto:", "data:")):
            continue
        if ref.startswith("/"):
            continue
        if ref.startswith("<") and ref.endswith(">"):
            ref = ref[1:-1]
        ref = unquote(ref.split("#", 1)[0])
        if not ref:
            continue
        target = (path.parent / ref).resolve()
        try:
            target.relative_to(root.resolve())
        except ValueError:
            continue
        if not target.exists():
            missing.append(f"{path.relative_to(root)} -> {match}")

if missing:
    print("missing markdown links:")
    for item in missing:
        print(f"- {item}")
    raise SystemExit(1)
PY
  then
    pass "local markdown links"
  else
    fail "local markdown links"
  fi
}

check_makefile_entry() {
  info "checking Makefile harness entry"
  if grep -q '^harness-check:' "${ROOT_DIR}/Makefile"; then
    pass "Makefile target: harness-check"
  else
    fail "Makefile missing harness-check target"
  fi

  if grep -q '^harness-check:' "${ROOT_DIR}/be/Makefile"; then
    pass "be/Makefile target: harness-check"
  else
    fail "be/Makefile missing harness-check target"
  fi
}

check_shell_scripts() {
  info "checking shell scripts"
  require_executable "scripts/verify/verify_harness.sh"
  require_executable "be/scripts/verify/verify_local_stack.sh"
  require_executable "be/scripts/verify/verify_debezium_pipeline.sh"

  local script
  while IFS= read -r script; do
    if bash -n "${ROOT_DIR}/${script}"; then
      pass "bash syntax: ${script}"
    else
      fail "bash syntax failed: ${script}"
    fi
  done < <(cd "${ROOT_DIR}" && find scripts be/scripts -name '*.sh' -type f | sort)
}

check_go() {
  if [[ "${BE_HARNESS_SKIP_GO:-0}" == "1" ]]; then
    info "skipping Go checks because BE_HARNESS_SKIP_GO=1"
    return
  fi

  info "checking Go formatting and tests"
  if ! command -v go >/dev/null 2>&1; then
    fail "missing go command; install Go or use BE_HARNESS_SKIP_GO=1 for docs-only checks"
    return
  fi

  local gofmt_output
  gofmt_output="$(cd "${ROOT_DIR}/be" && gofmt -l cmd internal)"
  if [[ -z "${gofmt_output}" ]]; then
    pass "gofmt clean: be/cmd be/internal"
  else
    printf '%s\n' "${gofmt_output}"
    fail "gofmt reported unformatted files"
  fi

  if (cd "${ROOT_DIR}/be" && go test ./...); then
    pass "go test ./... in be/"
  else
    fail "go test ./... in be/"
  fi
}

check_frontend() {
  info "checking frontend entry"

  if [[ -f "${ROOT_DIR}/fe/package.json" ]]; then
    if ! command -v npm >/dev/null 2>&1; then
      fail "fe/package.json exists but npm is missing"
      return
    fi

    if [[ -f "${ROOT_DIR}/fe/package-lock.json" && ! -d "${ROOT_DIR}/fe/node_modules" ]]; then
      if (cd "${ROOT_DIR}/fe" && npm ci); then
        pass "frontend dependencies installed"
      else
        fail "frontend dependencies install"
        return
      fi
    fi

    if (cd "${ROOT_DIR}/fe" && npm run build --if-present); then
      pass "frontend build script"
    else
      fail "frontend build script"
    fi
    return
  fi

  if [[ -f "${ROOT_DIR}/fe/dist/index.html" ]]; then
    if ! command -v python3 >/dev/null 2>&1; then
      fail "python3 required to inspect fe/dist assets"
      return
    fi

    if python3 - "${ROOT_DIR}/fe/dist/index.html" <<'PY'
from pathlib import Path
import re
import sys

index = Path(sys.argv[1])
root = index.parent
html = index.read_text(encoding="utf-8")
missing = []
for ref in re.findall(r'''(?:src|href)=["']([^"']+)["']''', html):
    if ref.startswith(("http://", "https://", "data:", "#")):
        continue
    normalized = ref.lstrip("/")
    candidate = root / normalized
    if not candidate.exists() and "/assets/" in f"/{normalized}":
        asset_suffix = normalized.split("assets/", 1)[1]
        candidate = root / "assets" / asset_suffix
    if not candidate.exists():
        missing.append(ref)

if missing:
    print("missing frontend assets:")
    for ref in missing:
        print(f"- {ref}")
    raise SystemExit(1)
PY
    then
      pass "fe/dist asset references"
    else
      fail "fe/dist asset references"
    fi
  else
    info "no fe/package.json or fe/dist/index.html; frontend checks skipped"
  fi
}

main() {
  check_harness_files
  check_harness_references
  check_directory_layout
  check_frontend_serving
  check_workflow_definition
  check_git_hygiene
  check_portability
  check_secret_placeholders
  check_markdown_links
  check_makefile_entry
  check_shell_scripts
  check_go
  check_frontend

  if [[ "${failures}" -ne 0 ]]; then
    printf 'harness verification failed: %d failure(s)\n' "${failures}"
    exit 1
  fi

  printf 'harness verification passed\n'
}

main "$@"
