# Feature Spec: Backend SDD Baseline

## User Story

As a developer, I want backend changes to have OpenSpec and Spec Kit homes so that API, data pipeline, runtime, and local operations work can be planned and maintained consistently.

## Acceptance Criteria

- `be/openspec` exists for lightweight backend change specs.
- `be/openspec/specs` contains durable backend capability specs.
- `be/specs` exists for Spec Kit style feature work.
- `be/.specify/memory/constitution.md` captures backend project rules.
- `be/README.md` points developers to both SDD workflows.
- `go test ./...` passes from `be/`.

## Non-Goals

- Rewriting existing Go services.
- Reintroducing old root-level docs.
- Creating extra lifecycle scripts.

