# Implementation Plan

## Architecture

The frontend is a Vite React app. The backend remains the API server and production static file host.

```text
fe/src/pages -> fe/src/api -> be/cmd/bi-api -> be/internal/modules/reporting
```

## Steps

1. Establish `fe/` and `be/` as the only root-level project directories.
2. Keep React routes under `/bi`.
3. Keep backend API calls centralized in `src/api/client.ts`.
4. Keep reusable UI in `src/components`.
5. Build the frontend before starting `bi-api`.
6. Keep OpenSpec and Spec Kit artifacts in `fe/`.

## Verification

```bash
cd fe
npm install
npm run build

cd ../be
go test ./...
make up
make verify
make down
```
