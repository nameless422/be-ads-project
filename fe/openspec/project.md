# OpenSpec Project

## Purpose

`fe` is the React BI frontend for the ads reporting project. It is served by `be/cmd/bi-api` under `/bi` and talks to existing `/api/bi/*` and `/api/control/*` endpoints.

## Scope

- Keep frontend changes small, reviewable, and tied to a concrete change folder.
- Do not change backend API contracts from frontend-only changes.
- Keep production access through `http://127.0.0.1:8080/bi`.
- Keep development access through Vite from this directory.

## Workflow

1. Create or update a folder under `openspec/changes/<change-id>/`.
2. Write `proposal.md`, `design.md`, and `tasks.md`.
3. Update affected capability specs under `openspec/specs/`.
4. Implement only the tasks accepted for that change.
5. Verify with `npm run build` and backend smoke checks when `/bi` serving changes.

