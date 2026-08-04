# ADR 0004: Request identity and transition authentication

[简体中文](0004-request-identity-and-transition-auth.zh-CN.md)

## Status

Accepted for Phase 2 Sprint 2.1.

## Decision

Every HTTP request receives a bounded `X-Request-ID`. Clients may provide one; otherwise the
server generates a random ID. The ID is returned in the response header, included in system
responses and error bodies, and written to request logs.

The server supports an optional Bearer token configured under `security`. Health remains public
for local diagnostics; other API endpoints require the token when `auth_enabled` is true. Token
comparison is constant-time and tokens are never logged.

## Rationale

Request identity makes failures diagnosable across clients, logs and future peer transport.
The token is deliberately a transition mechanism for Phase 2: it closes the current unauthenticated
LAN boundary without prematurely inventing the final pairing UX. A later device registry and
pairing flow will provision, rotate and revoke credentials through a documented mechanism.

## Consequences

- Existing v0.1 configurations remain compatible because authentication defaults to disabled.
- Deployment documentation must warn users not to expose the service publicly and must show how
  to enable a strong token.
- All future clients must preserve request IDs when reporting errors.
- Pairing must replace direct token editing before v1.0.
