#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"

cd "${ROOT_DIR}"

./scripts/dev/dev_base_stack_up.sh
./scripts/ops/start.sh
./scripts/ops/status.sh

cat <<EOF

be_ads is ready:
  control panel: http://127.0.0.1:8080/
  bi panel:      http://127.0.0.1:8080/bi
EOF
