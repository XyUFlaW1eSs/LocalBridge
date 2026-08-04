# LocalBridge Roadmap

[简体中文](roadmap.zh-CN.md)

This is the detailed delivery plan for the vision described in [Product Plan and Capability
Map](product-plan.md). The root [ROADMAP.md](../ROADMAP.md) is a compact status summary.

## Product direction

LocalBridge is not intended to remain a Windows-to-iPhone clipboard synchronizer. It is a
local-first, modular and pluggable LAN collaboration platform. The durable product boundary
is a trusted device network, a generic content/action transport and independently deployable
modules such as clipboard, files, URLs, images and notifications.

The product should remain useful with one Windows host and one phone, but its abstractions
must also support multiple devices, intermittent connectivity and multiple content types.

## Status and release line

| Release | Phase | Status | Result |
|---|---|---|---|
| `v0.1.0` | Phase 0 + Phase 1 / Sprint 1 | Delivered | Windows text clipboard bridge and iPhone Push/Pull Shortcuts |
| `v0.2.x` | Phase 2 | In progress | Sprint 2.1 request IDs/capabilities/authentication delivered; trust, discovery and production foundation remain |
| `v0.3.x` | Phase 3 | In progress | Sprint 3.1 Envelope/job store delivered; retries, history and rich clipboard remain |
| `v0.4.x` | Phase 4 | Planned | Files, URLs, images, notifications and action delivery |
| `v0.5.x` | Phase 5 | Planned | Additional clients and device adapters |
| `v0.6.x` | Phase 6 | Planned | Automation, tray/service UX and plugin SDK |
| `v1.0.0` | Phase 7 | Planned | Stable protocol, secure defaults and upgradeable releases |

Version numbers are indicative. A phase may contain several Sprints and releases; no phase
is complete until its acceptance criteria and release gate pass.

## Capability map

| Capability area | Long-term responsibility | Planned capabilities | Earliest phase |
|---|---|---|---|
| Core runtime | Keep the process and module lifecycle stable | Config versioning, module registry, EventBus, API versioning, structured logs, graceful shutdown | Phase 2 |
| Device identity and trust | Decide which devices may communicate | Pairing, device registry, rotating tokens, allow-list, TLS or Noise-style secure channel, revoke/reset | Phase 2 |
| Discovery and connectivity | Find and monitor peers on a LAN | mDNS/UDP discovery, manual pairing, network-change handling, health probes, diagnostics | Phase 2 |
| Transport | Move small and large payloads reliably | HTTP JSON, WebSocket/SSE events, chunking, checksums, resumable transfer, compression, retries, rate limits | Phase 2–4 |
| Sync engine | Apply content consistently across devices | Common envelope, capability negotiation, idempotency, deduplication, ordering, conflict policy, offline queue | Phase 3 |
| State and storage | Make history and recovery explicit | Bounded history, SQLite/file store boundary, retention, migrations, backup/export, privacy deletion | Phase 3 |
| Clipboard | Bridge OS clipboard formats | Text, images, HTML/RTF, screenshots, history search, per-device sync policy | Phase 1–3 |
| File and URL collaboration | Move useful work context between devices | File send/receive, folders, URL preview/hand-off, drag/drop, progress, resume, cancellation | Phase 4 |
| Notifications and actions | Trigger attention or workflows | Native notifications, acknowledgement, action buttons, clipboard/file/URL actions, expiry | Phase 4–6 |
| Clients and adapters | Fit each platform's lifecycle | iPhone Shortcuts, native iOS/iPadOS, Android, macOS, Linux, Windows tray/service, CLI and web diagnostics | Phase 5–6 |
| Automation and integrations | Let users compose workflows | Hotkeys, command palette, browser extension, webhooks, scripting/CLI, import/export | Phase 6 |
| Operations and release | Keep installations safe and supportable | Installer/service registration, signed artifacts, upgrade/rollback, metrics, support bundle, compatibility policy | Phase 2 and 7 |

## Phases

### Phase 0 — Core foundation (delivered)

- Repository layout, Go Module and cross-platform build baseline.
- Configuration, structured logging, lifecycle management and HTTP server.
- Module contract and non-blocking EventBus.
- Architecture, protocol, deployment, development and bilingual documentation.

### Phase 1 — Clipboard MVP (delivered)

- Windows Unicode text clipboard read/write adapter and sequence watcher.
- LAN HTTP Push/Pull API for iPhone Shortcuts.
- Device ID, SHA-256 deduplication and feedback-loop suppression.
- Runtime diagnostics, tests and a runnable `v0.1.0` package.

### Phase 2 — Platform hardening, trust and discovery (in progress; Sprints 2.1–2.5 delivered)

Goal: make the platform safe and diagnosable before adding more payload types.

Planned work:

- Versioned configuration schema, defaults, migration rules and redacted config diagnostics.
- Stable error envelope, request IDs, API capability endpoint and compatibility tests.
- Device identity, pairing flow, token storage, allow-list, revoke/reset and safe first-run UX.
- Sprint 2.2 now provides an explicit pairing-code flow, persisted peer registry and revoke/list APIs;
  peer-token authentication and automated provisioning remain pending.
- Sprint 2.3 now provides optional UDP discovery as an ephemeral, untrusted reachability list;
  peer health, authenticated outbound transport and automatic provisioning remain pending.
- Sprint 2.4 now accepts paired peer tokens at the protected HTTP boundary when authentication
  is enabled; expiry, rotation and user-facing provisioning remain pending.
- Sprint 2.5 now provides peer capability probes and best-effort local clipboard forwarding;
  durable sync, retries, receipts and offline replay remain Phase 3 work.
- LAN discovery through mDNS or UDP broadcast, plus manual IP/port fallback.
- Peer registry with online/last-seen/capability state and network-change recovery.
- TLS or an equivalent authenticated local channel; no silent downgrade on paired links.
- Retry policy, request timeouts, rate limits and payload-size policy shared by all modules.
- Persistent boundary decision: what survives restart, retention, encryption-at-rest and deletion.
- Windows firewall/service/tray installation design, health diagnostics and support bundle.

Exit criteria: two paired devices can discover each other, authenticate, report capabilities,
show useful diagnostics and reject unpaired traffic; configuration and upgrade behavior are
documented and tested.

### Phase 3 — Generic sync engine, rich clipboard and history (in progress; Sprint 3.1 delivered)

Goal: stop making every module invent its own synchronization semantics.

Sprint 3.1 provides the generic Envelope, bounded persistent Job store and read-only job
inspection. It currently wraps clipboard forwarding without claiming reliable delivery.

Planned work:

- Generic `Envelope` with event ID, origin device, content type, MIME type, hash, size,
  timestamp, TTL, priority and optional reply/correlation ID.
- Capability negotiation and content fallback, for example image → preview or text metadata.
- Idempotent delivery, ordering rules, conflict policy and per-device sync filters.
- Outbound peer transport so Windows changes can be delivered without requiring a phone Pull.
- Bounded clipboard history, search, pin/delete/clear and privacy-aware retention.
- Image clipboard formats, HTML/RTF where supported, screenshots and format conversion.
- Persistence behind a storage interface with migrations and export/import.

Exit criteria: a paired phone or desktop client receives a Windows clipboard change through an
authenticated outbound path; rich content degrades safely; history survives a configured
restart policy; duplicate and conflict tests pass.

### Phase 4 — LAN collaboration modules

Goal: move work context, not only clipboard text.

Planned modules:

- File transfer with metadata, thumbnails, chunking, checksum verification, progress,
  pause/resume/cancel, expiration and safe destination handling.
- URL push with title/preview metadata, browser hand-off and optional URL privacy controls.
- Image sending as a first-class payload, including compression and size thresholds.
- Notification delivery with severity, expiry, acknowledgement and action payloads.
- Screenshot capture/send and “send current context” composite actions.
- A unified transfer/job status API for queued, active, completed, failed and cancelled work.

Exit criteria: files and URLs can be sent in both directions on a trusted paired LAN with
resume and integrity checks; notifications are bounded and observable; modules do not take
direct dependencies on each other.

### Phase 5 — Multi-client and device ecosystem

Goal: make the protocol useful beyond one Windows host and iPhone Shortcuts.

- Native iOS/iPadOS client where background and notification capabilities justify it.
- Android client with share-sheet integration and foreground/background policy.
- macOS adapter for NSPasteboard, notifications, files and launch-at-login.
- Linux adapter for Wayland/X11 clipboard differences, notifications and service startup.
- Windows tray and Windows service modes with clear clipboard-access trade-offs.
- Headless CLI and a read-only web diagnostic page; the web UI must not become a security
  bypass for paired APIs.
- Device roles, multiple Windows hosts, per-device subscriptions and “send to selected
  device” versus “broadcast to a group”.

Exit criteria: at least two non-Windows clients interoperate with the documented protocol,
capability negotiation works, and platform-specific limitations are visible to users.

### Phase 6 — Automation, integrations and plugin SDK

Goal: turn the platform into a composable local workflow engine.

- Stable plugin manifest, capability declaration, lifecycle hooks, configuration namespace,
  permission model and version compatibility rules.
- CLI commands, hotkeys, command palette and automation rules such as “when URL arrives,
  open in browser” or “when screenshot arrives, save to folder”.
- Browser extension, OS share targets and webhooks for local applications.
- Event filters, transformations and approval gates for sensitive actions.
- Optional community modules with clear signing, review and sandboxing expectations.

Exit criteria: a third-party module can be developed against the SDK without modifying core
composition code, and permissions/upgrade behavior are explicit.

### Phase 7 — Stable platform release (`v1.0.0`)

- Freeze and document the `v1` protocol compatibility policy.
- Secure defaults, threat model, security review, fuzzing and dependency audit.
- Signed installers/artifacts, automatic update opt-in, rollback and migration tooling.
- Performance budgets for clipboard latency, file throughput, memory and disk retention.
- Crash recovery, offline queue recovery, observability and support bundle.
- Full bilingual user/developer docs, example clients, conformance suite and release checklist.

## High-value ideas beyond the original Phase 2/3 list

These are worth considering because they strengthen the platform rather than adding isolated
features:

- **Capability negotiation:** clients advertise what they can read/write before a transfer;
  this prevents rich content from breaking older clients.
- **Offline queue and delivery receipts:** users can understand whether a payload is queued,
  delivered, rejected or expired instead of guessing from a timeout.
- **Selective synchronization:** per-device subscriptions, content filters and privacy rules
  prevent every device from receiving every clipboard item.
- **History and privacy controls:** search is useful, but retention, redaction and one-click
  purge are essential because clipboard data can contain secrets.
- **Composite context transfer:** send a URL, screenshot, selected text and source metadata as
  one job, which is closer to real cross-device work than isolated primitives.
- **Diagnostics and conformance tools:** a CLI/support bundle and protocol test suite reduce
  the cost of adding clients and troubleshooting home networks.
- **Safe automation:** event filters, approvals, TTLs and permissions make local workflows
  powerful without turning a paired device into an unrestricted remote-control channel.
- **Graceful degradation:** a client should receive a preview or text fallback when it cannot
  render the original MIME type.
- **Privacy-preserving telemetry:** default to no cloud telemetry; if enabled later, make it
  opt-in, local-first and content-free.
- **Multi-host routing:** allow a laptop, desktop and home server to coexist without creating
  duplicate loops or ambiguous “latest” state.

## Dependency order

```text
Trust + discovery
        ↓
Generic envelope + capability negotiation + outbound transport
        ↓
Persistence + history + conflict/offline policy
        ↓
Rich clipboard
        ↓
Files / URLs / images / notifications
        ↓
Additional clients
        ↓
Automation + plugin ecosystem + v1 hardening
```

Feature work may run in parallel only when it does not bypass an upstream contract. For
example, file transfer can prototype its UI early, but production file delivery must wait for
the shared trust, transport, checksum and job-state contracts.

## Prioritization

- **P0 — platform safety:** pairing, authentication, protocol compatibility, data loss
  prevention, corruption detection, recovery and privacy controls.
- **P1 — core user value:** automatic outbound clipboard delivery, rich clipboard, history,
  files, URLs, notifications and the first two additional clients.
- **P2 — adoption and productivity:** tray UX, share sheets, browser extension, CLI, rules,
  screenshots and composite context transfer.
- **P3 — experiments:** cloud relay, public plugin marketplace, AI transformations and other
  features that add external services or significant security scope.

When choosing between features, score user value, platform leverage, security risk, dependency
impact and testability. Prefer work that unlocks several modules while keeping the core local.

## Sprint and release gates

Every phase is split into Sprints with one demonstrable outcome. A Sprint must record:

1. Problem statement and user scenario.
2. Scope, non-goals and dependency assumptions.
3. Public contract, data model and security/privacy impact.
4. Implementation plan and test strategy.
5. Documentation and configuration changes.
6. Acceptance evidence, known limitations and follow-up work.

Before a phase release, require:

- Unit, integration, cross-platform build and relevant manual LAN tests.
- Protocol compatibility and negative/error-path tests.
- Security review for new network, storage, execution or permission behavior.
- Performance and resource checks appropriate to the payload size.
- Updated English and Chinese docs, examples, changelog and rollback notes.
- A packaged artifact that another person can run from a clean directory.

## Non-goals and guardrails

- Core workflows must not require a cloud account or vendor relay.
- Do not promise an iOS background daemon where the platform does not allow one; use Shortcuts
  or a native client with explicit lifecycle constraints.
- Do not expose unauthenticated LAN APIs to the public internet.
- Do not make feature modules import each other directly; use contracts and events.
- Do not store clipboard/file content indefinitely by default.
- Do not call a demo or a one-way prototype “synchronization” until delivery state and error
  behavior are defined.
