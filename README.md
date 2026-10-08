# Golang Basic Restful Api

## API docs

Once the server is running, visit http://localhost:8080/docs for interactive API documentation powered by Scalar.

### Updating OpenAPI spec

After modifying Swagger annotations in the Go source files, regenerate the spec:

```
go run github.com/swaggo/swag/cmd/swag@v1.16.6 init -g router.go
```

## Getting Started

Run the service:

```
go run .
```

It listens on `localhost:8080` by default. Set `LISTEN_ADDRESS` to bind to a
different address, such as `:9090`.

Use `GET /health` for local and automated readiness checks. A healthy process
returns `200 OK` with `{"status":"ok"}`.

`GET /albums` accepts `limit` and `offset` query parameters. Paginated
responses include `X-Total-Count` with the collection size. Each pagination
parameter can appear only once; repeated values return `400 Bad Request`.
Malformed query strings, including invalid percent escapes and unescaped
semicolon separators, also return `400 Bad Request`.
An empty collection returns `[]`, including after deleting the last album.
Paginated requests for an empty collection include `X-Total-Count: 0`.

`PATCH /albums/{id}` updates any supplied `title`, `artist`, or `price` field
without requiring the complete album. Album IDs remain immutable. Explicit
`null` values for mutable fields return `400 Bad Request` instead of being
treated as omitted fields. Empty patch objects also return `400 Bad Request`.
Album creation, replacement, and partial updates reject repeated JSON field
names, including names that match under Unicode case folding such as `artist`
and `artiſt`, with `400 Bad Request`.
Album mutations also reject invalid UTF-8 bytes with `400 Bad Request` rather
than storing replacement characters.

On an interrupt or termination signal, the server stops accepting new
connections and gives in-flight requests up to five seconds to finish.
The server allows at most ten seconds to read each complete request and ten
seconds to write its response. Request headers are limited to 16 KiB.

## Verification

Run the same checks as CI before editing and against the final commit:

```text
pwsh -NoLogo -NoProfile -File ./scripts/verify.ps1
```

This runs race tests, vet, build, and checks Swagger output without rewriting
tracked files. Windows requires a compatible MinGW-w64 compiler on PATH or
under `~/.codex/tools/mingw64`; the script enables CGO in its own process.
