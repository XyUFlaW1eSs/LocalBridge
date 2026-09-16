# Sprint 4.2D — Native Windows host

[简体中文](sprint-4.2d.zh-CN.md)

## Status

Delivered. Phase-level release validation still includes Explorer integration on the final install image.

## Goal

Host the existing local Web GUI in a real Windows window so close-to-tray is enforceable without
duplicating application state or granting the frontend direct filesystem access.

## Delivered design

- A pure-Go `github.com/jchv/go-webview2` host creates one WebView2 window on a locked OS thread and
  navigates it to the loopback `/app/` origin.
- Normal launch opens the native window. Tray File Share, Receive History and Settings actions
  navigate and restore the same window instead of creating browser tabs.
- The host subclasses the window procedure. With `minimize_to_tray=true`, `WM_CLOSE` hides the window;
  otherwise it closes the host and requests normal application shutdown.
- Tray Exit and process shutdown force a real host close, then stop modules and HTTP cleanly.
- If WebView2 creation fails, LocalBridge logs a bounded error and opens the system browser. The
  service remains usable, but browser windows cannot provide close interception.

## Threading and security

All WebView calls run through the host UI thread. The host exposes no JavaScript-to-Go bindings and
loads only the existing loopback HTTP GUI, so network authentication and module APIs remain the
security boundary. The dependency and its loader are MIT-licensed; end users need the Microsoft
Edge WebView2 Runtime, present on current Windows installations.

## Verification evidence

- Full unit tests, `go vet`, frontend syntax check and Windows release build pass.
- Linux builds still compile because the host is isolated by Windows build tags.
- Interactive Windows smoke test used an isolated port and data directory: the WebView loaded `/app/`,
  health returned `ok`, posting `WM_CLOSE` hid the window while the process and HTTP service remained
  alive, and a tray double-click restored the same titled window.
- The isolated smoke configuration was removed and its generated files were deleted after testing.

