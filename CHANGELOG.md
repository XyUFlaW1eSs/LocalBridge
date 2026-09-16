# Changelog

[简体中文](CHANGELOG.zh-CN.md)

All notable changes to LocalBridge are documented here.

## [Unreleased]

- Added a bilingual security threat model and mandatory v1.0 security release gates.
- Phase 2 Sprint 2.1: added request IDs, capability discovery and optional Bearer-token authentication.
- Added security configuration, deployment guidance and protocol/ADR documentation for the transition boundary.
- Phase 2 Sprint 2.2: added explicit pairing, peer-token generation, a persisted device registry and revoke/list APIs.
- Phase 2 Sprint 2.3: added optional UDP discovery as an untrusted reachability hint.
- Phase 2 Sprint 2.4: paired peer tokens can authenticate protected APIs when auth is enabled.
- Phase 2 Sprint 2.5: added peer health probes and best-effort outbound clipboard delivery for paired LocalBridge peers.
- Phase 2 Sprint 2.6: fixed peer-token persistence across restart, added configurable expiry,
  bounded-overlap rotation, target-scoped rotation authorization and version 1 registry migration.
- Phase 2 Sprint 2.7: added configuration schema v1, non-destructive legacy migration, strict
  future/unknown input rejection, management-only redacted effective-configuration diagnostics
  and a non-interactive `-check-config` command.
- Phase 2 Sprint 2.8: added opt-in TLS 1.2+ with HTTPS-only startup, lowercase leaf-certificate
  SHA-256 advertisement, v3 secure/legacy peer metadata and v2 registry migration, exact
  certificate-pinned HTTPS peer transport, redirect rejection and scheme-aware GUI/Explorer URLs.
  First-use fingerprint UX, automatic certificate rotation and iPhone trust installation remain future work.
- Added `-support-bundle` for non-overwriting, owner-readable ZIP reports containing only redacted
  configuration, runtime information and state-file metadata; credentials, identity, paths, TLS
  material, payloads and persisted state bodies are excluded.
- Phase 3 Sprint 3.1: added generic Envelopes, durable sync Jobs, retention bounds and read-only job inspection.
- Phase 4 Sprint 4.2A: added the embedded `/app/` GUI, browser-safe multi-file shares, persisted
  receive records UI, real local QR PNG generation and versioned settings persistence.
- Phase 4 Sprint 4.2B foundation: added owned-share cleanup, per-user Windows startup and Explorer
  context-menu synchronization, a native tray, multi-file `-share` handling, metadata-only transfer
  events and completion notifications. Native hosted-window close-to-tray remains pending.
- Phase 4 Sprint 4.2C: added persisted pending/active/rejected upload states, loopback/authenticated
  approve/reject endpoints, Windows receive-request notifications and a mobile wait-and-resume flow.
- Phase 4 Sprint 4.2D: added a pure-Go WebView2 Windows host, real close-to-tray behavior, tray restore,
  browser fallback and an interactive window/health/close/restore smoke test.

## [0.1.0] - 2026-08-02

- Added the Windows/iPhone clipboard MVP over LAN HTTP.
- Added a modular Go application core, configuration, logging, EventBus and health API.
- Added development documentation and Shortcut setup instructions.
