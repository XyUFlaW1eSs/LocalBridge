# File Sharing Product Plan

[简体中文](file-sharing-product-plan.zh-CN.md)

## Product decision

The file workflow is not modeled as “copy a file into a clipboard”. It is a resumable,
expiring transfer session with a batch of files. The Windows application creates a share;
the receiving device opens a short URL or scans a QR code; the mobile page lists the files
and exposes native browser download/upload behavior.

The UI will be web-first so the same tested transfer page works on Windows and iPhone. A
Windows desktop shell will later host that UI and own tray, startup and Explorer integration.
The HTTP service remains the source of truth for transfer state, so a browser or Shortcut can
still be used when the desktop shell is not available.

## Share model

One selection action creates one `share` record. A share contains one or more `file` records.
The share has an opaque public token, expiry, revocation state and a bounded list of files.
The public token grants access only to that share; it is not a device-pairing credential.

Required states:

```text
created -> active -> completed / expired / revoked
                  \-> failed
```

Each file records a safe display name, size, MIME type, SHA-256 checksum, transfer progress
and a server-side source handle. Client-visible JSON must never expose unrestricted local
filesystem paths.

## Windows sharing flow

1. Open the **File Share** page.
2. Select multiple files through a file picker or drag them into the drop area.
3. Review the selected files and remove individual files or clear the selection.
4. Create one share; the page shows the QR code, URL, expiry and a collapsible file table.
5. Revoke one share or clear all active shares.
6. Watch per-file state, bytes, speed, checksum and errors in the sharing table.

## Mobile download flow

The QR code opens a mobile HTML page. It lists the share and each file. A file link uses
authenticated public-share access and supports HTTP `Range` requests. The page must avoid
pretending that iOS can save arbitrary files silently: the browser's download/share sheet is
the user-facing save path. Images may be opened inline; other files use a download link.

## Mobile upload flow

The same style of page can expose a receive token and a multi-file picker. The browser uploads
each file in bounded chunks using `Content-Range`; after an interruption it asks the server for
the committed offset and continues from there. Windows receives into a configured directory
and records the final checksum and file name. Automatic acceptance is a setting and must be
off by default.

## Transfer contract requirements

- `Range` downloads return `206`, `Content-Range`, `Accept-Ranges` and a stable checksum.
- Upload chunks carry `Content-Range: bytes start-end/total` and a per-chunk or whole-file hash.
- Resume status returns the committed offset and transfer state.
- Repeating the same committed chunk is idempotent; overlapping or invalid ranges are rejected.
- A transfer is complete only after the server verifies the whole-file SHA-256.
- Temporary files are not exposed as completed files and are atomically renamed on success.
- Shares and upload sessions expire and can be explicitly revoked or cancelled.
- File count, file size, total share size, chunk size, filename length and storage quota are bounded.
- Server paths are validated against the configured share/receive directories; traversal and
  symlink escapes are rejected.
- Logs include IDs, sizes, state and errors but never file contents, tokens or full private paths.

## Desktop pages

### File Share

Three visible regions are required: QR/URL share result, file selection/drop area and active
share table. A multi-file selection appears as one expandable share row with child file rows.

### Receive History

Shows received files and upload sessions with timestamp, source device/share, destination name,
size, checksum status, resumable state and failure reason. It supports retry, reveal in folder
and privacy deletion where the platform permits it.

### Settings

Settings are grouped into software behavior, security and notifications: launch at startup,
close-to-tray, Explorer context menu, restore defaults, automatic acceptance, notification
sound, receive sound and send sound. Changes are persisted atomically and are reflected in
the effective configuration endpoint without exposing secrets.

## Release gates

The feature is not ready for v1.0 until a Windows-to-iPhone download, iPhone-to-Windows upload,
multi-file share, QR scan, HTTP URL, interruption/resume, revoke/expiry, checksum mismatch,
path traversal rejection, settings persistence and clean restart have all been tested.
