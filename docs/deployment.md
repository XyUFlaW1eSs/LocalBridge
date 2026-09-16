# Deployment Runbook (Windows)

[简体中文](deployment.zh-CN.md)

## 1. Build

Install Go 1.24+ and build on Windows:

```powershell
go test ./...
go vet ./...
go build -trimpath -ldflags "-s -w" -o .\dist\localbridge.exe .\cmd\localbridge
```

## 2. Configure

```powershell
Copy-Item .\configs\config.example.yaml .\configs\config.yaml
notepad .\configs\config.yaml
```

Set a stable `device.id`. Keep `server.host` as `0.0.0.0` for LAN use and choose a port that
is not occupied. Never commit `configs/config.yaml` if it later contains credentials.

For a LAN deployment, enable the Phase 2 transition authentication before allowing other
devices to connect:

```yaml
security:
  auth_enabled: true
  bearer_token: "replace-with-a-long-random-token"
```

Generate a token outside the repository, for example with
`[guid]::NewGuid().ToString("N")` in PowerShell. The token must be at least 16 characters;
use a longer random value in practice. The server never logs it.

To pair another LocalBridge host, set a local `security.pairing_code`, then call
`POST /api/v1/devices/pair` with the peer's ID, address, port and capabilities. Store the
returned peer token on the peer side. Set `discovery.enabled: true` only when UDP broadcast on
the configured port is acceptable; discovery does not grant trust. A positive
`device.health_interval` enables best-effort peer health checks and local clipboard forwarding.

## 3. Firewall

Allow inbound TCP 8899 only on the Private profile, or replace the port with your configured
value:

```powershell
New-NetFirewallRule -DisplayName "LocalBridge (Private LAN)" `
  -Direction Inbound -Action Allow -Protocol TCP -LocalPort 8899 -Profile Private
```

Do not create a Public profile rule and do not port-forward this service from the router.

## 4. Run and verify

```powershell
.\dist\localbridge.exe -config .\configs\config.yaml
Invoke-RestMethod http://127.0.0.1:8899/api/v1/system/health
Invoke-RestMethod http://127.0.0.1:8899/api/v1/system/capabilities `
  -Headers @{ Authorization = "Bearer replace-with-a-long-random-token" }
```

Health is intentionally unauthenticated for local diagnostics. When authentication is
enabled, add the same `Authorization: Bearer <token>` header to clipboard and capabilities
requests, including the iPhone Shortcut. Every response includes `X-Request-ID`; preserve it
when reporting a failure.

File management endpoints under `/api/v1/files/` have an additional boundary: with
`security.auth_enabled: false`, they accept only loopback requests and reject LAN clients with
`403`. With authentication enabled, use the bearer or paired peer token for management. The
`/share/<token>` and `/receive/<token>` pages are intentionally public capability URLs for phone
access; keep their random, expiring URLs inside the trusted LAN. The QR endpoints provide both a
renderer-neutral JSON URL at `/api/v1/files/shares/<id>/qr` and a locally generated PNG at
`/api/v1/files/shares/<id>/qr.png`. The embedded browser GUI is available at
`http://127.0.0.1:8899/app/` with Shares, Receive records and Settings views. Browser uploads use
`POST /api/v1/files/browser-shares`; one multipart batch becomes one share and files are stored under
the configured `files.share_dir` (default `data/shared`), never under a client-provided path. When
authentication is enabled, local loopback management requests remain available to the local GUI;
LAN management clients still require a bearer or paired peer token.

The GUI settings document is stored at `settings.store_path` (default `data/settings.json`). It is
versioned, non-secret and atomically written with restrictive permissions. On Windows, `auto_start`
and `explorer_context_menu` now synchronize current-user HKCU keys, while the native tray exposes
share/receive/settings/exit actions. File completion events can produce tray notifications and sounds.
`auto_accept` is enforced: when disabled, new uploads remain pending until approved or rejected in
Receive History; no file bytes are accepted while pending. The Windows WebView2 host enforces
`minimize_to_tray`; when the runtime is unavailable LocalBridge falls back to the system browser,
where browser close cannot be intercepted. The QR image is generated locally
with the MIT-licensed `github.com/skip2/go-qrcode` dependency, so deployment does not require a CDN
or public QR service.

To keep the logs visible in the current PowerShell window, run:

```powershell
.\scripts\run.ps1 -Executable .\dist\localbridge-v0.1.0\localbridge.exe -Config .\configs\config.yaml
```

The process logs module startup, HTTP requests, request IDs, capabilities, clipboard Push/Pull,
deduplication and Win32 write errors. Clipboard content and authentication tokens are never logged.

From the iPhone, use the Windows private IPv4 address in the Shortcut URL. Test Push, then
Pull, then copy text directly on Windows and confirm the latest endpoint changes after the
watch interval.

## 5. Windows shell integration

Enable **Start with Windows** or **Explorer context menu** in `/app/#settings` to create per-user
HKCU entries; no administrator rights are required. Explorer invokes `localbridge.exe -share <files>`
and the command reuses a running loopback service when possible. The tray menu opens each GUI section
or requests graceful shutdown. Keep the process in an interactive user session because clipboard and
tray access belong to that desktop. This is not a Windows Service. The GUI uses the installed Microsoft
Edge WebView2 Runtime and the MIT-licensed pure-Go `github.com/jchv/go-webview2` host. Closing the
native window hides it when configured; use the tray menu to restore it or exit gracefully.

## Rollback

Stop the process, replace the executable with the previous version, and restart with the same
configuration. The Phase 1 latest item is memory-only, so rollback does not migrate storage.
