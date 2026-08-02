# Contributing to LocalBridge

## Workflow

1. Create a focused branch from `main`.
2. Write or update the design note when a public contract changes.
3. Keep a commit focused on one concern and preferably below 300 changed lines.
4. Run `go test ./...`, `go vet ./...`, and the platform build before opening a PR.
5. Update user-facing documentation and `CHANGELOG.md` with every feature change.

Use Conventional Commit prefixes such as `feat:`, `fix:`, `docs:`, `test:`, and `chore:`.

## Scope

The project is LAN-first. Do not add cloud services, telemetry, or a new dependency unless
the trade-off is documented and reviewed. New capabilities should be modules connected to
the stable core through interfaces and events.
