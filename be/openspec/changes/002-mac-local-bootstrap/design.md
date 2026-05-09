# Design: Mac Local Bootstrap

## Entry

```bash
cd be
make mac-start
```

`make mac-start` delegates to:

```bash
./scripts/ops/mac_local_start.sh
```

## Checks

The script verifies:

- macOS
- Xcode Command Line Tools
- Homebrew
- Go
- Node.js and npm
- Docker Desktop and Docker daemon readiness
- required local ports

## Startup Flow

```text
dependency checks
  -> npm ci under ../fe
  -> make up
  -> wait for GET /healthz
  -> make verify
  -> print URLs and next commands
```

## Options

- `--no-install`: fail instead of installing missing brew packages.
- `--skip-verify`: start only.
- `--check-only`: validate local dependencies without starting.
- `--reset`: run `make down` before startup.

