# Golang Basic Restful Api

## API docs

Once the server is running, visit http://localhost:8080/docs for interactive API documentation powered by Scalar.
When using a different listen address, open `/docs` at that address. Interactive
requests use the same host and scheme as the documentation page.

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

Request logs use the connection peer's IP address. `X-Forwarded-For` and
`X-Real-IP` headers are ignored because the service has no trusted-proxy
configuration. Behind a reverse proxy, logs therefore identify the proxy.

Use `GET /health` for local and automated readiness checks. A healthy process
returns `200 OK` with `{"status":"ok"}`. `HEAD /health` returns the same status
and content type without a response body.
Trailing-slash redirects for `/health/` and `/docs/` ignore untrusted
`X-Forwarded-Prefix` headers and preserve query strings. GET requests retain
301 redirects, and HEAD requests to `/health/` retain 307 redirects.

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
Album mutations also reject invalid UTF-8 bytes and unpaired Unicode surrogate
escapes with `400 Bad Request` rather than storing replacement characters.
Valid surrogate pairs and intentionally supplied replacement characters remain
accepted.

Canonical album URLs preserve literal plus signs in IDs, including IDs that also
contain escaped path separators. Lookup, replacement, partial updates, and
deletion use the same path decoding. Trailing-slash redirects on these routes
preserve escaped IDs and query strings, using 301 for GET and 307 for mutations
so clients retain the request method and body.
Collection redirects from `/albums/` to `/albums` also preserve query strings
and use 301 for GET and 307 for POST. They ignore `X-Forwarded-Prefix`, matching
the service's policy of not trusting proxy headers.

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
