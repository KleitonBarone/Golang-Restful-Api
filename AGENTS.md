# Repository Instructions

## Purpose

Maintain and incrementally improve the Albums REST API while preserving current
external behavior unless `ROADMAP.md` explicitly calls for a compatible feature.

## Toolchain

- Use the Go version declared in `go.mod` or a compatible newer Go release.
- Keep generated API documentation synchronized when Swagger annotations change.
- Do not add a dependency unless the change clearly requires it and the standard
  library is insufficient.

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

## Safety and scope

- Prefer small, non-breaking improvements supported by tests.
- Preserve existing response shapes and status codes unless the roadmap clearly
  authorizes an API change.
- Avoid authentication, secrets, deployment credentials, destructive storage
  migrations, and major dependency upgrades without explicit approval.
- Never use empty commits, force-push, rewrite published history, or discard a
  dirty working tree.
- Update `ROADMAP.md` only when implementation or repository inspection produces
  substantive new information.

## Completion criteria

A change is complete when its behavior is covered by tests, all verification
commands pass, API documentation remains accurate, and the commits form a small,
coherent unit of work.
