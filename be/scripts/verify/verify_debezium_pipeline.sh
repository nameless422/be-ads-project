#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "${SCRIPT_DIR}/common.sh"

setup_verify_env
fetch_verify_payloads
assert_common_api_health
assert_debezium_snapshot_mode

outbox_total="$(mysql_scalar be-ads-raw-mysql be_ads_raw 'select count(*) from outbox_events;')"

print_common_storage_counts
echo "outbox_total=${outbox_total}"
