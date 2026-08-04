# ADR 0006: LAN discovery is a reachability hint, not trust

[简体中文](0006-discovery-is-not-trust.zh-CN.md)

## Status

Accepted for Phase 2 Sprint 2.3.

## Decision

Use bounded UDP/IPv4 announcements for optional LAN discovery. An announcement contains a
protocol version, device ID/name, API port, capabilities and a nonce. The receiver derives the
peer address from the UDP source address and stores the result only in an ephemeral discovered
list.

Discovered peers do not enter the paired registry, receive credentials or gain API access.
Explicit pairing remains the authorization step. Manual address entry remains supported when
broadcast is blocked.

## Rationale

Discovery improves setup and diagnostics but is unauthenticated and easy to spoof. Treating it
as authorization would allow any LAN participant to appear trusted. Keeping discovery ephemeral
also avoids polluting durable state with stale or malicious announcements.

## Consequences

- Discovery is disabled by default and can be turned off for networks that do not permit broadcast.
- The protocol must tolerate unknown capabilities and invalid packets.
- Future discovery may use mDNS or authenticated responses, but it must preserve the trust boundary.
