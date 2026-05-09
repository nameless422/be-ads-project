# Backend Runtime Spec

## Capability

The backend runs the local ads data pipeline and BI API as a small set of Go services.

## Requirements

### Requirement: service entrypoints

The backend shall keep these primary service entrypoints:

- `cmd/collector-worker`
- `cmd/transformer-worker`
- `cmd/control-plane`
- `cmd/bi-api`

Utility entrypoints may exist under `cmd/`, but they shall not become part of the default local runtime unless added to this spec.

### Requirement: local lifecycle

The backend shall expose local lifecycle commands through `Makefile`:

- `make up`
- `make start`
- `make status`
- `make verify`
- `make stop`
- `make down`
- `make test`

### Requirement: script surface

The backend shall keep the active script surface limited to:

- `scripts/dev/dev_base_stack_up.sh`
- `scripts/dev/dev_base_stack_down.sh`
- `scripts/dev/dev_debezium_up.sh`
- `scripts/dev/dev_debezium_down.sh`
- `scripts/ops/up.sh`
- `scripts/ops/down.sh`
- `scripts/ops/start.sh`
- `scripts/ops/stop.sh`
- `scripts/ops/status.sh`
- `scripts/ops/mac_local_start.sh`
- `scripts/verify/common.sh`
- `scripts/verify/verify_local_stack.sh`
- `scripts/verify/verify_debezium_pipeline.sh`

### Requirement: Mac new-environment startup

The backend shall provide a macOS one-command startup path that checks the required local developer dependencies, prepares frontend dependencies, starts the local stack, and runs the standard verification flow.

### Requirement: runtime artifacts

Runtime artifacts shall be written under `logs/`, `run/`, or `tmp/`, and these directories shall remain ignored.
