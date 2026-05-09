# Implementation Plan

## Script Location

The new startup entry lives under backend ops scripts:

```text
be/scripts/ops/mac_local_start.sh
```

## Make Target

Expose the script through:

```bash
make mac-start
```

## Dependency Policy

The script installs missing Homebrew-managed tools by default, but it does not install Homebrew itself. If Homebrew is absent, it gives a clear error and points to `https://brew.sh`.

## Runtime Policy

The script starts the same stack as `make up`; it does not introduce a separate runtime path.

## Verification

```bash
bash -n scripts/ops/mac_local_start.sh
go test ./...
```

