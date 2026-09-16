# LocalBridge

语言： [English](README.md) · [简体中文](README.zh-CN.md)

LocalBridge is a lightweight, local-first LAN collaboration platform. Its first usable
module synchronizes text clipboard content between a Windows PC and an iPhone through two
iPhone Shortcuts. No cloud account or relay service is required.

> Status: Phase 0, Phase 1 / Sprint 1, Phase 2 foundation including Sprint 2.8 TLS, Phase 3.1, Phase 4.1, the embedded
> file GUI and Sprints 4.2B–4.2D Windows shell, receive approval and native hosted window are delivered.
> URL/image modules and the remaining v1.0 roadmap are still in progress.

## What it does

```text
iPhone Shortcut --HTTP POST--> LocalBridge on Windows --Win32--> Windows Clipboard
iPhone Shortcut <--HTTP GET--- LocalBridge on Windows <--Win32-- Windows Clipboard
```

- `POST /api/v1/clipboard` accepts text from an iPhone Shortcut.
- `GET /api/v1/clipboard/latest` returns the latest item for the Pull Shortcut.
- A Windows watcher polls the Win32 clipboard sequence number and publishes local changes.
- SHA-256 content hashes prevent duplicate updates and clipboard feedback loops.
- EventBus and module boundaries keep file, image and notification features separate.

Windows clipboard changes are not currently pushed to the iPhone automatically. The watcher
updates the in-memory `latest` item; the iPhone must run the Pull Shortcut to GET it. This is
the Phase 1 design under iOS background execution constraints. Paired LocalBridge desktop
peers can now receive local clipboard events through a best-effort outbound path; durable
queue/retry semantics are planned for Phase 3.

The local Web GUI is available at `http://127.0.0.1:8899/app/`. It provides file sharing,
receive history and persisted settings. The browser uploads multipart files into the server's
controlled `files.share_dir`; it never sends or displays a real local path. The share page has
drag/drop and multi-select support, expandable file rows, URL copy and an offline-generated PNG
QR code. Windows now has per-user startup and Explorer context-menu synchronization, a native tray,
multi-file `-share` handling, completion notifications and a WebView2-hosted native window. Closing
that window obeys `minimize_to_tray`; tray actions restore the same window. If WebView2 is unavailable,
LocalBridge logs the failure and falls back to the system browser.
When `auto_accept` is disabled, incoming file metadata waits in Receive History for an explicit
approve/reject decision before any file bytes are accepted; the mobile page continues after approval.

Runtime logs are written to the current console. Use `scripts/run.ps1` to run a release package
in the foreground and see Push, Pull, deduplication and clipboard-write diagnostics. Clipboard
content itself is never logged.

## Quick start on Windows

Requirements: Go 1.24 or newer and a trusted private LAN. Bearer/peer authentication is optional;
TLS is disabled by default for compatibility and can be enabled with a certificate/private-key pair.
Enable authentication for LAN deployment and never expose the service publicly.

```powershell
Copy-Item configs/config.example.yaml configs/config.yaml
go run ./cmd/localbridge -config configs/config.yaml
```

The default address is `0.0.0.0:8899`. Verify it from the Windows machine:

```powershell
Invoke-RestMethod http://127.0.0.1:8899/api/v1/system/health
```

Create a redacted support bundle without starting the service:

```powershell
.\dist\localbridge.exe -support-bundle .\support.zip -config .\configs\config.yaml
```

The bundle contains only redacted configuration, runtime information and state-file metadata. It
does not contain tokens, pairing codes, device identity, local paths, TLS certificate/private-key
paths or contents, payloads, or persisted state bodies. Existing output files are never overwritten.

Then replace `WINDOWS_IP` in [`shortcut/README.md`](shortcut/README.md) and create the Push
Clipboard and Pull Clipboard Shortcuts.

## Development

```powershell
go test ./...
go vet ./...
gofmt -w cmd internal
go build ./cmd/localbridge
```

For a local development configuration, use `configs/config.dev.yaml`. A non-Windows build
starts the HTTP service but uses a no-op clipboard integration so core packages can be tested
on CI and on other platforms.

## Repository map

| Path | Responsibility |
| --- | --- |
| `cmd/localbridge` | Process entry point and signal lifecycle |
| `internal/app` | Runtime composition and shutdown |
| `internal/config` | YAML configuration and validation |
| `internal/eventbus` | Typed event names and non-blocking subscriptions |
| `internal/module` | Stable module lifecycle and route registration |
| `internal/modules/device` | Device registry, explicit pairing and peer metadata |
| `internal/transport` | Bounded authenticated HTTP JSON peer transport |
| `internal/diagnostics` | Redacted support-bundle generation |
| `internal/syncstore` | Versioned Envelope and durable sync job state |
| `internal/server` | Standard-library HTTP server and health API |
| `internal/modules/clipboard` | Clipboard API, deduplication and platform adapter |
| `internal/modules/files` | Share/receive storage, Range transfer, browser upload and QR PNG |
| `internal/modules/settings` | Versioned non-secret GUI settings store and API |
| `internal/native` | Windows startup, Explorer command, tray and completion-notification adapter |
| `internal/web` | Embedded vanilla HTML/CSS/JS GUI, no CDN |
| `docs` | Architecture, protocol, deployment and development standards |
| `shortcut` | iPhone Shortcut setup and payload examples |

## Security boundary

The file-transfer foundation uses expiring capability URLs for public mobile pages. File
management APIs are restricted to loopback unless a request has passed the configured global or
paired-peer authentication boundary. The service remains appropriate only for a trusted
home/office LAN, with the Windows firewall restricted to the private network profile. Do not
expose port 8899 or share URLs to the public internet.

## Documentation

- [Architecture](docs/architecture.md)
- [Protocol and API](docs/protocol.md)
- [Sync engine foundation](docs/sync.md)
- [Clipboard module design](docs/clipboard.md)
- [Deployment runbook](docs/deployment.md)
- [Developer guide](docs/developer-guide.md)
- [Security threat model](docs/security-threat-model.md)
- [Coding style](docs/coding-style.md)
- [Sprint 1 delivery](docs/sprints/sprint-1.md)
- [Sprint 2.1 delivery](docs/sprints/sprint-2.1.md)
- [Sprint 2.2 delivery](docs/sprints/sprint-2.2.md)
- [Sprint 2.3 delivery](docs/sprints/sprint-2.3.md)
- [Sprint 2.4 delivery](docs/sprints/sprint-2.4.md)
- [Sprint 2.5 delivery](docs/sprints/sprint-2.5.md)
- [Sprint 2.6 peer-token lifecycle](docs/sprints/sprint-2.6.md)
- [Sprint 2.7 configuration schema and diagnostics](docs/sprints/sprint-2.7.md)
- [Sprint 2.8 TLS and certificate pinning](docs/sprints/sprint-2.8.md)
- [Sprint 3.1 delivery](docs/sprints/sprint-3.1.md)
- [Sprint 4.1 file transfer](docs/sprints/sprint-4.1-file-transfer.md)
- [Sprint 4.2 desktop GUI](docs/sprints/sprint-4.2-desktop-gui.md)
- [Sprint 4.2A delivery](docs/sprints/sprint-4.2a.md)
- [Sprint 4.2B Windows shell foundation](docs/sprints/sprint-4.2b.md)
- [Sprint 4.2C receive approval](docs/sprints/sprint-4.2c.md)
- [Sprint 4.2D native Windows host](docs/sprints/sprint-4.2d.md)
- [ADR 0009 peer-token lifecycle](docs/adr/0009-peer-token-lifecycle.md)
- [ADR 0010 versioned effective configuration](docs/adr/0010-versioned-effective-configuration.md)
- [ADR 0011 TLS transport and certificate pinning](docs/adr/0011-tls-transport-and-certificate-pinning.md)
- [Product plan and capability map](docs/product-plan.md)
- [File sharing product plan](docs/file-sharing-product-plan.md)
- [Engineering workflow and agent responsibilities](docs/engineering-workflow.md)
- [Detailed roadmap](docs/roadmap.md)
- [Roadmap](ROADMAP.md)
- [AI project context prompt](docs/ai-project-context-prompt.md)

## License

LocalBridge is released under the [MIT License](LICENSE).
