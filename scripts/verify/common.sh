#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "missing required command: $1"
    exit 1
  fi
}

setup_verify_env() {
  require_cmd curl
  require_cmd docker
  require_cmd python3

  VERIFY_TMP_DIR="$(mktemp -d)"
  trap 'rm -rf "${VERIFY_TMP_DIR}"' EXIT

  HEALTH_FILE="${VERIFY_TMP_DIR}/health.json"
  SNAPSHOTS_FILE="${VERIFY_TMP_DIR}/snapshots.json"
  CAMPAIGNS_FILE="${VERIFY_TMP_DIR}/campaigns.json"
  INSIGHTS_FILE="${VERIFY_TMP_DIR}/insights.json"
}

fetch_verify_payloads() {
  curl -sf http://127.0.0.1:8080/healthz >"${HEALTH_FILE}"
  curl -sf http://127.0.0.1:8080/api/bi/snapshots >"${SNAPSHOTS_FILE}"
  curl -sf 'http://127.0.0.1:8080/api/bi/campaigns?platform=google_ads&account_id=248-390-1805' >"${CAMPAIGNS_FILE}"
  curl -sf 'http://127.0.0.1:8080/api/bi/insights/summary?platform=tiktok_ads&platform_account_id=acct_tt_001&date_from=2026-04-22&date_to=2026-04-22' >"${INSIGHTS_FILE}"
}

assert_common_api_health() {
  python3 - "${HEALTH_FILE}" "${SNAPSHOTS_FILE}" "${CAMPAIGNS_FILE}" "${INSIGHTS_FILE}" <<'PY'
import json
import sys

health = json.load(open(sys.argv[1], "r", encoding="utf-8"))
snapshots = json.load(open(sys.argv[2], "r", encoding="utf-8"))
campaigns = json.load(open(sys.argv[3], "r", encoding="utf-8"))
insights = json.load(open(sys.argv[4], "r", encoding="utf-8"))

assert health["status"] == "ok", health
assert len(snapshots["items"]) >= 8, snapshots
assert len(campaigns["items"]) >= 1, campaigns
assert len(insights["items"]) >= 1, insights

print("health=ok")
print(f"snapshots={len(snapshots['items'])}")
print(f"google_campaigns={len(campaigns['items'])}")
print(f"insight_rows={len(insights['items'])}")
PY
}

assert_debezium_snapshot_mode() {
  python3 - "${SNAPSHOTS_FILE}" <<'PY'
import json
import sys

snapshots = json.load(open(sys.argv[1], "r", encoding="utf-8"))
assert all(item["LastSourceMode"] == "jetstream_async" for item in snapshots["items"]), snapshots
print("snapshot_source_mode=jetstream_async")
PY
}

mysql_scalar() {
  local container="$1"
  local database="$2"
  local query="$3"
  docker exec "${container}" mysql -ube_ads -pbe_ads -D "${database}" -Nse "${query}" | tr -d '\r'
}

clickhouse_scalar() {
  local query="$1"
  docker exec be-ads-clickhouse clickhouse-client --user be_ads --password be_ads --database be_ads --query "${query}" | tr -d '\r'
}

print_common_storage_counts() {
  local raw_count
  local snapshot_count
  local campaign_count
  local insight_count

  raw_count="$(mysql_scalar be-ads-raw-mysql be_ads_raw 'select count(*) from raw_records;')"
  snapshot_count="$(mysql_scalar be-ads-serving-mysql be_ads_serving 'select count(*) from bi_account_snapshots;')"
  campaign_count="$(mysql_scalar be-ads-serving-mysql be_ads_serving 'select count(*) from oltp_campaigns;')"
  insight_count="$(clickhouse_scalar 'select count() from olap_insights')"

  echo "raw_records=${raw_count}"
  echo "bi_account_snapshots=${snapshot_count}"
  echo "oltp_campaigns=${campaign_count}"
  echo "olap_insights=${insight_count}"
}
