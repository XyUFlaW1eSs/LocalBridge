# Sprint 4.2B — Windows shell foundation

[简体中文](sprint-4.2b.zh-CN.md)

## Status

Foundation delivered. Receive approval followed in Sprint 4.2C and the native host in Sprint 4.2D.

## Goal

Connect the persisted desktop settings and file-transfer events to real, current-user Windows
effects without requiring administrator privileges or coupling feature modules to Win32 code.

## Delivered

- `internal/native` isolates Windows code behind build tags and supplies a no-op adapter elsewhere.
- `auto_start` synchronizes `HKCU\Software\Microsoft\Windows\CurrentVersion\Run\LocalBridge`.
- `explorer_context_menu` synchronizes a per-user `LocalBridgeShare` verb, label, icon,
  multi-selection policy and command under `HKCU\Software\Classes\*\shell`.
- `localbridge.exe -share <file> [file...]` validates distinct regular non-symlink files, reuses a
  running loopback service when available, and creates one share for the selection.
- A native Windows notification-area icon provides File Share, Receive History, Settings and Exit
  actions. Exit requests follow normal application shutdown rather than terminating the process.
- File creation/completion publishes metadata-only `file.sent` and `file.received` EventBus events.
  The Windows adapter displays completion notifications and applies master/per-direction sound
  settings without exposing content, tokens or paths.
- Browser-owned share files are reclaimed on single delete, clear-all and expiry. Native-path shares
  never delete the user's source files, and stale browser temporary files are removed on restart.

## Security boundaries

- Registry writes are limited to the current user and use quoted executable/configuration paths.
- Explorer arguments are normalized and checked with `Lstat`; directories, missing files, symbolic
  links, duplicates and empty selections are rejected.
- The Explorer command prefers the loopback management API; remote file-path management remains
  forbidden without authentication.
- Event payloads contain identifiers, counts and sizes only.

## Verification

- Fake-registry tests cover enable/disable, labels, icons, multi-select and command quoting.
- Path tests cover regular files, missing files, directories, duplicates and symbolic links where
  the host permits creating them.
- File tests cover single delivery of sent/received events and idempotent share retries.
- `go test ./...`, `go vet ./...`, the trimmed Windows build and `git diff --check` pass.
- Linux package tests are compile-checked with `go test -c`, and the Linux command builds with the
  no-op native adapter.

## Known limitations and next slice

This foundation originally opened the embedded page in the user's browser. Sprint 4.2C subsequently
delivered receive approval, and Sprint 4.2D delivered the WebView2 host plus interactive close/restore
smoke evidence. Registry behavior remains covered by deterministic fake-registry tests; release
validation should still exercise Explorer on the target installation image.
