# Sprint 4.2 — Desktop GUI, Mobile UX and Share Management

[简体中文](sprint-4.2-desktop-gui.zh-CN.md)

Status: planned; starts only after Sprint 4.1 passes review.

## Goal

Turn the file-transfer contract into an understandable user workflow on Windows while keeping
the same mobile pages useful on iPhone. The UI is web-first and must not duplicate transfer
state or bypass the backend security boundary.

## Desktop UI contract

The application has three top-level sections:

1. **File Share** — QR/URL result, file picker/drop zone and active share table. A multi-file
   selection is one expandable share row; child files show size, checksum, progress, state and
   remove/retry controls. Clear-one and clear-all actions are explicit and confirm destructive
   deletion where required.
2. **Receive History** — receiver links, active uploads and completed files, with status,
   resume/retry, checksum result, destination and privacy deletion.
3. **Settings** — startup, close-to-tray, Explorer context menu, reset defaults, automatic
   acceptance, notification sounds and receive/send sound selection.

## Desktop shell decision

The initial GUI is a static web frontend served by LocalBridge so it can be tested in a browser
and reused by the iPhone page. The Windows release shell should use a WebView2-based Go wrapper
(Wails is the default candidate) only after the frontend contract is stable. The shell must own
window lifecycle, tray menu, single-instance behavior, startup registration and Explorer
context-menu installation; the backend remains independently runnable for CLI/headless use.

## Acceptance

- A clean Windows launch opens the GUI and can create/revoke a multi-file share.
- File picker `multiple` and drag/drop add files without leaking source paths to the browser UI.
- Share rows expand/collapse and clear-one/clear-all update durable state after restart.
- QR and copy-URL actions produce the exact public mobile URL.
- Receive history shows interrupted, resumed, completed, failed and deleted records.
- Settings are atomic, validated, redacted in diagnostics and reflected in behavior.
- Closing the shell follows the configured close-to-tray policy; tray exit stops the process.
- An iPhone can open the same share/receive page and use native browser save/share behavior.

