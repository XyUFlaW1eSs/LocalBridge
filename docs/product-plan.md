# Product Plan and Capability Map

[简体中文](product-plan.zh-CN.md)

## The original idea

LocalBridge began with a practical problem: move clipboard content between a Windows PC and
an iPhone on a trusted LAN. The broader product idea is more valuable:

> Build a local-first, modular and pluggable cross-device collaboration platform, not a
> single-purpose clipboard synchronizer.

The platform should make a device feel like a useful extension of another device without
requiring a cloud account. Clipboard text is the first proof that the transport, device and
module boundaries are useful; it is not the final product boundary.

## Product principles

- Local-first by default; cloud relay is never a prerequisite for core workflows.
- A device, transport and capability are separate concepts.
- Every payload has explicit ownership, delivery state, integrity and retention behavior.
- Modules own domain behavior; the core owns lifecycle, trust, transport and policy.
- The simplest one-host setup remains simple even as the multi-device design grows.
- Privacy is a product feature: local data should have retention, filtering and purge controls.
- Clients may be limited by their operating systems; the protocol must expose limitations rather
  than pretending all platforms behave the same.

## Users and high-value journeys

| User | Journey | What makes it valuable |
|---|---|---|
| Individual with phone + PC | Copy text, URL, image or file on one device and use it on the other | Removes repetitive self-messaging and cloud uploads |
| Developer | Send a link, stack trace, screenshot or test artifact between machines | Preserves context during debugging and research |
| Home/office user | Notify another device, share a file or send a command to a selected device | Makes the LAN a personal workspace |
| Power user | Trigger rules from hotkeys, Shortcuts, browser or CLI | Turns repeated actions into local automation |
| Integrator | Add a client or module without changing the core | Makes the platform grow through adapters |

## Brainstorm backlog

### Foundational platform capabilities

- Device identity, pairing, revocation, trust reset and human-readable device names.
- mDNS/UDP discovery plus manual fallback for networks that block multicast.
- A generic content/action envelope with MIME type, capability requirements, hash, size, TTL,
  priority, origin and correlation ID.
- Capability negotiation and safe fallback previews.
- Outbound delivery, acknowledgements, retries, expiration, offline queue and cancellation.
- End-to-end integrity checks, chunked/resumable transfer and rate limiting.
- Persistent state with migrations, bounded retention, export/import and privacy deletion.
- Conflict policies such as last-writer-wins, ask-user, per-device preference or append-to-history.
- Diagnostics CLI, support bundle, protocol conformance suite and network simulation tests.

### Content and collaboration modules

- Clipboard text, image, HTML/RTF and screenshots.
- Clipboard history, search, pinning, redaction and one-click purge.
- File transfer, folder transfer, safe destination selection, preview, resume and progress.
- URL push, metadata preview, browser hand-off and “open on selected device”.
- Notification push with severity, expiry, acknowledgement and action buttons.
- Composite “send context” jobs containing selected text, URL, screenshot and source metadata.
- Device-to-device notes, small snippets and ephemeral messages.
- Presence and selected-device targeting without turning the platform into a chat application.

### Clients and integrations

- iPhone/iPad Shortcuts first, then a native iOS client if background/notification needs justify it.
- Android share sheet and notification integration.
- Windows clipboard/tray/service modes with clearly documented behavior.
- macOS pasteboard and Linux Wayland/X11 adapters.
- Headless CLI, read-only diagnostics web page, browser extension and local Webhooks.
- Hotkeys, command palette, OS share targets and scripting API.

### Reliability, privacy and operations

- Secure defaults, no unauthenticated public exposure, token rotation and audit events.
- Content-free opt-in telemetry only; default logs never contain payload content.
- Crash recovery, queue recovery, safe shutdown and clear partial-transfer semantics.
- Signed artifacts, installer/service registration, upgrades, rollback and version compatibility.
- Resource budgets for memory, disk retention, CPU, clipboard latency and file throughput.

## What should not be built early

- A cloud account, mandatory relay or always-on remote service.
- A general chat/social layer that competes with the platform's focused collaboration purpose.
- Arbitrary remote command execution without a narrow capability and explicit approval model.
- A plugin marketplace before manifests, permissions, signing and compatibility are defined.
- Rich UI before the protocol, device trust and delivery state are reliable.

## Product success signals

The platform is progressing when:

1. A new device can pair and diagnose itself without reading source code.
2. A module can reuse trust, transport, job state and policy instead of implementing them again.
3. A payload's state is explainable: queued, delivered, rejected, expired, cancelled or failed.
4. Unsupported formats degrade safely and never silently corrupt data.
5. A release can be installed, upgraded, rolled back and recovered from a clean directory.
6. Users can see and remove retained private content.

See [Roadmap](roadmap.md) for phases, dependencies, priorities and release gates.
