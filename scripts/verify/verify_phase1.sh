#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "missing required command: $1"
    exit 1
  fi
}

require_cmd curl
require_cmd docker
require_cmd python3

tmp_dir="$(mktemp -d)"
trap 'rm -rf "${tmp_dir}"' EXIT

health_file="${tmp_dir}/health.json"
snapshots_file="${tmp_dir}/snapshots.json"
campaigns_file="${tmp_dir}/campaigns.json"
insights_file="${tmp_dir}/insights.json"

curl -sf http://127.0.0.1:8080/healthz >"${health_file}"
curl -sf http://127.0.0.1:8080/api/bi/snapshots >"${snapshots_file}"
curl -sf 'http://127.0.0.1:8080/api/bi/campaigns?platform=google_ads&account_id=248-390-1805' >"${campaigns_file}"
curl -sf 'http://127.0.0.1:8080/api/bi/insights/summary?platform=tiktok_ads&platform_account_id=acct_tt_001&date_from=2026-04-22&date_to=2026-04-22' >"${insights_file}"

python3 - "${health_file}" "${snapshots_file}" "${campaigns_file}" "${insights_file}" <<'PY'
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

raw_count="$(docker exec be-ads-raw-mysql mysql -ube_ads -pbe_ads -D be_ads_raw -Nse 'select count(*) from raw_records;' | tr -d '\r')"
outbox_published="$(docker exec be-ads-raw-mysql mysql -ube_ads -pbe_ads -D be_ads_raw -Nse "select count(*) from outbox_events where status='published';" | tr -d '\r')"
outbox_pending="$(docker exec be-ads-raw-mysql mysql -ube_ads -pbe_ads -D be_ads_raw -Nse "select count(*) from outbox_events where status='pending';" | tr -d '\r')"
snapshot_count="$(docker exec be-ads-serving-mysql mysql -ube_ads -pbe_ads -D be_ads_serving -Nse 'select count(*) from bi_account_snapshots;' | tr -d '\r')"
campaign_count="$(docker exec be-ads-serving-mysql mysql -ube_ads -pbe_ads -D be_ads_serving -Nse 'select count(*) from oltp_campaigns;' | tr -d '\r')"
insight_count="$(docker exec be-ads-clickhouse clickhouse-client --user be_ads --password be_ads --database be_ads --query 'select count() from olap_insights' | tr -d '\r')"

echo "raw_records=${raw_count}"
echo "outbox_published=${outbox_published}"
echo "outbox_pending=${outbox_pending}"
echo "bi_account_snapshots=${snapshot_count}"
echo "oltp_campaigns=${campaign_count}"
echo "olap_insights=${insight_count}"
