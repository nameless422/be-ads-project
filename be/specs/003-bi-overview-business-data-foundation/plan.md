# Implementation Plan

## Architecture

Phase 2 should treat BI Overview metrics as a data contract, not a frontend-only calculation. The chain should be explicit:

```text
source of truth
  -> collector / external ingest
  -> normalized model
  -> MySQL / ClickHouse projection
  -> reporting read model
  -> /api/bi/*
  -> React API types and metric helpers
  -> Overview KPI, trend, and summary table
```

## Workstreams

1. Confirm business definitions.
   - Product source.
   - Media source terminology.
   - Region/country ownership.
   - Purchase, CPA, CVR formulas.
   - D0 LTV, D0 ROAS, D7 ROAS formulas.

2. Design backend contract.
   - Decide whether to extend `/api/bi/ua-report` or add a dedicated Overview summary endpoint.
   - Keep existing pages compatible.
   - Define request filters for product, media source, region/country, device/platform, campaign, ad group, and ad.
   - Define response fields for KPI cards, daily trend, and summary table.

3. Extend data models.
   - Add only confirmed dimensions and metrics to shared normalized models.
   - Add MySQL migration for game-side KPI/product fields if the source is game-side.
   - Add ClickHouse migration if ad-side dimensions or summary metrics must live in OLAP.
   - Preserve idempotent `CREATE TABLE IF NOT EXISTS` / `ALTER TABLE ADD COLUMN IF NOT EXISTS` style.

4. Implement projection and reporting.
   - Carry new fields through normalizers and projectors.
   - Update `UAReportService` or new overview reporting service.
   - Derive ratio metrics from summed numerator/denominator inputs.
   - Keep unavailable metrics explicit instead of returning misleading zeroes.

5. Update frontend consumption.
   - Update `fe/src/api/types.ts`.
   - Update `fe/src/api/client.ts`.
   - Update metric aggregation helpers.
   - Wire Phase 1 Overview UI to real Phase 2 fields.

## Key Decisions

| Decision | Default recommendation |
| --- | --- |
| Product source | Prefer an explicit mapping/config or game-side dimension over inferring from campaign names. |
| `media_source` backend naming | Keep storage field `platform` initially, expose UI/API alias only if compatibility is clear. |
| D0 metrics | Require true D0 revenue/LTV source. |
| ROAS basis | Use revenue window / spend at matching grain and date range. |
| Ratio aggregation | Use aggregate numerator / aggregate denominator. |

## Verification

Backend:

```bash
cd be
go test ./...
```

Frontend:

```bash
cd fe
npm run build
```

Local smoke:

```bash
cd be
make up
make verify
curl "http://127.0.0.1:8080/api/bi/ua-report?date_from=2026-04-01&date_to=2026-04-30"
```

UI smoke:

```text
http://127.0.0.1:8080/bi/overview
```
