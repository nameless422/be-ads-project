# Feature Spec: Mac Local Bootstrap

## User Story

As a new developer on macOS, I want one command to prepare and start the local project so that I can open the BI page without rediscovering all setup steps.

## Acceptance Criteria

- `cd be && make mac-start` exists.
- The script checks macOS-only assumptions.
- The script checks Xcode Command Line Tools, Homebrew, Go, Node.js, npm, Docker, and required ports.
- The script can install missing Homebrew formulae/casks when `INSTALL_MISSING_TOOLS=1`.
- The script installs frontend dependencies from `fe/package-lock.json`.
- The script starts the backend local stack and waits for `GET /healthz`.
- The script runs `make verify` unless `--skip-verify` is passed.
- The script prints local URLs and next commands after success.

## Non-Goals

- Installing Homebrew itself.
- Supporting non-macOS environments.
- Hiding Docker Desktop permission prompts.

