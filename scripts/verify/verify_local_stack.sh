#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "${SCRIPT_DIR}/common.sh"

setup_verify_env
fetch_verify_payloads
assert_common_api_health

outbox_published="$(mysql_scalar be-ads-raw-mysql be_ads_raw "select count(*) from outbox_events where status='published';")"
outbox_pending="$(mysql_scalar be-ads-raw-mysql be_ads_raw "select count(*) from outbox_events where status='pending';")"

print_common_storage_counts
echo "outbox_published=${outbox_published}"
echo "outbox_pending=${outbox_pending}"
