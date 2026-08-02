# Clipboard Module Design

## Responsibility

The clipboard module translates between the stable HTTP/EventBus contracts and the operating
system clipboard. It owns deduplication, the latest in-memory item and the remote-write
suppression window.

## Interfaces

`Platform` exposes `ReadText`, `WriteText` and `Watch`. The module's `Watcher` adapts the
platform callback to domain handling. Windows implements the platform using the
Win32 Unicode clipboard APIs and `GetClipboardSequenceNumber`. Other operating systems use a
safe no-op adapter so the server remains buildable while their native integrations are planned.

## Data rules

- Only UTF-8 text is accepted in Sprint 1.
- Hashes are SHA-256 over the UTF-8 content.
- A duplicate hash is ignored regardless of source device.
- A remote write is suppressed for two seconds, which prevents the local watcher from creating
  a feedback loop after `WriteText`.
- The latest item is memory-only and is lost on process restart.

## Known limitations

- The watcher is polling-based to keep Win32 message-window lifecycle out of the domain layer.
- Clipboard ownership and access can transiently fail while another Windows process holds the
  clipboard; the watcher retries on its next interval.
- Phase 1 does not carry images, HTML, files, encryption or device authentication.
