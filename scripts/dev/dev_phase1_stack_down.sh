#!/usr/bin/env bash

set -euo pipefail

docker rm -f \
  be-ads-raw-mysql \
  be-ads-serving-mysql \
  be-ads-clickhouse \
  be-ads-nats \
  be-ads-debezium >/dev/null 2>&1 || true

echo "removed phase1 stack containers"
