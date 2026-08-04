# Sprint 2.2 — Explicit pairing and device registry

[简体中文](sprint-2.2.zh-CN.md)

## Goal

Introduce an explicit trust record without treating LAN discovery as authorization.

## Scope

- Persisted local peer registry with a versioned JSON format.
- Pair, list, inspect and revoke device endpoints.
- Pairing-code validation, peer-token generation and token redaction from responses/logs.
- Device module registration and capability metadata.

## Non-goals

- Automatic discovery, outbound peer delivery or peer-token authentication.
- Encrypted registry storage or automatic token rotation policy.
- Native pairing UI on iPhone or other clients.

## Acceptance criteria

- The application starts with no registry file and creates it only when pairing succeeds.
- Invalid pairing codes and invalid device identities are rejected.
- Pairing creates a peer token, but list/get responses and logs do not expose it.
- Re-pairing rotates the token; deleting a peer removes its trusted record.
- Registry writes are bounded, versioned and protected with restrictive file permissions.
- Existing clipboard behavior and v0.1 configuration remain compatible.

## Verification

```text
go test ./...
go vet ./...
go build ./cmd/localbridge
git diff --check
```

## Follow-up

Sprint 2.3 should add discovery as a reachability hint, explicit device health/last-seen
updates and the first outbound authenticated transport using this registry.
