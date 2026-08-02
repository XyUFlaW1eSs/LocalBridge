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
```

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
