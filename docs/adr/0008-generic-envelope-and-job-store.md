# ADR 0008: Generic envelope and durable job state

[简体中文](0008-generic-envelope-and-job-store.zh-CN.md)

## Status

Accepted for Phase 3 Sprint 3.1.

## Decision

Represent cross-device work as an `Envelope` and track each targeted delivery as a persistent
Job. Jobs have explicit states, bounded payload/storage size, retention and transition rules.
The current implementation uses a versioned JSON store behind a Go API; feature modules do not
depend on the file format.

The best-effort clipboard forwarder creates and updates jobs. A delivered job ID is idempotent;
failed jobs are visible for future retry work but are not retried by this Sprint.

## Rationale

Direct event-to-HTTP forwarding cannot explain failures or recover after restart. A durable job
boundary lets future files, URLs, notifications and images share delivery semantics while each
module keeps its own payload validation.

## Consequences

- Job payloads may contain private clipboard data and require retention/access controls.
- The JSON store is adequate for the current bounded workload but is not a high-concurrency
  database; the storage interface can move to SQLite later.
- Phase 3 must add idempotent retry, offline replay, receipts, ordering and conflict policy
  before claiming reliable synchronization.
