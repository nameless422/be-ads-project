#!/usr/bin/env bash

set -euo pipefail

docker rm -f be-ads-debezium >/dev/null 2>&1 || true
echo "removed debezium container"
