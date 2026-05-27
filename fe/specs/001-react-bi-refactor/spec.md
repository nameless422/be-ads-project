# Feature Spec: React BI Refactor

## User Story

As a developer and BI user, I want the dashboard frontend to live in a dedicated React project so that UI changes can be developed, reviewed, and verified independently from backend business logic.

## Acceptance Criteria

- The frontend source lives under `fe/`.
- The backend source lives under `be/`.
- Production `/bi` routes are served by `be/cmd/bi-api` from `fe/dist`.
- `npm run build` succeeds from `fe/`.
- `go test ./...` succeeds from `be/`.
- Root directory contains no scattered scripts, docs, screenshots, runtime logs, or build artifacts.

## Non-Goals

- Replacing Go APIs.
- Introducing a component library before repeated UI patterns justify it.
- Reworking ingestion, transformation, or BI storage semantics.
