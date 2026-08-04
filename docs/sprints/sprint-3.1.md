# Sprint 3.1 — Generic envelope and durable job state

[简体中文](sprint-3.1.zh-CN.md)

## Goal

Give cross-device delivery an explainable, persistent state boundary before adding more payload
modules.

## Scope

- Versioned `Envelope` and Job model.
- Bounded JSON job store with retention and transition validation.
- Read-only job inspection endpoints.
- Integration with best-effort outbound clipboard delivery.

## Non-goals

- Automatic retries, offline replay, ordering, receipts or conflict resolution.
- File/image/URL/notification payload modules.
- Encrypted payload storage or SQLite migration.

## Acceptance criteria

- A local clipboard delivery creates a pending job and records delivering/delivered or failed.
- Job state survives a restart and invalid transitions are rejected.
- Job payload, file size, count and retention limits are enforced.
- Job inspection is available without allowing clients to forge delivery state.
- Existing clipboard API and peer feedback-loop behavior remain unchanged.

## Verification

```text
go test ./...
go vet ./...
go build ./cmd/localbridge
git diff --check
```

## Follow-up

Sprint 3.2 should add retry/offline policy, receipts and conflict/idempotency tests before
implementing rich clipboard formats.
