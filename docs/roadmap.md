# LocalBridge Roadmap

This document is the detailed companion to the root-level [`ROADMAP.md`](../ROADMAP.md).

## Phase 0 — Core foundation (delivered)

- Repository layout and Go module.
- Configuration, structured logging and lifecycle management.
- Standard-library HTTP server with versioned health API.
- Module interface and non-blocking EventBus.
- Architecture, protocol, deployment and developer documentation.

## Phase 1 — Clipboard MVP (delivered)

- Windows text clipboard read/write adapter.
- Clipboard sequence watcher.
- LAN HTTP Push/Pull API for iPhone Shortcuts.
- Device ID, SHA-256 deduplication and feedback-loop suppression.
- Test coverage and deployment/runbook materials.

## Phase 2 — Trust and discovery

- Pairing token and device allow-list.
- LAN discovery using mDNS or UDP broadcast.
- Connection diagnostics and explicit device management.

## Phase 3 — Rich clipboard and history

- Images and HTML formats.
- Bounded local history and conflict policy.
- Optional persistence with a documented storage boundary.

## Phase 4 — Collaboration modules

- File transfer, URL push, notifications and screenshots.
- Android, macOS and Linux client adapters.
- Desktop tray/service integration that preserves interactive clipboard access.
