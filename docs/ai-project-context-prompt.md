# AI Project Context Prompt

[简体中文版本](ai-project-context-prompt.zh-CN.md)

Copy the prompt below into another coding model when asking it to understand or continue
LocalBridge. The model should still inspect the repository and treat the code and tests as the
implementation source of truth.

```text
You are continuing development of LocalBridge, an open-source Go project in the repository
root. First inspect the repository; do not assume that this prompt is newer than the code.

## Product identity

LocalBridge is a local-first, modular and pluggable LAN cross-device collaboration platform.
It started as a Windows-to-iPhone clipboard bridge, but clipboard is only the first module.
The long-term platform moves clipboard content, files, URLs, images, notifications and safe
actions between trusted devices without requiring a cloud account or vendor relay.

## Current baseline

- Phase 0 and Phase 1 / Sprint 1 are delivered.
- The current release line is v0.1.x; the delivered package is v0.1.0.
- The current production path is Windows host + iPhone Shortcuts on a trusted private LAN.
- Phase 1 supports UTF-8 text clipboard content only.
- Windows uses a Win32 clipboard adapter and a polling watcher. Other platforms use a safe
  no-op adapter so the core remains buildable and testable.
- Latest clipboard data is memory-only in the current implementation.
- Phase 2.1 added optional Bearer authentication, request IDs and capability discovery.
- Phase 2.2 added an explicit pairing endpoint and persisted peer registry. Phase 2.3 added
  optional UDP discovery as an ephemeral, untrusted reachability list. Peer-token
  authentication, TLS, outbound peer delivery and a public-client ecosystem are not implemented yet.
- Windows-to-iPhone is currently a Pull flow: Windows updates in-memory latest; the iPhone
  must GET it. Do not claim automatic push until outbound transport and peer registration exist.

## Current HTTP contract

- `GET /api/v1/system/health`
- `POST /api/v1/clipboard`: accepts a JSON object with `content`, compatibility alias `text`,
  optional `type`, `mime_type`, `device_id`, `id` and `hash`; it also accepts raw UTF-8
  `text/plain`. It returns `{accepted, item}`.
- `GET /api/v1/clipboard/latest`: returns the latest item directly, or 404 when empty.
- `GET /api/v1/clipboard/status`: returns module, enabled and has_latest.
- `GET /api/v1/devices`: lists the local device and paired peers without peer tokens.
- `GET /api/v1/devices/{id}`: returns one paired peer without its token.
- `POST /api/v1/devices/pair`: pairs a peer with configured `security.pairing_code` and returns
  a generated peer token once; re-pairing rotates it.
- `DELETE /api/v1/devices/{id}`: revokes a peer.
- `GET /api/v1/devices/discovered`: lists ephemeral UDP discovery hints; discovered devices are
  not trusted and are not added to the paired registry.
- Phase 1 default maximum is 1048576 UTF-8 bytes. Hashes are SHA-256; duplicate hashes are
  ignored. Remote writes have a short suppression window to prevent watcher echo.
- The complete contract is in `docs/clipboard.md` and `docs/protocol.md`.

## Architecture and boundaries

- `cmd/localbridge`: CLI flags, signals and exit codes only.
- `internal/app`: dependency composition; it wires concrete modules.
- `internal/config`: configuration loading and validation.
- `internal/server`: standard-library HTTP server and system endpoints.
- `internal/module`: stable module lifecycle and route contract.
- `internal/eventbus`: non-blocking in-process notifications; it is not a durable queue.
- `internal/modules/device`: explicit pairing and persisted peer registry. Discovery must remain
  a reachability hint, not authorization.
- `internal/modules/clipboard`: clipboard domain behavior and platform interface.
- Planned layers are clients/adapters -> identity/trust/discovery/policy -> transport -> sync
  engine -> feature modules -> storage/observability.
- Feature modules must not import or directly control one another. Use public contracts and
  EventBus events. The core owns lifecycle, trust, transport and policy; modules own domain
  validation and platform application.

## Direction of future work

1. Phase 2 / v0.2.x: configuration hardening, peer-token authentication, token
   provisioning/rotation, diagnostics, shared retries/timeouts, persistence boundary and
   Windows service/tray design. Request IDs, capabilities, transition auth, explicit pairing
   and the peer registry are already partially delivered; UDP discovery is implemented only as
   an untrusted hint.
2. Phase 3 / v0.3.x: generic content envelope, capability negotiation, outbound delivery,
   delivery state, offline queue, history, rich clipboard, images, HTML/RTF and screenshots.
3. Phase 4 / v0.4.x: resumable/integrity-checked file transfer, URL push, image delivery,
   notifications, screenshots and composite context jobs.
4. Phase 5 / v0.5.x: native iOS/iPadOS, Android, macOS, Linux, Windows tray/service, CLI,
   diagnostics web page and multi-device targeting.
5. Phase 6 / v0.6.x: plugin SDK, manifests, capabilities, permissions, hotkeys, browser
   extension, Webhooks, scripting and safe automation rules.
6. Phase 7 / v1.0.0: stable protocol policy, security review, signed artifacts, upgrade/
   rollback, recovery, performance budgets and conformance tests.

See `docs/product-plan.md`, `docs/roadmap.md` and their Chinese versions for the full plan.

## Non-negotiable engineering rules

- Inspect current code, tests, git status and relevant docs before editing.
- Preserve unrelated user changes. Do not use destructive git commands.
- Prefer small, reviewable changes. Avoid speculative rewrites.
- Define or update public contracts before implementing cross-module behavior.
- Every feature change should update code, tests, docs, configuration examples and scripts as
  applicable. Keep English and Chinese docs synchronized.
- Keep core workflows local-first. Never silently introduce a cloud dependency or public relay.
- Never expose unauthenticated LAN APIs to the public internet.
- Do not log clipboard/file/message bodies. Log metadata, sizes, hashes, IDs, source, status
  and errors instead.
- Define idempotency, ordering, retry, timeout, cancellation, expiration, integrity and
  restart behavior for networked payloads.
- Treat OS limitations as explicit capability differences and implement safe fallback behavior.
- Keep the standard library preference unless a dependency has a clear maintenance and security
  justification.

## Development workflow

1. Read `README`, `docs/architecture`, `docs/product-plan`, `docs/roadmap`, `docs/developer-guide`,
   relevant protocol docs, ADRs and Sprint records.
2. Inspect the affected packages, tests and configuration. State the current behavior and the
   smallest useful implementation slice.
3. For a boundary change, write a design note or ADR and document the API/data model first.
4. Implement with focused tests, including negative paths and restart/network failure behavior.
5. Run `gofmt`, `go test ./...`, `go vet ./...`, the relevant platform build and `git diff --check`.
6. Update English/Chinese docs, examples, changelog or release notes when behavior changes.
7. Report changed files, verification results, known limitations and the next safe step.

## How to handle an assigned task

Do not broaden the task merely because a future feature is attractive. If a requested feature
depends on missing trust, transport or persistence foundations, explain the dependency and
implement the smallest safe prerequisite or ask for direction when the choice is material.
Never claim a feature is complete based only on a successful HTTP status; verify the end-to-end
user outcome and failure behavior.

When responding, use this structure:

1. Current understanding and assumptions.
2. Plan and affected boundaries.
3. Implementation summary.
4. Tests/builds/manual checks.
5. Known limitations and follow-up.
```
