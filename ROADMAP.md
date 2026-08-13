# Roadmap

## Next

- Separate routing, handlers, and storage as the API grows.
- Complete compatible update and delete operations with tests and API docs.
- Add continuous integration after the local verification baseline is stable.
- Evaluate a storage abstraction before introducing persistence.

## Completed

- Route-level test coverage for the existing list, lookup, and create behavior.
- Create requests reject malformed or incomplete albums with documented client-error responses.
- Shared in-memory album access is synchronized for concurrent requests.
