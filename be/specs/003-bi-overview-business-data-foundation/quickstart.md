# Quickstart

## Read The Phase Split

Phase 1 frontend OpenSpec:

```bash
sed -n '1,220p' ../fe/openspec/changes/002-bi-overview-user-feedback/proposal.md
sed -n '1,220p' ../fe/openspec/changes/002-bi-overview-user-feedback/design.md
sed -n '1,160p' ../fe/openspec/changes/002-bi-overview-user-feedback/tasks.md
```

Phase 2 backend/data Spec Kit:

```bash
sed -n '1,220p' specs/003-bi-overview-business-data-foundation/spec.md
sed -n '1,220p' specs/003-bi-overview-business-data-foundation/plan.md
sed -n '1,220p' specs/003-bi-overview-business-data-foundation/tasks.md
```

## Inspect Current Data Chain

```bash
rg -n "UAReport|GameKPI|bi_game_kpis|olap_insights|platform|country|retention|ltv|roas" internal
```

Key files:

```text
internal/shared/ads/normalized.go
internal/modules/reporting/domain/read_models.go
internal/modules/reporting/application/ua_report_service.go
internal/modules/reporting/infrastructure/mysql/repository.go
internal/modules/reporting/infrastructure/clickhouse/repository.go
internal/modules/transformation/infrastructure/projector/clickhouse/projector.go
internal/modules/transformation/infrastructure/projector/mysql/projector.go
internal/modules/reporting/infrastructure/httpapi/server.go
```

## Verify Current Baseline

Backend:

```bash
go test ./...
```

Frontend:

```bash
cd ../fe
npm run build
```

Local stack:

```bash
cd ../be
make up
make verify
curl "http://127.0.0.1:8080/api/bi/ua-report"
```

Open UI:

```text
http://127.0.0.1:8080/bi/overview
```

## Implementation Rule

Do not implement D0 LTV, D0 ROAS, D7 ROAS, CPA, or CVR until the metric source and formula are confirmed. Missing metrics should remain explicit empty states rather than silently returning zero or a proxy field.
