# ADR 0005: Explicit pairing and a persistent device registry

[简体中文](0005-explicit-pairing-and-device-registry.zh-CN.md)

## Status

Accepted for Phase 2 Sprint 2.2.

## Decision

Maintain a local registry of explicitly paired peers under `device.registry_path`. A peer is
not trusted merely because it is discovered on the LAN. Pairing requires the configured local
pairing code and records the peer ID, display name, address, port, capabilities, status and
timestamps.

The pairing response returns a generated peer token once. Listing and reading devices never
return peer tokens, and logs never contain them. Pairing an existing ID rotates its token;
deleting the peer revokes the local record.

## Rationale

Discovery is an unauthenticated hint about reachability, not authorization. Separating the two
prevents any LAN device from becoming trusted simply by broadcasting a packet. A small JSON
registry is sufficient for this Sprint and leaves a storage interface open for encrypted
storage, migrations and SQLite when history/queues arrive.

## Consequences

- v0.1 deployments remain compatible; the registry is created only when a peer is paired.
- Pairing code and bearer token are currently configured manually, which is transitional.
- The registry contains a peer token needed by future outbound transport and must be protected
  by filesystem permissions and the host account.
- Peer tokens are now accepted by the HTTP authentication boundary when authentication is
  enabled. Automatic provisioning, token rotation policy, encrypted-at-rest storage and
  discovery remain follow-up work.
