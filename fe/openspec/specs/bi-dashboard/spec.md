# BI Dashboard Spec

## Capability

The frontend provides a BI dashboard for account snapshots, campaign performance, UA reporting, creative quality, and local control operations.

## Requirements

### Requirement: dashboard routes

The app shall support these production routes:

- `/bi`
- `/bi/overview`
- `/bi/breakdown`
- `/bi/creatives`
- `/bi/quality`
- `/bi/control`

`/bi` shall redirect or resolve to the overview experience.

### Requirement: backend contract

The app shall read BI data from existing backend endpoints without changing their response shape:

- `/api/bi/snapshots`
- `/api/bi/campaigns`
- `/api/bi/insights/summary`
- `/api/bi/insights/detail`
- `/api/bi/campaign-diagnostics`
- `/api/bi/search-terms`
- `/api/bi/ua-report`
- `/api/bi/game-kpis`
- `/api/bi/ua-fields`

### Requirement: local control

The control page shall use `/api/control/*` endpoints for local stack actions and shall surface command output clearly.

### Requirement: build artifact

The production frontend shall build into `fe/dist`, and the backend shall serve it from `../fe/dist` when running from `be/`.
