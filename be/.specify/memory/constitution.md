# Constitution

## Principles

### 1. Contract First

Backend work must identify the API, storage, message, or script contract it changes before implementation.

### 2. Module Boundaries Stay Clear

Collection, transformation, reporting, and control-plane responsibilities must stay in their owning modules unless a spec explicitly changes the boundary.

### 3. Local Operations Are Product Surface

`Makefile` and `scripts/` are part of the developer experience. Changes to startup, status, verification, or shutdown behavior must be specified and verified.

### 4. Root Stays Clean

Repository root remains limited to `be/`, `fe/`, and repository metadata. Backend SDD files live inside `be/`.

### 5. Verification Is Required

Every backend implementation should run `go test ./...`. Runtime or integration changes should also run the local stack verification flow.

