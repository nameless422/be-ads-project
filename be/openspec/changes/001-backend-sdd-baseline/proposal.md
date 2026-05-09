# Change: Backend SDD Baseline

## Why

The repository now has a clean `be/` and `fe/` split. The backend needs the same SDD structure as the frontend so future backend work can start from specs instead of ad hoc notes.

## What Changes

- Add backend OpenSpec project metadata under `be/openspec`.
- Add backend capability specs for runtime, BI API, and data pipeline behavior.
- Add backend Spec Kit style feature documentation under `be/specs`.
- Add backend constitution under `be/.specify`.
- Update `be/README.md` to show where SDD artifacts live.

## Out Of Scope

- Changing Go runtime behavior.
- Changing API contracts.
- Changing database schemas.
- Adding new scripts.

