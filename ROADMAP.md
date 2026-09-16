# Roadmap

[简体中文](ROADMAP.zh-CN.md)

LocalBridge is evolving from a Windows/iPhone clipboard MVP into a local-first, modular LAN
collaboration platform.

## Delivered

- Phase 0: repository bootstrap and core runtime.
- Phase 1 / Sprint 1: text clipboard exchange between Windows and iPhone Shortcuts.
- `v0.1.0`: runnable Windows package with foreground diagnostics and bilingual documentation.
- Phase 2 Sprints 2.1–2.6: request identity, capabilities, authentication, explicit pairing,
  discovery, peer health/outbound transport and restart-safe expiring peer-token rotation.
- Phase 4.1: persisted multi-file shares, mobile download/upload pages, HTTP Range,
  Content-Range resume and recovery-safe file storage foundation.
- Phase 4.2A–4.2D: embedded file GUI, local QR PNG, controlled browser uploads, per-user Windows
  startup/Explorer synchronization, native tray, completion events, receive approval and WebView2 host.

## Planned releases

- **Phase 2 / `v0.2.x` — Platform hardening:** Sprints 2.1–2.6 delivered request identity through
  peer-token lifecycle; remaining work includes protected credential storage, user provisioning,
  secure transport, diagnostics, shared retry/timeout policy and configuration migration.
- **Phase 3 / `v0.3.x` — Sync engine and rich clipboard:** generic envelopes, capability
  negotiation, outbound delivery, offline queue, history, images, HTML/RTF and screenshots.
- **Phase 4 / `v0.4.x` — LAN collaboration:** file transfer, the embedded GUI and the Windows shell
  foundation, receive approval and native hosted window are implemented; URL push, image delivery,
  composite context jobs and broader transfer state remain.
- **Phase 5 / `v0.5.x` — Clients and devices:** native iOS/iPadOS, Android, macOS, Linux,
  Windows tray/service, CLI, diagnostics web page and multi-device targeting.
- **Phase 6 / `v0.6.x` — Automation and plugins:** plugin SDK, permissions, hotkeys, browser
  extension, Webhooks, scripting, command palette and safe workflow rules.
- **Phase 7 / `v1.0.0` — Stable platform:** protocol compatibility, security review, signed
  artifacts, upgrade/rollback, recovery, performance budgets and conformance suite.

The detailed capability map, brainstorm backlog, dependencies, priorities and release gates are
in [`docs/roadmap.md`](docs/roadmap.md), [`docs/product-plan.md`](docs/product-plan.md) and
[`docs/file-sharing-product-plan.md`](docs/file-sharing-product-plan.md). The agent execution
workflow is in [`docs/engineering-workflow.md`](docs/engineering-workflow.md).
The development workflow is documented in [`docs/developer-guide.md`](docs/developer-guide.md).
