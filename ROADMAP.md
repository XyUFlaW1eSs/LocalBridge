# Roadmap

[简体中文](ROADMAP.zh-CN.md)

LocalBridge is evolving from a Windows/iPhone clipboard MVP into a local-first, modular LAN
collaboration platform.

## Delivered

- Phase 0: repository bootstrap and core runtime.
- Phase 1 / Sprint 1: text clipboard exchange between Windows and iPhone Shortcuts.
- `v0.1.0`: runnable Windows package with foreground diagnostics and bilingual documentation.

## Planned releases

- **Phase 2 / `v0.2.x` — Platform hardening:** Sprint 2.1 delivered request IDs, capabilities and
  transition authentication; remaining work includes configuration versioning, pairing,
  authentication provisioning,
  device registry, LAN discovery, diagnostics, shared retry/timeout policy and service/tray design.
- **Phase 3 / `v0.3.x` — Sync engine and rich clipboard:** generic envelopes, capability
  negotiation, outbound delivery, offline queue, history, images, HTML/RTF and screenshots.
- **Phase 4 / `v0.4.x` — LAN collaboration:** file transfer, URL push, image delivery,
  notifications, composite context jobs and resumable transfer state.
- **Phase 5 / `v0.5.x` — Clients and devices:** native iOS/iPadOS, Android, macOS, Linux,
  Windows tray/service, CLI, diagnostics web page and multi-device targeting.
- **Phase 6 / `v0.6.x` — Automation and plugins:** plugin SDK, permissions, hotkeys, browser
  extension, Webhooks, scripting, command palette and safe workflow rules.
- **Phase 7 / `v1.0.0` — Stable platform:** protocol compatibility, security review, signed
  artifacts, upgrade/rollback, recovery, performance budgets and conformance suite.

The detailed capability map, brainstorm backlog, dependencies, priorities and release gates are
in [`docs/roadmap.md`](docs/roadmap.md) and [`docs/product-plan.md`](docs/product-plan.md).
The development workflow is documented in [`docs/developer-guide.md`](docs/developer-guide.md).
