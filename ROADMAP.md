# Roadmap

This roadmap is based on the current repository state. Reassess priorities after
each substantive change; avoid commits that only move checkboxes.

## Next

- Add explicit request validation and documented client-error responses.
- Protect shared in-memory state so concurrent requests are race-safe.
- Separate routing, handlers, and storage as the API grows.
- Complete compatible update and delete operations with tests and API docs.
- Add continuous integration after the local verification baseline is stable.
- Evaluate a storage abstraction before introducing persistence.

## Completed

- Document repository-specific agent instructions and verification requirements.
- Establish route-level tests for the existing list, lookup, and create behavior.
