# Sprint 2.6 — Peer-Token Lifecycle

[简体中文](sprint-2.6.zh-CN.md)

## Goal

Make paired-device credentials restart-safe, expiring and safely rotatable without exposing them
through normal device APIs.

## Delivered

- Version 2 private device-registry persistence including current and bounded previous credentials.
- Configurable 30-day default token lifetime and 10-minute default rotation overlap.
- Explicit token-rotation endpoint plus same-ID re-pair rotation.
- Target-scoped rotation authorization: loopback, management token or the target peer's own token.
- `token_issued_at`, `token_expires_at`, `token_expired` and `repair_required` public diagnostics.
- Version 1 migration, restart validation, overlap/expiry tests and Windows-safe close-before-replace.
- Atomic-style temporary-file persistence with flush, restrictive mode and rollback of in-memory
  pair/rotate/revoke mutations when persistence fails.

## API and configuration

- `POST /api/v1/devices/{id}/token/rotate` returns `{rotated, device, token}`. The token is shown
  once and must be stored immediately by the caller.
- `security.peer_token_ttl: 720h`
- `security.token_overlap_ttl: 10m`; it must be shorter than the token lifetime.
- Capability advertisement includes `device.token.rotate`.

## Acceptance evidence

- A paired token authenticates after recreating the device module from disk.
- Both old and new tokens work only during the configured overlap; the old token then fails.
- The current token fails after its lifetime and the public status becomes `token_expired`.
- Public responses and logs do not contain peer-token values.
- A token from an unrelated peer cannot rotate the target peer.
- A legacy registry migrates to version 2 and missing credentials become `repair_required`.

## Remaining Phase 2 work

Credential-at-rest protection, user-facing provisioning UX, TLS/authenticated channel design,
rate limits, shared retry policy, diagnostics/support bundles and configuration migrations beyond
the device registry remain open.

