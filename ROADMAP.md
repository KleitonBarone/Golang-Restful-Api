# Roadmap

## Next

- No capabilities are currently scheduled.

## Completed

- Route-level test coverage for the existing list, lookup, and create behavior.
- Create requests reject malformed or incomplete albums with documented client-error responses.
- Shared in-memory album access is synchronized for concurrent requests.
- Routing, HTTP handlers, and synchronized in-memory storage are separated for independent testing and composition.
- Compatible update and delete operations include route-level tests and API documentation.
- Continuous integration runs tests, static analysis, and builds for pushes and pull requests.
- HTTP handlers depend on a storage abstraction with a synchronized in-memory implementation, allowing persistence to be added without changing route behavior.
