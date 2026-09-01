# Roadmap

## Next

- Shut the HTTP server down gracefully on process signals so in-flight requests can finish within a fixed timeout.
- Add a lightweight health endpoint for local and automated readiness checks.

## Completed

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
