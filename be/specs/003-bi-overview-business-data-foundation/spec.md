# Feature Spec: BI Overview Business Data Foundation

## User Story

As a UA business user, I want BI Overview to support product, media source, region, device/platform, campaign, ad group, ad, and day-based return metrics from real data, so that I can evaluate acquisition quality and payback without relying on placeholder fields or ambiguous formulas.

## Scope

This is Phase 2 for the BI Overview feedback. Phase 1 is the frontend-only OpenSpec change under `fe/openspec/changes/002-bi-overview-user-feedback`.

Phase 2 owns the heavier data work:

- Product source of truth.
- D0 LTV, D0 ROAS, D7 ROAS source and storage.
- Purchase, CPA, CVR business definitions.
- Backend filters and response fields needed by Overview.
- Data pipeline propagation through collection, normalization, projection, reporting, and frontend API types.

## Acceptance Criteria

- Product dimension has a confirmed source of truth and is available to BI filters and summary rows.
- `media_source` is exposed to the frontend as the business-facing name while preserving or explicitly migrating existing backend `platform` semantics.
- D0 LTV, D0 ROAS, and D7 ROAS are backed by real fields or documented as unavailable with no fake substitutions.
- Purchase, CPA, and CVR formulas are confirmed and implemented from summed numerator/denominator inputs, not averaged row ratios.
- `/api/bi/ua-report` or a replacement BI endpoint returns the fields needed by the Phase 1 Overview UI without breaking existing pages.
- MySQL and ClickHouse schema changes are migrated idempotently.
- Collector/normalizer/projector changes preserve current Google Ads, Facebook, and TikTok behavior unless a platform is explicitly out of scope.
- Frontend API types and metric helpers are updated to consume the new fields.
- Verification covers backend tests, frontend build, and at least one local API/UI smoke check.

## Data Requirements

| Data | Requirement |
| --- | --- |
| product | Must have a source of truth before implementation. Candidate sources: deployment config, account metadata, game-side KPI dimension, or external product mapping table. |
| media_source | Should map to the existing ad source concept currently named `platform`, unless a separate migration is approved. |
| region/country | Must define whether ad-side rows, game-side rows, or merged UA rows own the dimension. |
| device/platform | Must disambiguate device, OS, and product/client platform before adding a new field. |
| D0 LTV | Needs real D0 revenue or LTV input. Do not derive from D1/D7/total revenue unless the business explicitly defines that formula. |
| D0 ROAS | Needs D0 revenue basis and spend basis at the same grain. |
| D7 ROAS | Needs D7 revenue basis and spend basis at the same grain. |
| purchase | Must choose purchaser users, purchase count, or purchase revenue. |
| CPA | Must choose cost per purchaser, cost per purchase event, or ad-platform cost per conversion. |
| CVR | Must choose install CVR (`installs / clicks`) or ad conversion CVR (`conversions / clicks`). |

## Non-Goals

- Reworking the entire dashboard visual design.
- Removing existing BI endpoints before compatibility is planned.
- Renaming every backend `platform` field just for UI terminology.
- Filling missing D0 fields with approximate or misleading values.
- Replacing existing worker lifecycle or local stack scripts.
