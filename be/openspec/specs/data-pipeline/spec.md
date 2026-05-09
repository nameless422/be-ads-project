# Data Pipeline Spec

## Capability

The backend ingests platform data, stores raw records, transforms them into normalized BI records, and projects them into serving stores.

## Requirements

### Requirement: collection boundary

The collection module shall own sync targets, connector execution, raw record persistence, and raw outbox publishing.

### Requirement: transformation boundary

The transformation module shall own raw event consumption, platform normalization, and projection fanout.

### Requirement: reporting boundary

The reporting module shall own BI read models, query repositories, HTTP API handlers, and local control surfaces.

### Requirement: control-plane boundary

The control-plane module shall own job construction and dispatch. Worker subscription and shard assignment behavior shall remain explicit in the service entrypoints.

### Requirement: verification

The local verification flow shall prove at least:

- `GET /healthz` is healthy.
- BI query endpoints return expected seeded data.
- raw, serving, and ClickHouse storage counts are readable.
- local stack scripts can start and stop the required dependencies.

