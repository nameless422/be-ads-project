# Quickstart

## Read Current Backend Specs

```bash
ls openspec/specs
ls specs
```

## Start A Lightweight Backend Change

```bash
mkdir -p openspec/changes/002-change-name
touch openspec/changes/002-change-name/proposal.md
touch openspec/changes/002-change-name/design.md
touch openspec/changes/002-change-name/tasks.md
```

## Start A Heavier Spec Kit Feature

```bash
mkdir -p specs/002-feature-name
touch specs/002-feature-name/spec.md
touch specs/002-feature-name/plan.md
touch specs/002-feature-name/tasks.md
touch specs/002-feature-name/quickstart.md
```

## Verify Backend

```bash
go test ./...
```

