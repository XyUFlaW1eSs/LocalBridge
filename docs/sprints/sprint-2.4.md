# Sprint 2.4 — Paired peer-token authentication

[简体中文](sprint-2.4.zh-CN.md)

## Goal

Make the token issued by explicit pairing usable by a paired peer without exposing registry
secrets or weakening the management-token boundary.

## Scope

- Constant-time peer-token validation from the device registry.
- HTTP authentication accepts either the configured management token or a paired peer token
  when `security.auth_enabled` is enabled.
- Capability and protocol documentation for the two token roles.

## Non-goals

- Automatic provisioning, token expiry/rotation policy or encrypted registry storage.
- Outbound sync, peer health polling or a full pairing UI.

## Acceptance criteria

- A valid management token and a valid paired peer token can access protected APIs.
- Invalid and revoked peer tokens are rejected with `401`.
- Peer-token comparison is constant-time and tokens are absent from logs/list responses.
- Authentication remains disabled by default for v0.1 compatibility.
- Tests and docs explain that peer tokens are effective only when authentication is enabled.

## Verification

```text
go test ./...
go vet ./...
go build ./cmd/localbridge
git diff --check
```

## Follow-up

Use this boundary in the outbound transport and peer health client. Replace manual management
and pairing-code configuration with a user-facing pairing flow before v1.0.
