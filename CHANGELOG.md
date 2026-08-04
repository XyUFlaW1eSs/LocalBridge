# Changelog

[简体中文](CHANGELOG.zh-CN.md)

All notable changes to LocalBridge are documented here.

## [Unreleased]

- Phase 2 Sprint 2.1: added request IDs, capability discovery and optional Bearer-token authentication.
- Added security configuration, deployment guidance and protocol/ADR documentation for the transition boundary.
- Phase 2 Sprint 2.2: added explicit pairing, peer-token generation, a persisted device registry and revoke/list APIs.
- Phase 2 Sprint 2.3: added optional UDP discovery as an untrusted reachability hint.
- Phase 2 Sprint 2.4: paired peer tokens can authenticate protected APIs when auth is enabled.
- Phase 2 Sprint 2.5: added peer health probes and best-effort outbound clipboard delivery for paired LocalBridge peers.
- Phase 3 Sprint 3.1: added generic Envelopes, durable sync Jobs, retention bounds and read-only job inspection.

## [0.1.0] - 2026-08-02

- Added the Windows/iPhone clipboard MVP over LAN HTTP.
- Added a modular Go application core, configuration, logging, EventBus and health API.
- Added development documentation and Shortcut setup instructions.
