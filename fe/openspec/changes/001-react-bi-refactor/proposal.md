# Change: React BI Refactor

## Why

The original BI page was embedded in the Go server and had grown hard to iterate. A React frontend gives clearer ownership, reusable UI components, and a better base for future dashboard work.

## What Changes

- Move the BI experience into a React + Vite project under `fe/`.
- Keep the Go backend as the API and production static file server.
- Keep existing `/api/bi/*` and `/api/control/*` contracts.
- Keep `/bi` and `/bi/*` as the user-facing production routes.
- Keep SDD artifacts inside `fe/` so the repository root stays clean.

## Out Of Scope

- Backend data model redesign.
- New BI API response contracts.
- Authentication and permission work.
- Full design system extraction.
