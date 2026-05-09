# OpenSpec Project

## Purpose

`be` is the Go backend for ads ingestion, transformation, BI reporting, and local operational control.

It owns:

- service entrypoints under `cmd/`
- business modules under `internal/modules/`
- local lifecycle scripts under `scripts/`
- BI and control APIs served by `cmd/bi-api`

## Scope

- Keep backend behavior changes tied to a concrete OpenSpec change folder.
- Keep API, storage, messaging, and worker behavior explicit before implementation.
- Keep frontend source ownership in `../fe`.
- Keep production `/bi` static serving from `../fe/dist`.
- Keep root-level repository shape limited to `be/` and `fe/`.

## Workflow

1. Create or update `openspec/changes/<change-id>/`.
2. Write `proposal.md`, `design.md`, and `tasks.md`.
3. Update affected capability specs under `openspec/specs/`.
4. Implement only the accepted tasks.
5. Verify with `go test ./...`.
6. If runtime behavior changes, also run `make up`, `make verify`, and `make down`.

