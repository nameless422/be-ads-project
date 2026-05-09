#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
RUN_DIR="${ROOT_DIR}/run/debezium"
DATA_DIR="${RUN_DIR}/data"
CONFIG_FILE="${RUN_DIR}/application.properties"
CONTAINER_NAME="be-ads-debezium"
DEBEZIUM_IMAGE="${DEBEZIUM_IMAGE:-quay.io/debezium/server:3.4.3.Final}"

mkdir -p "${DATA_DIR}"

if ! docker ps --format '{{.Names}}' | grep -qx "be-ads-raw-mysql"; then
  echo "raw mysql is not running; start the base stack first with ./scripts/dev/dev_base_stack_up.sh"
  exit 1
fi

cat >"${CONFIG_FILE}" <<'EOF'
debezium.source.connector.class=io.debezium.connector.mysql.MySqlConnector
debezium.source.database.hostname=host.docker.internal
debezium.source.database.port=3307
debezium.source.database.user=debezium
debezium.source.database.password=debezium
debezium.source.database.server.id=5401
debezium.source.database.server.name=be_ads_raw_mysql
debezium.source.topic.prefix=be_ads_raw
debezium.source.database.include.list=be_ads_raw
debezium.source.table.include.list=be_ads_raw.outbox_events
debezium.source.include.schema.changes=false
debezium.source.snapshot.mode=no_data
debezium.source.snapshot.locking.mode=none
debezium.source.tombstones.on.delete=false
debezium.source.schema.history.internal=io.debezium.storage.file.history.FileSchemaHistory
debezium.source.schema.history.internal.file.filename=/debezium/data/schemahistory.dat
debezium.source.offset.storage=org.apache.kafka.connect.storage.FileOffsetBackingStore
debezium.source.offset.storage.file.filename=/debezium/data/offsets.dat
debezium.source.offset.flush.interval.ms=1000

debezium.format.key=json
debezium.format.key.schemas.enable=false
debezium.format.value=json
debezium.format.value.schemas.enable=false

debezium.transforms=outbox
debezium.transforms.outbox.type=io.debezium.transforms.outbox.EventRouter
debezium.transforms.outbox.table.field.event.id=event_id
debezium.transforms.outbox.table.field.event.key=aggregate_id
debezium.transforms.outbox.table.field.event.payload=payload_json
debezium.transforms.outbox.table.expand.json.payload=true
debezium.transforms.outbox.route.by.field=topic
debezium.transforms.outbox.route.topic.replacement=$${routedByValue}
debezium.transforms.outbox.table.fields.additional.placement=event_type:header:type

debezium.sink.type=nats-jetstream
debezium.sink.nats-jetstream.url=nats://host.docker.internal:4222
debezium.sink.nats-jetstream.create-stream=false
debezium.sink.nats-jetstream.subjects=raw.events.ingested
debezium.sink.nats-jetstream.async.enabled=true
EOF

docker rm -f "${CONTAINER_NAME}" >/dev/null 2>&1 || true
docker run -d \
  --name "${CONTAINER_NAME}" \
  -p 8083:8080 \
  -v "${CONFIG_FILE}:/debezium/config/application.properties:ro" \
  -v "${DATA_DIR}:/debezium/data" \
  "${DEBEZIUM_IMAGE}" >/dev/null

sleep 3
echo "debezium image: ${DEBEZIUM_IMAGE}"
docker logs --tail 40 "${CONTAINER_NAME}"
