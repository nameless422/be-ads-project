# Design: Backend SDD Baseline

## Structure

```text
be/
  openspec/
    project.md
    specs/
    changes/
  specs/
  .specify/
```

## OpenSpec Usage

Use `openspec/` for lightweight backend changes:

- `changes/<change-id>/proposal.md`: why and what
- `changes/<change-id>/design.md`: implementation shape
- `changes/<change-id>/tasks.md`: execution checklist
- `specs/<capability>/spec.md`: durable capability requirements

## Spec Kit Usage

Use `specs/<number>-<feature>/` for heavier feature work that needs a fuller lifecycle:

- `spec.md`: feature intent and acceptance criteria
- `plan.md`: implementation plan
- `tasks.md`: execution breakdown
- `quickstart.md`: verification and local usage

## Boundary

Backend SDD artifacts describe Go services, storage, messaging, APIs, and local operations. Frontend UI design and React component work belong in `../fe`.

