# ADR 0009: Persisted Peer-Token Lifecycle

[简体中文](0009-peer-token-lifecycle.zh-CN.md)

## Status

Accepted for Phase 2 Sprint 2.6.

## Context

Pairing generated a random peer token, but the version 1 registry serialized the public `Peer`
shape, whose token field was intentionally excluded from JSON. A restart therefore silently lost
the credential. Tokens also had no issue time, expiry or bounded rotation overlap.

## Decision

- Registry version 2 uses a private persistence model and stores current and temporary previous
  credentials separately from every public API representation.
- A token is valid for `security.peer_token_ttl` (default 720 hours). Rotation preserves a still
  valid previous token for at most `security.token_overlap_ttl` (default 10 minutes), capped by
  the old token's own expiry.
- Pairing the same device rotates its token. `POST /api/v1/devices/{id}/token/rotate` provides an
  explicit rotation operation and returns the new token exactly once.
- Remote rotation is limited to the management token or the target peer's own current/overlap
  token. Loopback management is allowed. A token belonging to another peer is insufficient.
- Public list/get responses expose issue and expiry timestamps but never credential values.
- Version 1 registries migrate automatically. Because their tokens were never persisted, affected
  peers become `repair_required` and must be re-paired or rotated by local management.
- Registry replacement uses a flushed temporary file and restrictive file mode before replacement.

## Consequences

Tokens now survive restart and expire predictably. A short overlap avoids breaking in-flight calls
during rotation without accepting an old credential indefinitely. The registry is security-sensitive:
tokens are currently stored as plaintext JSON and Windows file mode is not an ACL guarantee. The data
directory must remain private to the user. OS credential vault integration and authenticated transport
remain later hardening work.

