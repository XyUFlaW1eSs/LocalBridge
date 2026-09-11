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
access; keep their random, expiring URLs inside the trusted LAN. The QR endpoint currently returns
only a local-rendering URL payload; it does not generate an image yet.

To keep the logs visible in the current PowerShell window, run:

```powershell
.\scripts\run.ps1 -Executable .\dist\localbridge-v0.1.0\localbridge.exe -Config .\configs\config.yaml
```

The process logs module startup, HTTP requests, request IDs, capabilities, clipboard Push/Pull,
deduplication and Win32 write errors. Clipboard content and authentication tokens are never logged.

From the iPhone, use the Windows private IPv4 address in the Shortcut URL. Test Push, then
Pull, then copy text directly on Windows and confirm the latest endpoint changes after the
watch interval.

## 5. Service installation

Service-manager integration is intentionally not part of Sprint 1. For a first deployment,
run the executable from a supervised user session because Windows clipboard access belongs to
the interactive desktop session. A future Windows service/tray design must preserve that
session requirement and document its security boundary.

## Rollback

Stop the process, replace the executable with the previous version, and restart with the same
configuration. The Phase 1 latest item is memory-only, so rollback does not migrate storage.
