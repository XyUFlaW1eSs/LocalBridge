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
.\dist\localbridge.exe -check-config -config .\configs\config.yaml
```

Set a stable `device.id`. Keep `server.host` as `0.0.0.0` for LAN use and choose a port that
is not occupied. Never commit `configs/config.yaml` if it later contains credentials.

The current root schema is `version: 1`. Existing files without a version are treated as legacy
version 0 and migrated in memory; the service does not rewrite them. After validating a legacy
deployment, add `version: 1` manually. A future version is rejected instead of being guessed.
The `-check-config` command performs the same strict load, prints redacted effective JSON and exits
without starting the GUI or server.

To collect troubleshooting data without starting the service, run
`localbridge.exe -support-bundle .\\support.zip -config .\\configs\\config.yaml`. The ZIP contains
only redacted configuration, runtime information and state-file metadata; it excludes credentials,
device identity, local paths, TLS key/certificate paths or contents, payloads and state-file bodies.
The destination must not already exist.

For a LAN deployment, enable the Phase 2 transition authentication before allowing other
devices to connect:

```yaml
security:
  credential_protection: required
  credential_store_path: "data/credentials.json"
  auth_enabled: true
  bearer_token: ""
  bearer_token_ref: management
  pairing_code: ""
  pairing_code_ref: pairing
  peer_token_ttl: 720h
  token_overlap_ttl: 10m
```

Generate a token outside the repository, for example with `[guid]::NewGuid().ToString("N")`,
then write both secrets through stdin (or omit the pipe for hidden terminal input):

```powershell
"replace-with-a-long-random-token" | .\dist\localbridge.exe -config .\configs\config.yaml -credential-action set -credential-name management
"replace-with-a-local-pairing-code" | .\dist\localbridge.exe -config .\configs\config.yaml -credential-action set -credential-name pairing
.\dist\localbridge.exe -config .\configs\config.yaml -credential-action status -credential-name management
```

The management token must contain at least 16 characters. Secret values are never accepted as
arguments or printed by set/status/delete. The store is current-user DPAPI protected, versioned,
size-bounded and atomically replaced with owner-only permissions. A destination entry update
preserves unrelated entries. Deletion uses `-credential-action delete`.

To pair another LocalBridge host, set a local `security.pairing_code`, then call
`POST /api/v1/devices/pair` with the peer's ID, address, port and capabilities. Store the
returned peer token on the peer side. It expires after `peer_token_ttl`; rotate it before expiry
with `POST /api/v1/devices/{id}/token/rotate`. The old credential remains usable only for the
short overlap period. Under Windows `auto` or `required`, registry v4 stores current and previous
peer tokens only as current-user DPAPI ciphertext. A v3 plaintext registry is rewritten atomically
only after every token is protected; protection/decryption failure aborts startup and preserves the
original file. Version 1 registry
entries without a persisted token become `repair_required` and must be paired or locally rotated.
Set `discovery.enabled: true` only when UDP broadcast on the configured port is acceptable;
discovery does not grant trust. A positive
`device.health_interval` enables best-effort peer health checks and local clipboard forwarding.

`credential_protection: auto` and `required` require Windows DPAPI. Other operating systems report
protected storage as unsupported and refuse startup; only explicit `disabled` permits legacy plaintext
storage. `disabled` is not encryption, and base64 encoding is never described as protection. Switching
from protected registry v4 to disabled is rejected. Existing inline YAML values remain compatible in
Windows `auto` mode but are not migrated or deleted automatically; use references and clear them manually.

### TLS transport

TLS is disabled by default. To enable it, configure a matching certificate/private-key pair:

```yaml
server:
  tls_enabled: true
  tls_cert_file: "C:\\path\\to\\localbridge.crt"
  tls_key_file: "C:\\path\\to\\localbridge.key"
```

The application validates both files before composing modules and starts HTTPS only, with TLS
1.2 minimum (TLS 1.3 preferred). It never silently falls back to HTTP. The leaf certificate's
lowercase SHA-256 fingerprint appears in capabilities and discovery and must be entered in the
pairing record for `secure: true`. Discovery is only a hint and does not establish trust.

Self-signed certificates can be pinned by peer-to-peer transport, but Windows browsers/WebView2
and iPhone Safari/Shortcuts may reject them until the private CA or certificate is installed in
the platform trust store. Initial fingerprint confirmation/distribution UX, automatic certificate
rotation, and an iPhone trust-install flow are future work.

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
Invoke-RestMethod http://127.0.0.1:8899/api/v1/system/config
```

Health is intentionally unauthenticated for local diagnostics. When authentication is
enabled, add the same `Authorization: Bearer <token>` header to clipboard and capabilities
requests, including the iPhone Shortcut. Every response includes `X-Request-ID`; preserve it
when reporting a failure.

The effective-configuration response is redacted: credentials are represented only by configured
booleans and the source file path is omitted. With authentication disabled, loopback may read it
without a token and remote access is rejected. With authentication enabled, all requests require
the management token specifically; peer tokens are rejected.

File management endpoints under `/api/v1/files/` have an additional boundary: with
`security.auth_enabled: false`, they accept only loopback requests and reject LAN clients with
`403`. With authentication enabled, use the bearer or paired peer token for management. The
`/share/<token>` and `/receive/<token>` pages are intentionally public capability URLs for phone
access; keep their random, expiring URLs inside the trusted LAN. The QR endpoints provide both a
renderer-neutral JSON URL at `/api/v1/files/shares/<id>/qr` and a locally generated PNG at
`/api/v1/files/shares/<id>/qr.png`. The embedded browser GUI is available at
`http://127.0.0.1:8899/app/` when TLS is disabled, or `https://127.0.0.1:8899/app/` when enabled,
with Shares, Receive records and Settings views. Browser uploads use
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
