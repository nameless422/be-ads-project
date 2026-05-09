# Implementation Plan

## Architecture

Backend SDD files live inside `be/`:

```text
be/openspec -> lightweight change proposals and durable capability specs
be/specs    -> heavier Spec Kit feature specs
be/.specify -> backend constitution and long-term rules
```

## Steps

1. Add OpenSpec backend project metadata.
2. Add capability specs for runtime, BI API, and data pipeline.
3. Add Spec Kit baseline files.
4. Link the workflow from `be/README.md`.
5. Verify the backend still builds and tests.

## Verification

```bash
cd be
go test ./...
```

For runtime changes, also run:

```bash
make up
make verify
make down
```

