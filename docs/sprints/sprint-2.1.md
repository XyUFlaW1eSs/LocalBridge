# Sprint 2.1 — Protocol and security foundation

[简体中文](sprint-2.1.zh-CN.md)

## Goal

Make Phase 2 traffic diagnosable and provide a backward-compatible authentication boundary
before device pairing and discovery are implemented.

## Scope

- Bounded request IDs with response propagation and structured request logs.
- `GET /api/v1/system/capabilities` for client capability negotiation groundwork.
- Optional Bearer token protection for non-health API endpoints.
- Security configuration and deployment guidance.

## Non-goals

- Automatic pairing, token rotation or device revocation.
- LAN discovery or peer-to-peer outbound delivery.
- TLS termination or a persistent device registry.

## Acceptance criteria

- Existing v0.1 configuration without a `security` section still starts.
- A configured token protects capabilities and feature endpoints while health remains available.
- Missing and invalid tokens return `401` with a request ID; valid tokens succeed.
- Clients can inspect version, device identity and capabilities before using a feature.
- Tokens are not logged and the implementation uses constant-time comparison.
- Unit tests, `go vet`, documentation and deployment examples pass review.

## Verification

```text
go test ./...
go vet ./...
git diff --check
```

## Follow-up

Sprint 2.2 should add device identity persistence, pairing, token provisioning/revocation and
the device registry. The direct YAML token is intentionally temporary.
