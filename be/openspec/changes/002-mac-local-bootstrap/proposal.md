# Change: Mac Local Bootstrap

## Why

A new Mac environment currently requires the developer to know the dependency chain manually: Xcode Command Line Tools, Homebrew, Go, Node.js, Docker Desktop, frontend dependencies, stack startup, and verification.

That is too much context for a first local run.

## What Changes

- Add `scripts/ops/mac_local_start.sh` as the macOS one-command local startup entry.
- Add `make mac-start` as the stable command developers can remember.
- Check required tools and local ports before starting the stack.
- Install missing Homebrew packages when possible.
- Start Docker Desktop if it is installed but not running.
- Install frontend dependencies, run `make up`, wait for `/healthz`, and run `make verify`.
- Document the new entry in `README.md`.

## Out Of Scope

- Supporting Linux or Windows.
- Installing Homebrew itself.
- Changing runtime services, API contracts, or storage schemas.

