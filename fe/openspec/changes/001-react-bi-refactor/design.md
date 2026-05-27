# Design: React BI Refactor

## Structure

```text
fe/
  src/
    api/
    components/
    pages/
    utils/
  openspec/
  specs/
  .specify/
```

## Runtime

Development uses Vite from `fe/`. Production uses `be/cmd/bi-api`, which serves `fe/dist` under `/bi`.

## Data Flow

```text
React pages -> src/api/client.ts -> /api/bi/* and /api/control/* -> Go backend
```

## UI Boundaries

- `pages/` owns route-level composition.
- `components/` owns reusable display and input controls.
- `api/` owns request and response mapping.
- `utils/` owns pure calculations, routing helpers, and formatting.

## Risk Controls

- No frontend change should require backend schema changes unless a separate backend spec is opened.
- Build output stays ignored and reproducible.
- Local operations stay behind backend `/api/control/*` endpoints.
