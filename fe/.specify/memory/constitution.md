# Constitution

## Principles

### 1. Contract First

Frontend work must identify the backend endpoints it depends on before implementation. If an endpoint shape changes, create a backend-facing spec before coding.

### 2. Small Slices

Each frontend change should fit one feature folder under `specs/<number>-<name>/` and should finish with build verification.

### 3. Production Route Stability

`/bi` and `/bi/*` remain stable production routes. Vite dev routes can differ only during local development.

### 4. Clean Repository Shape

Repository root stays clean with `be/` and `fe/`. SDD artifacts for frontend changes live under `fe/`.

### 5. Verification Required

Every frontend implementation should run `npm run build`. Changes touching backend serving should also verify from `be/`.

