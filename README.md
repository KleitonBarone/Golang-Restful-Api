# Golang Basic Restful Api

A Go and Gin REST API for creating, listing, updating, and deleting albums.
Albums live only in process memory. Each start loads three sample albums;
created albums and changes to existing albums are lost when the process exits.

## API docs

Once the server is running, visit http://localhost:8080/docs for interactive API documentation powered by Scalar.
When using a different listen address, open `/docs` at that address. Interactive
requests use the same host and scheme as the documentation page.
`HEAD /docs` and `HEAD /swagger/doc.json` return the same status and content type
as GET without a response body for link and uptime checks of the documentation
page and its OpenAPI document.

`/swagger/doc.json` serves the checked-in generated JSON embedded in the binary.
Scalar is the documentation UI; the service does not bundle or serve Swagger UI
pages or assets under `/swagger/`.

The documentation page loads Scalar 1.76.0 from jsDelivr. To update it, review
the new release and change the exact version in `docs/scalar.html`. Keep the
version pinned so CDN updates cannot change the UI independently of this repo.

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
different address, such as `:9090`. After binding, the startup log reports
`Listening on <address>`. Use `127.0.0.1:0` to let the OS assign a free local
port, then use the address from that log to reach the API and `/docs`.

To try the complete API, follow the [PowerShell album walkthrough](docs/album-workflow.md).
It creates its own album, reads and updates it, inspects pagination metadata,
and deletes it without changing the sample albums.

Request logs use the connection peer's IP address. `X-Forwarded-For` and
`X-Real-IP` headers are ignored because the service has no trusted-proxy
configuration. Behind a reverse proxy, logs therefore identify the proxy.

Use `GET /health` for local and automated readiness checks. A healthy process
returns `200 OK` with `{"status":"ok"}`. `HEAD /health` returns the same status
and content type without a response body.
Trailing-slash redirects for `/health/` and `/docs/` ignore untrusted
`X-Forwarded-Prefix` headers and preserve query strings. GET requests retain
301 redirects, and HEAD requests use 307 redirects.

`GET /albums` accepts `limit` and `offset` query parameters. Paginated
responses include `X-Total-Count` with the collection size. Each pagination
parameter can appear only once; repeated values return `400 Bad Request`.
Malformed query strings, including invalid percent escapes and unescaped
semicolon separators, also return `400 Bad Request`.
An empty collection returns `[]`, including after deleting the last album.
Paginated requests for an empty collection include `X-Total-Count: 0`.

`HEAD /albums` and `HEAD /albums/{id}` return the same status and content type
as GET without a response body, including validation errors and missing albums.
Paginated HEAD requests validate `limit` and `offset` and include `X-Total-Count`
on success. Trailing-slash HEAD requests redirect with 307, preserving escaped
IDs and query strings and ignoring `X-Forwarded-Prefix`.

`POST /albums`, `PUT /albums/{id}`, and `PATCH /albums/{id}` require
`Content-Type: application/json`. Media type parameters such as `charset=utf-8`
are accepted; a missing, malformed, or different media type returns
`415 Unsupported Media Type`.
The complete body is limited to 64 KiB, including trailing whitespace or data.
Larger bodies with an accepted media type return `413 Request Entity Too Large`.
Send exactly one JSON object containing only recognized fields. Malformed JSON,
unknown fields such as `genre`, and additional JSON values return
`400 Bad Request`. Rejected requests leave stored albums unchanged.

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

Creation and replacement reject album IDs `.` and `..` with `400 Bad Request`
because URL resolution would treat their canonical URLs as the collection or
its parent. Other IDs containing dots, such as `.hidden`, `...`, or `album.v1`,
remain accepted.

On an interrupt or termination signal, the server stops accepting new
connections and gives in-flight requests up to five seconds to finish.
The server allows at most ten seconds to read each complete request and ten
seconds to write its response. Request headers are limited to 16 KiB.

## Verification

CI runs the complete verification on Ubuntu and Windows Server 2022. The Windows
runner supplies MinGW-w64 for the race detector; both jobs use the script below.

Run the same checks as CI before editing and against the final commit:

```text
pwsh -NoLogo -NoProfile -File ./scripts/verify.ps1
```

This checks Go module metadata and formatting, runs race tests, vet, build, and
checks Swagger output without rewriting files. If module metadata is stale, run
`go mod tidy` and commit `go.mod` and `go.sum`. If formatting fails, run `gofmt -w`
on the reported paths and verify again.
Windows requires a compatible MinGW-w64 compiler on PATH or
under `~/.codex/tools/mingw64`; the script enables CGO in its own process.
PowerShell 7 (`pwsh`) must be on PATH for Go tests. The suite runs the commands
from the album walkthrough against a temporary local HTTP server, checking
creation, lookup, replacement, partial updates, pagination, and deletion while
confirming that sample albums remain unchanged.
