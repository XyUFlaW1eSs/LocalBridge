# Sprint 1 — Clipboard MVP

[简体中文](sprint-1.zh-CN.md)

## Goal

Deliver a usable Windows ↔ iPhone text clipboard exchange over a trusted LAN.

## Deliverables

- Clipboard module with `Platform` interface and Windows Win32 adapter.
- HTTP Push, Pull and status endpoints.
- SHA-256 deduplication and remote-write loop suppression.
- iPhone Shortcut setup guide and example payloads.
- Tests/build checks and deployment runbook.

## Acceptance criteria

1. `go test ./...` and `go vet ./...` pass.
2. The Windows binary starts with the example configuration.
3. Health endpoint returns `status=ok`.
4. POSTing text returns an item and makes it available from latest.
5. Repeating the same content returns `accepted=false`.
6. Shortcut documentation is sufficient to configure Push and Pull without undocumented steps.

## Explicitly out of scope

Images, history, automatic iPhone background polling, authentication, TLS, discovery and
Windows service installation. They are tracked in `ROADMAP.md`.
