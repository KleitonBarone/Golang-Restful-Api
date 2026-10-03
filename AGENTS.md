# Repository Instructions

## Toolchain

- Use the Go version declared in `go.mod` or a compatible newer Go release.
- Do not add a dependency unless the change clearly requires it and the standard
  library is insufficient.

## Repository structure

- `main.go` starts the service, while `router.go` composes the Gin router and
  documentation endpoints.
- `handlers.go` contains HTTP handlers, `storage.go` defines the `albumStore`
  abstraction and its synchronized in-memory implementation, and `album.go`
  defines the API model and validation.
- `main_test.go` contains route-level tests using `httptest`.
- `docs/` contains generated Swagger output and the embedded Scalar UI.
- Tests should construct the in-memory store explicitly when they need to inspect
  handler state, or provide a small `albumStore` test implementation when they
  need to verify behavior at the storage boundary.

## Verification

Run the same complete verification used by CI before editing and again against
the final commit before updating the default branch:

```powershell
pwsh -NoLogo -NoProfile -File ./scripts/verify.ps1
```

This runs race tests, vet, build, and a non-mutating Swagger freshness check.
Windows needs a compatible MinGW-w64 compiler. The script uses `gcc` on PATH or
the gardener's compiler under `~/.codex/tools/mingw64`. Missing prerequisites
block implementation; do not skip race verification.
When Swagger annotations change, regenerate the checked-in documentation.

## Conventions

- Format Go files with `gofmt`.
- Cover handler behavior with route-level tests.
- Keep existing response shapes and status codes compatible unless a requested
  change explicitly updates the API contract.
