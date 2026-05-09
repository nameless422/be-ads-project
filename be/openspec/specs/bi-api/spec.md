# BI API Spec

## Capability

`cmd/bi-api` serves BI query APIs, control APIs, health checks, and the production React BI frontend.

## Requirements

### Requirement: health endpoint

The backend shall expose `GET /healthz` for cheap liveness verification.

### Requirement: BI query endpoints

The backend shall expose these BI query endpoints:

- `GET /api/bi/snapshots`
- `GET /api/bi/campaigns`
- `GET /api/bi/insights/summary`
- `GET /api/bi/insights/detail`
- `GET /api/bi/campaign-diagnostics`
- `GET /api/bi/search-terms`
- `GET /api/bi/ua-report`
- `GET /api/bi/ua-fields`
- `GET /api/bi/game-kpis`
- `POST /api/bi/game-kpis`

### Requirement: local control endpoints

The backend shall expose `/api/control/*` endpoints for local operational control and dead-letter visibility.

These endpoints shall execute backend-owned operations only; frontend code shall not run local shell scripts directly.

### Requirement: frontend serving

When running from `be/`, `cmd/bi-api` shall serve the React production build from `../fe/dist` under:

- `/bi`
- `/bi/*`

If the build artifact is missing, the error shall tell the developer how to build `../fe`.

