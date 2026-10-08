# Roadmap

## Next

- Reject explicit null patch values for Unicode aliases of mutable field names.
  The patch null check currently lowercases names while the decoder case-folds
  them, allowing `artiſt: null` to behave like an omitted artist.
- Copy only the requested page of albums while obtaining its total count from
  the same storage snapshot. Paginated requests currently copy the complete
  collection before slicing it in the handler.

## Completed

- Album mutations reject duplicate JSON fields under the same Unicode case
  folding used by the decoder, including `artist` and `artiſt`, without changing
  stored albums.

- Generated OpenAPI schemas document required album fields, non-blank text,
  and strictly positive prices. Partial updates keep fields optional while
  documenting the constraints on supplied values.

- Album-list requests reject malformed query strings, including invalid percent
  escapes and unescaped semicolon separators, so parsing errors cannot silently
  drop pagination limits.

- Empty album collections return a JSON array for both full and paginated lists,
  including after deleting the last album and when an alternate store returns a
  nil slice. Paginated empty collections retain a total count of zero.

- Album lookup not-found responses use the shared error schema in both the
  handler and generated OpenAPI specification.

- Local verification and CI use one command for race tests, vet, build, and
  non-mutating Swagger freshness checks.

- Album request decoding no longer changes Gin's process-wide JSON decoder
  setting when a router is constructed.
- Album mutations reject invalid UTF-8 bytes instead of silently storing
  replacement characters in text fields.
- Album mutation requests reject duplicate JSON field names so a single
  payload cannot assign conflicting values to the same field.
- Empty partial album updates return a client error instead of reporting a
  successful update with no changes.
- Paginated album requests reject repeated `limit` or `offset` parameters
  instead of silently using the first value.
- Partial album updates reject explicit `null` values instead of silently
  treating them as omitted fields.
- Partial album updates let clients change selected mutable fields without
  resending the complete album.
- Paginated `GET /albums` responses include the collection's total count so
  clients can determine whether another page exists.
- Album IDs containing escaped path separators round-trip through the
  canonical URLs returned by mutation endpoints.
- Successful album creation includes the canonical album URL in the `Location`
  response header.
- Unmatched routes and unsupported HTTP methods return the API's JSON error
  shape instead of Gin's plain-text fallback responses.
- The server force-closes remaining HTTP connections when graceful shutdown
  reaches its deadline, then waits for the serving goroutine to stop before
  returning the shutdown error.
- `GET /health` appears in the generated OpenAPI specification so the
  interactive documentation matches the public routes and README.
- CI runs the test suite with the race detector so synchronized storage and
  concurrent route behavior stay checked on every change.
- Album mutations return the documented 413 response whenever the complete
  request exceeds the 64 KiB body limit, including excess trailing data after a
  valid JSON value.
- Request headers are capped at a documented 16 KiB service limit instead of
  relying on the larger `net/http` default.
- HTTP response writes have a ten-second deadline so slow readers cannot hold
  server resources after handlers finish their bounded work.
- Complete HTTP request reads have a ten-second deadline so clients cannot
  trickle album mutation bodies indefinitely.
- Album mutation routes reject non-JSON media types with a documented 415
  response.
- `GET /albums` accepts validated `limit` and `offset` pagination while requests
  without pagination parameters retain the full-list response.
- HTTP request-header reads and idle connections have explicit timeouts so slow
  clients cannot hold server resources indefinitely.
- Album mutation routes reject request bodies containing more than one JSON
  value instead of accepting the first value and ignoring the rest.
- A lightweight `GET /health` endpoint returns a stable response for local and automated readiness checks.
- The HTTP server handles interrupt and termination signals by allowing in-flight requests up to five seconds to finish before shutdown.
- JSON mutation request bodies are capped at 64 KiB and oversized payloads receive a 413 response without changing stored albums.
- Create and update requests reject unknown JSON fields instead of silently accepting misspelled input.
- CI regenerates Swagger files from handler annotations and rejects documentation drift.
- Route-level test coverage for the existing list, lookup, and create behavior.
- Create requests reject malformed or incomplete albums with documented client-error responses.
- Shared in-memory album access is synchronized for concurrent requests.
- Routing, HTTP handlers, and synchronized in-memory storage are separated for independent testing and composition.
- Compatible update and delete operations include route-level tests and API documentation.
- Continuous integration runs tests, static analysis, and builds for pushes and pull requests.
- HTTP handlers depend on a storage abstraction with a synchronized in-memory implementation, allowing persistence to be added without changing route behavior.
- Album creation rejects duplicate IDs atomically, including concurrent requests, so lookup and mutation routes retain one record per ID.
- The server listen address is configurable through `LISTEN_ADDRESS`, and startup failures terminate the process with an error.
