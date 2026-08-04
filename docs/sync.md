# Sync Engine Foundation

[简体中文](sync.zh-CN.md)

Phase 3 begins with a generic envelope and durable job state. The current implementation wraps
the existing best-effort clipboard forwarding path; it does not yet provide retries, offline
replay or conflict resolution.

## Envelope

An `Envelope` describes a content/action transfer independently from the feature module:

| Field | Meaning |
|---|---|
| `id` | Stable event/job identity used for idempotency. |
| `kind` | Domain operation, currently `clipboard.push`. |
| `type` | Logical type, currently `text`. |
| `mime_type` | Payload MIME type, normally `text/plain`. |
| `hash` | Content deduplication/integrity hash. |
| `source_device_id` | Device that originated the content. |
| `target_device_id` | Intended peer, when delivery is targeted. |
| `correlation_id` | Source event ID for tracing related work. |
| `size` | Payload byte size. |
| `payload` | JSON payload retained for future replay; treat as private data. |
| `created_at` | UTC creation time. |
| `expires_at` | Optional expiration time. |

Feature modules validate their own payload. The sync layer owns identity, state, retention and
delivery semantics.

## Job state

```text
pending -> delivering -> delivered
                    \-> failed -> delivering
pending/delivering/failed -> canceled or expired
```

Every transition updates `updated_at`. Entering `delivering` increments `attempts`. Invalid
transitions are rejected. A delivered job is idempotently returned rather than recreated when
the same ID is submitted again.

## Current HTTP inspection API

`GET /api/v1/sync/jobs?limit=100` returns recent jobs. The limit is clamped to `1..1000`.

`GET /api/v1/sync/jobs/{id}` returns one job or `404`.

These are inspection endpoints only; clients cannot directly mark a job delivered. Delivery is
owned by the transport/module that created it.

## Storage

The current store is a versioned JSON file at `sync.store_path` (default
`data/sync-jobs.json`). It has a 16 MiB file limit, a configurable maximum job count and
retention period. Files are created with restrictive permissions. Job payloads may contain
clipboard text, so deployments must protect the data directory and configure retention.

SQLite or another transactional store may replace this implementation once history, queues and
concurrent clients require stronger durability. The public boundary is the store behavior, not
the JSON file format.

## Current limitations

- Only clipboard outbound delivery creates jobs.
- Failed jobs are recorded but not retried automatically.
- There is no offline queue replay, delivery receipt, ordering guarantee or conflict policy.
- Payload encryption at rest and user-facing history deletion are not implemented yet.
