# Repository Instructions

## Toolchain

- Use the Go version declared in `go.mod` or a compatible newer Go release.
- Do not add a dependency unless the change clearly requires it and the standard
  library is insufficient.

## Repository structure

- `main.go` starts the service, while `router.go` composes the Gin router and
  documentation endpoints.
- `handlers.go` contains HTTP handlers, `storage.go` owns synchronized in-memory
  album state, and `album.go` defines the API model and validation.
- `main_test.go` contains route-level tests using `httptest`.
- `docs/` contains generated Swagger output and the embedded Scalar UI.
- Tests should construct an `albumStore` explicitly when they need to inspect or
  control handler state.

## Verification

Run all of the following before committing and again before updating the default
branch:

```powershell
go test ./...
go vet ./...
go build ./...
```

When Swagger annotations change, regenerate the checked-in documentation and
verify that only intentional generated changes remain.

## Conventions

- Format Go files with `gofmt`.
- Cover handler behavior with route-level tests.
- Keep existing response shapes and status codes compatible unless a requested
  change explicitly updates the API contract.
