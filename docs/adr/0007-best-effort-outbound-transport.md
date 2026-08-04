# ADR 0007: Best-effort outbound transport before the durable sync engine

[简体中文](0007-best-effort-outbound-transport.zh-CN.md)

## Status

Accepted for Phase 2 Sprint 2.5 and intended as a bridge to Phase 3.

## Decision

Use a small standard-library HTTP transport client for authenticated peer capabilities and
clipboard delivery. The device module subscribes to `clipboard.changed`, forwards only local
events to paired peers advertising `clipboard.text.push`, and never forwards a remote event.

The delivery is best-effort with bounded response/request sizes and timeouts. It does not claim
durability, retries, ordering, receipts or offline replay. Those guarantees belong to the
future sync engine and job store.

## Rationale

This provides a real multi-host path and exercises peer tokens and capability negotiation without
inventing a second queue implementation inside a feature module. The explicit source check is
the minimum feedback-loop guard for two LocalBridge hosts.

## Consequences

- LocalBridge-to-LocalBridge clipboard delivery can work when both peers are reachable and paired.
- A failed request is observable in logs but is not replayed automatically.
- iPhone Shortcuts and other pull-only clients are unaffected.
- Phase 3 must replace direct event forwarding with a durable, idempotent sync/job pipeline.
