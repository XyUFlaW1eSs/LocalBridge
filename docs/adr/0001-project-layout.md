# ADR 0001: Modular internal layout

## Decision

Use `cmd/`, `internal/app`, `internal/server`, `internal/eventbus`, `internal/module` and
`internal/modules/<feature>`. Keep platform-specific adapters inside their feature package
behind a small interface.

## Rationale

This keeps the executable composition explicit, prevents feature packages from becoming a
second framework, and makes future file/image/device modules independent of the core.
