# Agent Rules

This repository uses the project Harness documented in [HARNESS.md](HARNESS.md). Follow these rules before changing files.

## Required Context

- Read [HARNESS.md](HARNESS.md) first for the project workflow.
- Read [docs/harness/dev-map.md](docs/harness/dev-map.md) before code changes.
- For complex or cross-module work, create or update a SPEC from [docs/harness/tasks/_template/01-spec.md](docs/harness/tasks/_template/01-spec.md).

## Boundaries

- Keep backend code inside the existing `be/cmd/`, `be/internal/modules/`, `be/internal/shared/`, and `be/internal/platform/` structure.
- Do not create a parallel business structure when an existing module owns the behavior.
- Do not edit `be/vendor/` unless the task explicitly requires vendored dependency maintenance.
- Do not treat `fe/dist` or `fe/node_modules` as frontend source. Restore or edit `fe/src` when frontend source work is required.
- Do not store secrets, OAuth tokens, `.env`, logs, run files, or generated frontend dependencies in git.

## Validation

- Run `make harness-check` before handing off work.
- Run `make test` for Go code changes.
- Run `make verify` when the local data path, service lifecycle, BI API, storage, or messaging behavior changes.
- Run `make verify-debezium` when outbox, Debezium, raw events, or CDC behavior changes.
- If a required tool is missing, report the exact missing tool and the commands that did run.

## Documentation

- Update `docs/harness/task-board.md` when a task changes stage, blocks, or reaches a durable conclusion.
- Update `docs/harness/dev-map.md` when module entrypoints, ownership, or frontend source layout changes.
- Update `docs/harness/playbook.md`, `scripts/README.md`, `be/scripts/README.md`, and `Makefile help` when validation commands change.
