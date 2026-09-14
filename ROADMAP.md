# Roadmap

## Next

1. Run the test suite with the race detector in CI so synchronized storage and
   concurrent route behavior stay checked on every change.
2. Include `GET /health` in the generated OpenAPI specification so the
   interactive documentation matches the public routes and README.

## Completed

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
